package consensus

import (
	"bytes"
	"context"
	"fmt"
	"github.com/relab/gorums"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"hxy352/src/model"
	"hxy352/src/proto/basichotstuffpb"
	"hxy352/src/service"
	"hxy352/src/types"
	"log"
	"sync"
	"time"
)

type BasicHotStuff struct {
	Conf    *model.ReplicaConf
	gConf   *model.Config
	Timeout time.Duration

	Nodes []*basichotstuffpb.Node // All nodes in the configuration

	BlockChain model.BlockChain

	mut         sync.RWMutex // to protect the following
	CurrentView types.View
	PrepareQC   types.QuorumCert
	PreCommitQC types.QuorumCert
	CommitQC    types.QuorumCert
	HighQC      types.QuorumCert
}

func NewBasicHotStuff(conf *model.ReplicaConf, gConf *model.Config) *BasicHotStuff {
	hs := &BasicHotStuff{
		Conf:       conf,
		gConf:      gConf,
		Timeout:    1 * time.Second, // Default timeout duration
		BlockChain: service.NewBlockChain(),

		CurrentView: 1, // Initial view number
		PrepareQC:   types.NewQuorumCert(nil, 0, model.GetGenesis().Hash()),
		PreCommitQC: types.NewQuorumCert(nil, 0, model.GetGenesis().Hash()),
		CommitQC:    types.NewQuorumCert(nil, 0, model.GetGenesis().Hash()),
		HighQC:      types.NewQuorumCert(nil, 0, model.GetGenesis().Hash()),
	}

	// init all replicas except myself
	hs.initAllReplicaClients()

	return hs
}

func (hs *BasicHotStuff) initAllReplicaClients() {
	mgr := basichotstuffpb.NewManager(
		gorums.WithGrpcDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)

	var adds []string
	if hs.GetLeader() == hs.Conf.Id {
		// init 3 replicas
		for _, config := range hs.gConf.Replica {
			if types.ID(config.Id) == hs.Conf.Id {
				// Skip myself
				continue
			} else {
				adds = append(adds, fmt.Sprintf("%s:%d", config.Host, config.Port))
			}
		}

	} else {
		// init leader
		leaderIdx := int(hs.GetLeader())
		adds = append(adds, fmt.Sprintf("%s:%d", hs.gConf.Replica[leaderIdx].Host, hs.gConf.Replica[leaderIdx].Port))
	}

	// Create a configuration including all nodes
	allNodesConfig, err := mgr.NewConfiguration(gorums.WithNodeList(adds))
	if err != nil {
		log.Panic(err)
	}

	hs.Nodes = make([]*basichotstuffpb.Node, len(adds))
	hs.Nodes = allNodesConfig.Nodes()

	//	state := &basichotstuffpb.BasicMessage{
	//	Type:       basichotstuffpb.BasicMessageType_NewView,
	//	ReplicaId:  0,
	//	ViewNumber: 1,
	//}
	//
	//// Invoke Write RPC on all nodes in config
	//for _, node := range allNodesConfig.Nodes() {
	//	node.NewView(context.Background(), state)
	//}
}

func (hs *BasicHotStuff) GetLeader() types.ID {
	return types.ID(0) // Placeholder for the leader ID
}

func (hs *BasicHotStuff) NewView() {

}

// OnReceiveNewView is called when a new view message is received.
func (hs *BasicHotStuff) OnReceiveNewView() {}

// SendPrepare is called to propose a new block.
func (hs *BasicHotStuff) SendPrepare(cmd *basichotstuffpb.Request) *basichotstuffpb.Block {
	//qc, _ := cert.QC()

	block := model.NewBlock(
		hs.HighQC.BlockHash(), // todo: if hs.HighQC.BlockHash() or qc, _ := cert.QC()
		hs.HighQC,             // todo if highQC
		types.Command(cmd.Cmd),
		hs.CurrentView,
		hs.Conf.Id,
	)

	hs.BlockChain.Store(block)

	pbBlock := basichotstuffpb.BlockToProto(block)

	for _, node := range hs.Nodes {
		node.Prepare(context.Background(), pbBlock)
	}
	return pbBlock
}

// OnReceivePrepare is called when a prepare message is received.
func (hs *BasicHotStuff) OnReceivePrepare(msg *basichotstuffpb.Msg) {

	if !hs.MatchingMsg(msg, basichotstuffpb.BasicMessageType_Prepare) {
		logger.Warn("[BASIC HOTSTUFF PREPARE] msg does not match")
		return
	}

	pbBlock := msg.GetBlock()
	block := basichotstuffpb.BlockFromProto(pbBlock)

	if !bytes.Equal([]byte(block.Parent()), []byte(block.QuorumCert().BlockHash())) ||
		!hs.SafeNode(block, prepare.HighQC) {
		logger.Warn("[HOTSTUFF PREPARE] node is not correct")
		return
	}
}

// OnReceivePrepareVote is called when a prepare vote is received.
func (hs *BasicHotStuff) OnReceivePrepareVote() {}

// PreCommit is called to pre-commit a block.
func (hs *BasicHotStuff) PreCommit() {}

// OnReceivePreCommitVote is called when a pre-commit vote is received.
func (hs *BasicHotStuff) OnReceivePreCommitVote() {}

// Commit is called to commit a block.
func (hs *BasicHotStuff) Commit() {}

// OnReceiveCommitVote is called when a commit vote is received.
func (hs *BasicHotStuff) OnReceiveCommitVote() {}

// Decide is called to decide on a block.
func (hs *BasicHotStuff) Decide() {}

//func (hs *BasicHotStuff) CreateLeaf(cert types.SyncInfo, cmd types.Command) {
//
//
//
//
//	// Create a new leaf block
//	leaf := model.NewLeafBlock(hs.Conf.Id, hs.CurrentView, hs.PrepareQC, hs.PreCommitQC, hs.CommitQC)
//	hs.BlockChain.AddBlock(leaf)
//
//	// Update the current view
//	hs.CurrentView++
//	hs.PrepareQC = types.NewQuorumCert(nil, 0, leaf.Hash())
//	hs.PreCommitQC = types.NewQuorumCert(nil, 0, leaf.Hash())
//	hs.CommitQC = types.NewQuorumCert(nil, 0, leaf.Hash())
//}

func (hs *BasicHotStuff) MatchingMsg(msg *basichotstuffpb.Msg, msgType basichotstuffpb.BasicMessageType) bool {
	switch msgType {
	case basichotstuffpb.BasicMessageType_Prepare:
		return msg.GetBlock() != nil && msg.GetBlock().GetView() == uint64(hs.CurrentView)
		//case pb.MsgType_PREPARE_VOTE:
		//	return msg.GetPrepareVote() != nil && msg.GetPrepareVote().ViewNum == h.View.ViewNum
		//case pb.MsgType_PRECOMMIT:
		//	return msg.GetPreCommit() != nil && msg.GetPreCommit().ViewNum == h.View.ViewNum
		//case pb.MsgType_PRECOMMIT_VOTE:
		//	return msg.GetPreCommitVote() != nil && msg.GetPreCommitVote().ViewNum == h.View.ViewNum
		//case pb.MsgType_COMMIT:
		//	return msg.GetCommit() != nil && msg.GetCommit().ViewNum == h.View.ViewNum
		//case pb.MsgType_COMMIT_VOTE:
		//	return msg.GetCommitVote() != nil && msg.GetCommitVote().ViewNum == h.View.ViewNum
		//case pb.MsgType_NEWVIEW:
		//	return msg.GetNewView() != nil && msg.GetNewView().ViewNum == h.View.ViewNum
	}
	return false
}

func (hs *BasicHotStuff) SafeNode(block *model.Block, qc *basichotstuffpb.QuorumCert) bool {

	//return bytes.Equal([]byte(block.Parent()), []byte(hs.PreCommitQC.BlockHash())) || //safety rule
	//	qc.ViewNum > h.PreCommitQC.ViewNum // liveness rule

	return false
}
