package consensus

import (
	"context"
	"fmt"
	"github.com/relab/gorums"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"hxy352/src/crypto"
	"hxy352/src/crypto/bls12"
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/proto/basichotstuffpb"
	"hxy352/src/service"
	"hxy352/src/types"
	"net"
	"sync"
	"time"
)

type BasicHotStuff struct {
	Conf    *model.ReplicaConf
	gConf   *model.Config
	Timeout time.Duration
	crypto  crypto.Crypto

	Nodes []*basichotstuffpb.Node // All nodes in the configuration

	BlockChain model.BlockChain

	mut         sync.RWMutex // to protect the following
	CurrentView types.View
	PrepareQC   types.QuorumCert
	PreCommitQC types.QuorumCert //LockedQC
	CommitQC    types.QuorumCert
	HighQC      types.QuorumCert

	verifiedVotes map[types.Hash][]types.PartialCert
}

func NewBasicHotStuff(conf *model.ReplicaConf, gConf *model.Config) *BasicHotStuff {
	bc := service.NewBlockChain()

	hs := &BasicHotStuff{
		Conf:       conf,
		gConf:      gConf,
		Timeout:    1 * time.Second, // Default timeout duration
		BlockChain: bc,

		crypto: crypto.CryptoImpl{
			Conf:       conf,
			Bc:         bc,
			CryptoBase: bls12.New(conf),
		},

		CurrentView: 1, // Initial view number
		//PrepareQC:   types.NewQuorumCert(nil, 0, model.GetGenesis().Hash()),
		//PreCommitQC: types.NewQuorumCert(nil, 0, model.GetGenesis().Hash()),
		//CommitQC:    types.NewQuorumCert(nil, 0, model.GetGenesis().Hash()),
		//HighQC:      types.NewQuorumCert(nil, 0, model.GetGenesis().Hash()),
		//HighQC: CreateQuorumCert(),
	}
	var err error
	hs.HighQC, err = hs.crypto.CreateQuorumCert(model.GetGenesis(), []types.PartialCert{})
	if err != nil {
		log.Panicf("Failed to create initial quorum certificate: %v", err)
	}
	hs.PreCommitQC = hs.HighQC

	// init all replicas except myself
	//hs.initAllReplicaClients()

	return hs
}

func (hs *BasicHotStuff) LockedQC() types.QuorumCert {
	return hs.PreCommitQC
}

func (hs *BasicHotStuff) initAllReplicaClients() {

	// todo: find a solution to lazy load !!!!!

	sync.OnceFunc(func() {
		mgr := basichotstuffpb.NewManager(
			gorums.WithGrpcDialOptions(
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			),
		)

		var adds []string
		for _, config := range hs.gConf.Replica {
			if types.ID(config.Id) == hs.Conf.Id {
				// Skip myself
				continue
			} else {
				adds = append(adds, fmt.Sprintf("%s:%d", config.Host, config.Port))
			}
		}

		// Create a configuration including all nodes
		//log.Debugf("other replica adds: %v", adds)
		allNodesConfig, err := mgr.NewConfiguration(gorums.WithNodeList(adds))
		if err != nil {
			log.Panic(err)
		}

		hs.Nodes = make([]*basichotstuffpb.Node, len(adds))
		hs.Nodes = allNodesConfig.Nodes()

	})()

}

func (hs *BasicHotStuff) GetLeader() types.ID {
	return types.ID(0) // Placeholder for the leader ID
}

func (hs *BasicHotStuff) NewView() {

}

// OnReceiveNewView is called when a new view message is received.
func (hs *BasicHotStuff) OnReceiveNewView() {}

// CreateLeaf is called to create a new leaf block.
func (hs *BasicHotStuff) CreateLeaf(parentHash types.Hash, cert types.QuorumCert, cmd types.Command) *model.Block {
	return model.NewBlock(
		parentHash, // todo: if hs.HighQC.BlockHash() or qc, _ := cert.QC()
		cert,       // todo if highQC
		cmd,
		hs.CurrentView,
		hs.Conf.Id,
	)
}

// SendPrepare is called to propose a new block.
func (hs *BasicHotStuff) SendPrepare(cmd *basichotstuffpb.Request) *basichotstuffpb.Block {

	block := hs.CreateLeaf(hs.HighQC.BlockHash(), hs.HighQC, types.Command(cmd.GetCmd()))

	hs.BlockChain.Store(block)

	pbBlock := basichotstuffpb.BlockToProto(block)

	hs.initAllReplicaClients()

	prepareMsg := &basichotstuffpb.Msg{
		Type:  basichotstuffpb.BasicMessageType_Prepare,
		View:  uint64(hs.CurrentView),
		Block: pbBlock,
		QC:    basichotstuffpb.QuorumCertToProto(hs.HighQC),
	}

	for _, node := range hs.Nodes {
		node.Prepare(context.Background(), prepareMsg)
	}

	log.Infof("send prepare msg: %+v", prepareMsg)
	return pbBlock
}

// OnReceivePrepare is called when a prepare message is received.
func (hs *BasicHotStuff) OnReceivePrepare(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceivePrepare: %.8s", msg.GetBlock().Hash)

	if !hs.MatchingMsg(msg, basichotstuffpb.BasicMessageType_Prepare) {
		log.Errorf("[BASIC HOTSTUFF PREPARE] msg does not match: %s", msg.GetType().String())
		return
	}

	pbBlock := msg.GetBlock()
	block := basichotstuffpb.BlockFromProto(pbBlock)

	if !hs.crypto.VerifyQuorumCert(block.QuorumCert()) {
		log.Warnf("[BASIC HOTSTUFF PREPARE] invalid quorum certificate for block: %+v", block)
		return
	}

	// Ensure the block is proposed by the expected leader
	if hs.GetLeader() != block.Proposer() {
		log.Warnf("[BASIC HOTSTUFF PREPARE] block was not proposed by the expected leader: %d, got: %d", hs.GetLeader(), block.Proposer())
		return
	}

	if !hs.SafeNode(block) {
		log.Warn("[BASIC HOTSTUFF PREPARE] node is not safe")
		return
	}

	hs.BlockChain.Store(block)

	pc, err := hs.crypto.CreatePartialCert(block)

	if err != nil {
		log.Errorf("[BASIC HOTSTUFF PREPARE] failed to create partial certificate: %v", err)
		return
	}

	// Send prepare vote
	pCert := basichotstuffpb.PartialCertToProto(pc)

	prepareVote := &basichotstuffpb.Msg{
		Type:        basichotstuffpb.BasicMessageType_PrepareVote,
		View:        uint64(hs.CurrentView),
		Block:       pbBlock,
		PartialCert: pCert,
		QC:          nil,
	}

	log.Debugf("leader node: %v", hs.GetLeaderNode())

	hs.GetLeaderNode().PrepareVote(context.Background(), prepareVote)

	log.Infof("[BASIC HOTSTUFF PREPARE] sent prepare vote for block: %s", block.Hash())
}

// OnReceivePrepareVote is called when a prepare vote is received.
func (hs *BasicHotStuff) OnReceivePrepareVote(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceivePrepareVote: %.8s", msg.GetBlock().Hash)

	if !hs.MatchingMsg(msg, basichotstuffpb.BasicMessageType_PrepareVote) {
		log.Errorf("prepare vote msg does not match: %s", msg.GetType().String())
		return
	}

	pcPb := msg.GetPartialCert()
	if pcPb == nil {
		log.Errorf("prepare vote partial certificate is nil")
		return
	}

	pc := basichotstuffpb.PartialCertFromProto(pcPb)

	block, ok := hs.BlockChain.Get(pc.BlockHash())
	if !ok {
		log.Warnf("OnReceivePrepareVote: Could not find block for vote: %.8s.", pc.BlockHash())
		return
	}

	if block.View() <= hs.HighQC.View() {
		// too old
		log.Warnf("OnReceivePrepareVote: block too old: %.8s.", pc.BlockHash())
		return
	}

	if !hs.crypto.VerifyPartialCert(pc) {
		log.Info("OnReceivePrepareVote: Vote could not be verified!")
		return
	}

	hs.mut.Lock()
	defer hs.mut.Unlock()

	// store vote
	// todo: clean old votes
	// todo: or only store current view's votes?
	votes := hs.verifiedVotes[pc.BlockHash()]
	votes = append(votes, pc)
	hs.verifiedVotes[pc.BlockHash()] = votes

	if len(votes) < crypto.QuorumSize {
		return
	}

	qc, err := hs.crypto.CreateQuorumCert(block, votes)
	if err != nil {
		log.Info("OnReceivePrepareVote: could not create QC for block: ", err)
		return
	}

	// store qc
	hs.PrepareQC = qc

	// clean votes after create QC
	delete(hs.verifiedVotes, pc.BlockHash())

	// send pre-commit

	hs.initAllReplicaClients()

	for _, node := range hs.Nodes {
		node.Prepare(context.Background(), &basichotstuffpb.Msg{
			Type:        basichotstuffpb.BasicMessageType_PreCommit,
			View:        uint64(hs.CurrentView),
			Block:       msg.GetBlock(),
			PartialCert: nil,
			QC:          basichotstuffpb.QuorumCertToProto(qc),
		})
	}
}

// OnReceivePreCommitVote is called when a pre-commit vote is received.
func (hs *BasicHotStuff) OnReceivePreCommitVote(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceivePreCommitVote: %.8s", msg.GetBlock().Hash)
}

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
	return msg.GetType() == msgType && msg.GetView() == uint64(hs.CurrentView)
}

func (hs *BasicHotStuff) SafeNode(block *model.Block) bool {

	// liveness
	if block.View() > hs.LockedQC().View() {
		return true
	}

	log.Debug("[BASIC HOTSTUFF PREPARE] OnPropose: liveness condition failed")

	// safety
	lockedBlock, ok := hs.BlockChain.Get(hs.LockedQC().BlockHash())

	if !ok {
		log.Error("[BASIC HOTSTUFF PREPARE] OnPropose: failed to get locked block")
		return false
	}

	if hs.BlockChain.Extends(block, lockedBlock) {
		return true
	}

	log.Debug("[BASIC HOTSTUFF PREPARE] OnPropose: safety condition failed")

	return false
}

// GetLeaderAddress get leader address
func (hs *BasicHotStuff) GetLeaderAddress() string {
	leaderIdx := int(hs.GetLeader())
	return fmt.Sprintf("%s:%d", hs.gConf.Replica[leaderIdx].Host, hs.gConf.Replica[leaderIdx].Port)
}

func (hs *BasicHotStuff) Unicast(msg *basichotstuffpb.Msg) {
	hs.initAllReplicaClients()
	address := hs.GetLeaderAddress()
	a1, _ := normalizeAddr(address)

	for _, node := range hs.Nodes {
		a2, _ := normalizeAddr(node.Address())
		if a1 == a2 {
			node.ReceiveRequestFromClient(context.Background(), msg)
			return
		}
	}
	return
}

// GetLeaderNode get leader node
func (hs *BasicHotStuff) GetLeaderNode() *basichotstuffpb.Node {
	hs.initAllReplicaClients()
	leaderAddress := hs.GetLeaderAddress()
	a1, _ := normalizeAddr(leaderAddress)

	for _, node := range hs.Nodes {
		//log.Debugf("node.Address: %s, leaderAddress: %s", node.Address(), leaderAddress)
		a2, _ := normalizeAddr(node.Address())
		if a1 == a2 {
			return node
		}
	}
	return nil
}

func normalizeAddr(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}

	ipAddr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		return "", err
	}

	return net.JoinHostPort(ipAddr.IP.String(), port), nil
}
