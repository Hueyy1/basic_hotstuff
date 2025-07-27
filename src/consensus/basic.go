package consensus

import (
	"context"
	"fmt"
	"github.com/relab/gorums"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"hxy352/src/crypto"
	"hxy352/src/crypto/ecdsa"
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/proto/basichotstuffpb"
	"hxy352/src/proto/clientpb"
	"hxy352/src/proto/commonpb"
	"hxy352/src/service"
	"hxy352/src/types"
	"net"
	"sync"
	"time"
)

type BasicHotStuff struct {
	Conf   *model.ReplicaConf
	gConf  *model.Config
	crypto crypto.Crypto

	CmdCache *service.CmdCache

	Ready chan struct{} // current

	MsgChan  chan *basichotstuffpb.Msg
	MsgQueue *service.MessageQueueService

	Nodes    []*basichotstuffpb.Node // All nodes in the configuration
	NodesCfg *basichotstuffpb.Configuration
	Client   *clientpb.Node

	BlockChain model.BlockChain

	timeout service.TimeoutService

	mut sync.RWMutex // to protect the following

	ViewChanging       bool
	ViewChangeCond     *sync.Cond
	finishedViewChange map[types.View]bool // fix bug: after view change finished, receive another late req

	CurrentBlock *model.Block
	CurrentView  types.View
	PrepareQC    types.QuorumCert
	PreCommitQC  types.QuorumCert //LockedQC
	CommitQC     types.QuorumCert
	HighQC       types.QuorumCert

	verifiedPrepareVotes   map[types.Hash][]types.PartialCert
	verifiedPreCommitVotes map[types.Hash][]types.PartialCert
	verifiedCommitVotes    map[types.Hash][]types.PartialCert

	highQCTmp map[types.View][]types.QuorumCert

	onceReplica sync.Once
	onceClient  sync.Once

	// metric
	//metric *service.MetricService
}

func NewBasicHotStuff(conf *model.ReplicaConf, gConf *model.Config) *BasicHotStuff {
	bc := service.NewBlockChain()

	cry := crypto.CryptoImpl{
		GConf:      gConf,
		Conf:       conf,
		CryptoBase: ecdsa.New(conf, gConf),
	}
	cry.Set_Normal_And_Fault_Size(gConf.FaultNumber)

	hs := &BasicHotStuff{
		Conf:       conf,
		gConf:      gConf,
		BlockChain: bc,

		CmdCache: service.NewCmdCache(),
		Ready:    make(chan struct{}),

		timeout: service.NewTimeoutService(1 * time.Second),

		MsgChan: make(chan *basichotstuffpb.Msg, 1000),
		//PendingMessages: make(map[types.View][]*basichotstuffpb.Msg),
		MsgQueue: service.NewMessageQueueService(),

		crypto: cry,

		CurrentView: 1, // Initial view number
		//PrepareQC:   types.NewQuorumCert(nil, 0, model.GetGenesis().Hash()),
		//PreCommitQC: types.NewQuorumCert(nil, 0, model.GetGenesis().Hash()),
		//CommitQC:    types.NewQuorumCert(nil, 0, model.GetGenesis().Hash()),
		//HighQC:      types.NewQuorumCert(nil, 0, model.GetGenesis().Hash()),
		//HighQC: CreateQuorumCert(),

		verifiedPrepareVotes:   make(map[types.Hash][]types.PartialCert),
		verifiedPreCommitVotes: make(map[types.Hash][]types.PartialCert),
		verifiedCommitVotes:    make(map[types.Hash][]types.PartialCert),

		highQCTmp: make(map[types.View][]types.QuorumCert),

		finishedViewChange: make(map[types.View]bool),

		onceReplica: sync.Once{},
		onceClient:  sync.Once{},

		//metric: service.NewMetricService(conf, gConf),
	}
	var err error
	hs.HighQC, err = hs.crypto.CreateQuorumCert(model.GetGenesis(), []types.PartialCert{})
	if err != nil {
		log.Panicf("Failed to create initial quorum certificate: %v", err)
	}
	hs.PreCommitQC = hs.HighQC

	hs.ViewChangeCond = sync.NewCond(&hs.mut)

	go func() {
		if hs.GetLeader() == hs.Conf.Id {
			hs.Ready <- struct{}{}
		}
	}()

	//go hs.metric.Handle()

	return hs
}

func (hs *BasicHotStuff) HandleMsg() {

	for {
		select {
		case msg := <-hs.MsgChan:

			//// try process current view msg
			//hs.ProcessCurrentViewQueue(hs.CurrentView)

			//hs.mut.Lock()
			cView := hs.CurrentView
			//hs.mut.Unlock()
			msgView := types.View(msg.GetView())

			if msgView > cView {
				hs.MsgQueue.Put(msg)
				log.Warnf("HandleMsg: msg(%s) %d too fast, current view: %d", msg.GetType(), msgView, cView)

				// try process current view msg
				hs.ProcessCurrentViewQueue(cView)

				continue
			} else if msgView < cView {
				log.Warnf("HandleMsg: msg(%s) %d too old, current view: %d", msg.GetType(), msgView, cView)

				// try process current view msg
				hs.ProcessCurrentViewQueue(cView)

				continue
			}

			// try process current view msg
			hs.ProcessCurrentViewQueue(cView)

			switch msg.Type {

			case commonpb.MessageType_Prepare:
				hs.OnReceivePrepare(msg)

			case commonpb.MessageType_PrepareVote:
				hs.OnReceivePrepareVote(msg)

			case commonpb.MessageType_PreCommit:
				hs.OnReceivePreCommit(msg)

			case commonpb.MessageType_PreCommitVote:
				hs.OnReceivePreCommitVote(msg)

			case commonpb.MessageType_Commit:
				hs.OnReceiveCommit(msg)

			case commonpb.MessageType_CommitVote:
				hs.OnReceiveCommitVote(msg)

			case commonpb.MessageType_Decide:
				hs.OnReceiveDecide(msg)

			case commonpb.MessageType_NewView:
				hs.OnReceiveNewView(msg)

			}

		case <-hs.timeout.Timeout():
			log.Warnf("Timeout received, go to new view !!!")

			// todo: timeout * 2
			// hs.timeout = service.NewTimeoutService()
			hs.timeout.Reset()
			hs.timeout.Stop()

			// todo: if need create empty block???
			//hs.BlockChain.Store(hs.CreateLeaf(hs.CurrentBlock.Parent(), types.QuorumCert{}, ""))

			hs.mut.Lock()

			hs.CurrentView += 1

			hs.CurrentBlock = nil

			// send new view msg
			hs.SendNewView()

			hs.mut.Unlock()

		}
	}

}

func (hs *BasicHotStuff) HandleReq() {
	for {
		select {
		case <-hs.Ready:

			for {
				req, ok := hs.CmdCache.Dequeue()
				log.Debugf("hs.CmdCache.Dequeue(): %+v", req)
				if !ok {
					continue
				}
				if hs.CmdCache.IsFinished(req.GetCmd()) {
					log.Debugf("hs.CmdCache.IsFinished(): %+v", req.GetCmd())
					continue
				}
				hs.SendPrepare(req)
				break
			}

		}
	}
}

func (hs *BasicHotStuff) ProcessCurrentViewQueue(cView types.View) {
	//hs.mut.Lock()
	//defer hs.mut.Unlock()

	lastMsg := hs.MsgQueue.Pop(cView)

	if lastMsg == nil {
		log.Debugf("ProcessCurrentViewQueue: no msg found for view %d", cView)
		return
	}

	log.Infof("ProcessCurrentViewQueue: msg(%s) found for view %d, lastMsg: %+v", lastMsg.GetType(), cView, lastMsg)

	// todo: 除了decide，猜测其他情况下可能有bug，只处理了一个vote

	switch lastMsg.GetType() {
	case commonpb.MessageType_NewView:

		// leader
		hs.OnReceiveNewView(lastMsg)

		for {
			tMsg := hs.MsgQueue.Pop(cView)
			if tMsg == nil {
				break
			}

			log.Infof("ProcessCurrentViewQueue: continue new view msg found for view %d", cView)
			hs.OnReceiveNewView(tMsg)
		}

	case commonpb.MessageType_Prepare:
		hs.OnReceivePrepare(lastMsg)
	case commonpb.MessageType_PrepareVote:
		hs.OnReceivePrepareVote(lastMsg)
	case commonpb.MessageType_PreCommit:
		hs.OnReceivePreCommit(lastMsg)
	case commonpb.MessageType_PreCommitVote:
		hs.OnReceivePreCommitVote(lastMsg)
	case commonpb.MessageType_Commit:
		hs.OnReceiveCommit(lastMsg)
	case commonpb.MessageType_CommitVote:
		hs.OnReceiveCommitVote(lastMsg)
	case commonpb.MessageType_Decide:
		hs.OnReceiveDecide(lastMsg)
		hs.ProcessCurrentViewQueue(hs.CurrentView)
	}

}

func (hs *BasicHotStuff) LockedQC() types.QuorumCert {
	return hs.PreCommitQC
}

func (hs *BasicHotStuff) InitAllReplicaClients() {

	// todo: find a solution to lazy load !!!!!

	// todo: bug, grpc retry
	// could not create configuration:
	//connection failed for addr: 127.0.0.1:8002:
	//starting stream failed: rpc error: code = Unavailable desc = connection error:
	//desc = "transport: Error while dialing: dial tcp 127.0.0.1:8002:
	//connect: can't assign requested address"

	mgr := basichotstuffpb.NewManager(
		gorums.WithGrpcDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)

	var adds []string
	for _, config := range hs.gConf.Replica[0 : crypto.FaultSize+crypto.QuorumSize] {
		if types.ID(config.Id) == hs.Conf.Id {
			// Skip myself
			continue
		} else {
			adds = append(adds, fmt.Sprintf("%s:%d", config.Host, config.Port))
		}
	}

	allNodesConfig, err := mgr.NewConfiguration(gorums.WithNodeList(adds))
	if err != nil {
		log.Panic(err)
	}

	hs.Nodes = make([]*basichotstuffpb.Node, len(adds))
	hs.Nodes = allNodesConfig.Nodes()
	hs.NodesCfg = allNodesConfig

	log.Infof("InitAllReplicaClients finished")
}

func (hs *BasicHotStuff) GetNodes() []*basichotstuffpb.Node {
	hs.onceReplica.Do(hs.InitAllReplicaClients)
	return hs.Nodes
}

func (hs *BasicHotStuff) GetNodesCfg() *basichotstuffpb.Configuration {
	hs.onceReplica.Do(hs.InitAllReplicaClients)
	return hs.NodesCfg
}

func (hs *BasicHotStuff) InitClient() {
	mgr := clientpb.NewManager(
		gorums.WithGrpcDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)

	var adds []string
	adds = append(adds, fmt.Sprintf("%s:%d", hs.gConf.Client.Host, hs.gConf.Client.Port))

	allNodesConfig, err := mgr.NewConfiguration(gorums.WithNodeList(adds))
	if err != nil {
		log.Panic(err)
	}

	hs.Client = allNodesConfig.Nodes()[0]

	log.Infof("InitClient finished")
}

func (hs *BasicHotStuff) GetClient() *clientpb.Node {
	hs.onceClient.Do(hs.InitClient)
	return hs.Client
}

func (hs *BasicHotStuff) GetLeader() types.ID {
	totalSize := crypto.FaultSize + crypto.QuorumSize
	return types.ID(int(hs.CurrentView) % totalSize)
}

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
func (hs *BasicHotStuff) SendPrepare(cmd *basichotstuffpb.Request) {

	hs.mut.Lock()
	defer hs.mut.Unlock()

	// check if leader, otherwise send to leader
	l := hs.GetLeader()
	if l != hs.Conf.Id {
		log.Warnf("current replica is not leader, current leader is %d, wait... current view: %d", l, hs.CurrentView)
		return
	}

	//hs.mut.Lock()
	//for hs.ViewChanging {
	//	// 等待 view change 完成
	//	log.Infof("SendPrepare: waiting view change finished...")
	//	hs.ViewChangeCond.Wait()
	//}
	//// 此时 view 已经稳定
	//hs.mut.Unlock()

	log.Debugf("try create leaf, %s", hs.HighQC)
	block := hs.CreateLeaf(hs.HighQC.BlockHash(), hs.HighQC, types.Command(cmd.GetCmd()))

	hs.CurrentBlock = block

	pbBlock := basichotstuffpb.BlockToProto(block)

	prepareMsg := &basichotstuffpb.Msg{
		Type:        commonpb.MessageType_Prepare,
		View:        uint64(hs.CurrentView),
		Block:       pbBlock,
		PartialCert: nil,
		QC:          basichotstuffpb.QuorumCertToProto(hs.HighQC),
		ReplicaId:   uint32(hs.Conf.Id),
	}

	hs.GetNodesCfg().Prepare(context.Background(), prepareMsg)

	log.Infof("SendPrepare: send prepare msg: %+v", prepareMsg)

	// vote myself

	pc, err := hs.crypto.CreatePartialCert(block)

	if err != nil {
		log.Errorf("SendPrepare: failed to create partial certificate: %v", err)
	} else {
		log.Infof("SendPrepare: vote myself: %s", hs.CurrentBlock)
		hs.VoteMyself(&pc, commonpb.MessageType_PrepareVote)
	}

	// metric
	//p := time.Now()
	//go hs.metric.Put(model.MetricChanInfo{
	//	Hash:        block.Hash(),
	//	View:        hs.CurrentView,
	//	ProposeTime: &p,
	//	CommitTime:  nil,
	//})

	hs.timeout.SoftStart()

	return
}

// OnReceivePrepare is called when a prepare message is received.
func (hs *BasicHotStuff) OnReceivePrepare(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceivePrepare: view:%d", msg.GetView())

	if !hs.MatchingMsg(msg, commonpb.MessageType_Prepare) {
		log.Errorf("OnReceivePrepare: msg does not match")
		return
	}

	pbBlock := msg.GetBlock()
	block := basichotstuffpb.BlockFromProto(pbBlock)

	// metric
	//p := time.Now()
	//go hs.metric.Put(model.MetricChanInfo{
	//	Hash:        block.Hash(),
	//	View:        hs.CurrentView,
	//	ProposeTime: &p,
	//	CommitTime:  nil,
	//})

	qcPb := msg.GetQC()
	if qcPb == nil {
		log.Errorf("OnReceivePrepare: partial certificate is nil")
		return
	}

	// todo: bug, can't verify
	//if !hs.crypto.VerifyQuorumCert(block, basichotstuffpb.QuorumCertFromProto(qcPb)) {
	//	log.Warnf("OnReceivePrepare: invalid quorum certificate for block: %+v", block)
	//	return
	//}

	// Ensure the block is proposed by the expected leader
	if hs.GetLeader() != block.Proposer() {
		log.Warnf("OnReceivePrepare: block was not proposed by the expected leader: %d, got: %d", hs.GetLeader(), block.Proposer())
		return
	}

	if !hs.SafeNode(block) {
		log.Warn("OnReceivePrepare: node is not safe")
		return
	}

	hs.CurrentBlock = block

	pc, err := hs.crypto.CreatePartialCert(block)

	if err != nil {
		log.Errorf("OnReceivePrepare: failed to create partial certificate: %v", err)
		return
	}

	// Send prepare vote
	pCert := basichotstuffpb.PartialCertToProto(pc)

	if crypto.IsFaultNode {
		pCert = nil
	}

	prepareVote := &basichotstuffpb.Msg{
		Type:        commonpb.MessageType_PrepareVote,
		View:        uint64(hs.CurrentView),
		Block:       pbBlock,
		PartialCert: pCert,
		QC:          nil,
		ReplicaId:   uint32(hs.Conf.Id),
	}

	log.Debugf("leader node: %v", hs.GetLeaderNode())

	hs.GetLeaderNode().PrepareVote(context.Background(), prepareVote)

	log.Infof("OnReceivePrepare: sent prepare vote for block: %s", block.Hash())

	hs.timeout.SoftStart()

}

// OnReceivePrepareVote is called when a prepare vote is received.
func (hs *BasicHotStuff) OnReceivePrepareVote(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceivePrepareVote: view:%d", msg.GetView())

	if !hs.MatchingMsg(msg, commonpb.MessageType_PrepareVote) {
		log.Errorf("prepare vote msg does not match")
		return
	}

	pcPb := msg.GetPartialCert()
	if pcPb == nil {
		log.Errorf("OnReceivePrepareVote: partial certificate is nil")
		return
	}

	pc := basichotstuffpb.PartialCertFromProto(pcPb)
	block := basichotstuffpb.BlockFromProto(msg.GetBlock())

	//if hs.CurrentBlock.Hash() != pc.BlockHash() {
	//	log.Warnf("OnReceivePrepareVote: currentBlock.Hash() != pc.BlockHash(): %.8s.", pc.BlockHash())
	//	return
	//}

	//if hs.CurrentBlock.View() <= hs.HighQC.View() {
	//	// too old
	//	log.Warnf("OnReceivePrepareVote: block too old: %.8s.", pc.BlockHash())
	//	return
	//}

	if !hs.crypto.VerifyPartialCert(block, pc) {
		log.Info("OnReceivePrepareVote: Vote could not be verified!")
		return
	}

	log.Debugf("OnReceivePrepareVote: Vote PC verified: %.8s", pc.BlockHash())

	hs.mut.Lock()
	defer hs.mut.Unlock()

	hs.CurrentBlock = block

	// store vote
	votes := hs.verifiedPrepareVotes[pc.BlockHash()]
	votes = append(votes, pc)
	hs.verifiedPrepareVotes[pc.BlockHash()] = votes

	if len(votes) < crypto.QuorumSize {
		return
	}

	// todo: bug
	// if we have 3 pc and we finished to create qc, but got 4th pc

	log.Debugf("OnReceivePrepareVote: get vote size: %d", len(votes))

	qc, err := hs.crypto.CreateQuorumCert(hs.CurrentBlock, votes)
	if err != nil {
		log.Info("OnReceivePrepareVote: could not create QC for block: ", err)
		return
	}

	// store qc
	hs.PrepareQC = qc

	// clean votes after create QC
	delete(hs.verifiedPrepareVotes, pc.BlockHash())

	// send pre-commit

	hs.NodesCfg.PreCommit(context.Background(), &basichotstuffpb.Msg{
		Type:        commonpb.MessageType_PreCommit,
		View:        uint64(hs.CurrentView),
		Block:       msg.GetBlock(),
		PartialCert: nil,
		QC:          basichotstuffpb.QuorumCertToProto(qc),
		ReplicaId:   uint32(hs.Conf.Id),
	})

	// vote myself

	preCommitPC, err := hs.crypto.CreatePartialCert(block)

	if err != nil {
		log.Errorf("OnReceivePrepareVote:: failed to create partial certificate: %v", err)
	} else {
		hs.VoteMyself(&preCommitPC, commonpb.MessageType_PreCommitVote)
	}

	hs.timeout.SoftStart()

}

func (hs *BasicHotStuff) OnReceivePreCommit(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceivePreCommit: view:%d", msg.GetView())

	if !hs.MatchingMsg(msg, commonpb.MessageType_PreCommit) {
		log.Errorf("OnReceivePreCommit: msg does not match")
		return
	}

	//if msg.GetView() < uint64(hs.CurrentView) {
	//	log.Warnf("OnReceivePreCommit: too old msg")
	//	return
	//}

	//pbBlock := msg.GetBlock()
	//block := basichotstuffpb.BlockFromProto(pbBlock)

	qcPb := msg.GetQC()
	if qcPb == nil {
		log.Errorf("OnReceivePreCommit: could not find QC")
		return
	}

	qc := basichotstuffpb.QuorumCertFromProto(qcPb)
	block := basichotstuffpb.BlockFromProto(msg.GetBlock())

	if !hs.crypto.VerifyQuorumCert(block, qc) {
		log.Warnf("OnReceivePreCommit: invalid quorum certificate for block: %s", msg.GetType().String())
		return
	}

	//if hs.CurrentBlock.Hash() != qc.BlockHash() {
	//	log.Warnf("OnReceivePreCommit: CurrentBlock.Hash() != qc.BlockHash(): %.8s.", qc.BlockHash())
	//	return
	//}

	// Ensure the block is proposed by the expected leader
	if hs.GetLeader() != block.Proposer() {
		log.Warnf("OnReceivePreCommit: block was not proposed by the expected leader: %d, got: %d", hs.GetLeader(), block.Proposer())
		return
	}

	pc, err := hs.crypto.CreatePartialCert(block)

	if err != nil {
		log.Errorf("OnReceivePreCommit: failed to create partial certificate: %.8s: %v", msg.GetBlock().Hash, err)
		return
	}

	pCert := basichotstuffpb.PartialCertToProto(pc)

	// Send preCommit vote
	if crypto.IsFaultNode {
		pCert = nil
	}

	preCommitVote := &basichotstuffpb.Msg{
		Type:        commonpb.MessageType_PreCommitVote,
		View:        uint64(hs.CurrentView),
		Block:       msg.GetBlock(),
		PartialCert: pCert,
		QC:          nil,
		ReplicaId:   uint32(hs.Conf.Id),
	}

	//log.Debugf("leader node: %v", hs.GetLeaderNode())

	hs.GetLeaderNode().PreCommitVote(context.Background(), preCommitVote)

	log.Infof("OnReceivePreCommit sent preCommit vote for block: %s", block.Hash())

	hs.timeout.SoftStart()

	hs.mut.Lock()
	defer hs.mut.Unlock()
	hs.PrepareQC = qc
	hs.CurrentBlock = block
}

// OnReceivePreCommitVote is called when a pre-commit vote is received.
func (hs *BasicHotStuff) OnReceivePreCommitVote(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceivePreCommitVote: view:%d", msg.GetView())

	//if msg.GetView() < uint64(hs.CurrentView) {
	//	log.Warnf("OnReceivePreCommitVote: vote from view %d is too low, current view has moved to %d ", msg.GetView(), hs.CurrentView)
	//	return
	//}

	if !hs.MatchingMsg(msg, commonpb.MessageType_PreCommitVote) {
		log.Panicf("OnReceivePreCommitVote: preCommit vote msg does not match, msg.GetType %s; msg.GetView() %d; hs.CurrentView %d", msg.GetType(), msg.GetView(), hs.CurrentView)
		return
	}

	pcPb := msg.GetPartialCert()
	if pcPb == nil {
		log.Errorf("OnReceivePreCommitVote: partial certificate is nil")
		return
	}

	pc := basichotstuffpb.PartialCertFromProto(pcPb)
	block := basichotstuffpb.BlockFromProto(msg.GetBlock())

	//if hs.CurrentBlock.Hash() != pc.BlockHash() {
	//	log.Warnf("OnReceivePreCommitVote: CurrentBlock.Hash() != pc.BlockHash(): %.8s.", pc.BlockHash())
	//	return
	//}

	//if hs.CurrentBlock.View() <= hs.HighQC.View() {
	//	// too old
	//	log.Warnf("OnReceivePreCommitVote: block too old: %.8s.", pc.BlockHash())
	//	return
	//}

	if !hs.crypto.VerifyPartialCert(block, pc) {
		log.Info("OnReceivePreCommitVote: Vote could not be verified!")
		return
	}

	log.Debugf("OnReceivePreCommitVote: Vote PC verified: %.8s", pc.BlockHash())

	hs.mut.Lock()
	defer hs.mut.Unlock()

	hs.CurrentBlock = block

	// store vote
	votes := hs.verifiedPreCommitVotes[pc.BlockHash()]
	votes = append(votes, pc)
	hs.verifiedPreCommitVotes[pc.BlockHash()] = votes

	if len(votes) < crypto.QuorumSize {
		return
	}

	log.Debugf("OnReceivePreCommitVote: get vote size: %d", len(votes))

	qc, err := hs.crypto.CreateQuorumCert(hs.CurrentBlock, votes)
	if err != nil {
		log.Info("OnReceivePreCommitVote: could not create QC for block: ", err)
		return
	}

	// store qc
	hs.PreCommitQC = qc

	// clean votes after create QC
	delete(hs.verifiedPreCommitVotes, pc.BlockHash())

	// send pre-commit

	commitMsg := &basichotstuffpb.Msg{
		Type:        commonpb.MessageType_Commit,
		View:        uint64(hs.CurrentView),
		Block:       msg.GetBlock(),
		PartialCert: nil,
		QC:          basichotstuffpb.QuorumCertToProto(qc),
		ReplicaId:   uint32(hs.Conf.Id),
	}

	hs.GetNodesCfg().Commit(context.Background(), commitMsg)

	// vote myself

	commitPC, err := hs.crypto.CreatePartialCert(block)

	if err != nil {
		log.Errorf("OnReceivePreCommitVote: failed to create partial certificate: %v", err)
	} else {
		hs.VoteMyself(&commitPC, commonpb.MessageType_CommitVote)
	}

	hs.timeout.SoftStart()
}

// OnReceiveCommit is called to commit a block.
func (hs *BasicHotStuff) OnReceiveCommit(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceiveCommit: view:%d", msg.GetView())

	if !hs.MatchingMsg(msg, commonpb.MessageType_Commit) {
		log.Errorf("OnReceiveCommit: msg does not match")
		return
	}

	//pbBlock := msg.GetBlock()
	//block := basichotstuffpb.BlockFromProto(pbBlock)

	qcPb := msg.GetQC()
	if qcPb == nil {
		log.Errorf("OnReceiveCommit: could not find QC")
		return
	}

	qc := basichotstuffpb.QuorumCertFromProto(qcPb)
	block := basichotstuffpb.BlockFromProto(msg.GetBlock())

	if !hs.crypto.VerifyQuorumCert(block, qc) {
		log.Warnf("OnReceiveCommit: invalid quorum certificate for block: %s", msg.GetType().String())
		return
	}

	//if hs.CurrentBlock.Hash() != qc.BlockHash() {
	//	log.Warnf("OnReceiveCommit: Could not find block for vote: %.8s.", qc.BlockHash())
	//	return
	//}

	// Ensure the block is proposed by the expected leader
	if hs.GetLeader() != block.Proposer() {
		log.Warnf("OnReceiveCommit: block was not proposed by the expected leader: %d, got: %d", hs.GetLeader(), block.Proposer())
		return
	}

	pc, err := hs.crypto.CreatePartialCert(block)

	if err != nil {
		log.Errorf("OnReceiveCommit: failed to create partial certificate: %.8s: %v", msg.GetBlock().Hash, err)
		return
	}

	// Send preCommit vote
	pCert := basichotstuffpb.PartialCertToProto(pc)

	if crypto.IsFaultNode {
		pCert = nil
	}

	commitVote := &basichotstuffpb.Msg{
		Type:        commonpb.MessageType_CommitVote,
		View:        uint64(hs.CurrentView),
		Block:       msg.GetBlock(),
		PartialCert: pCert,
		QC:          nil,
		ReplicaId:   uint32(hs.Conf.Id),
	}

	//log.Debugf("leader node: %v", hs.GetLeaderNode())

	hs.GetLeaderNode().CommitVote(context.Background(), commitVote)

	log.Infof("OnReceiveCommit sent commit vote for block: %s", block.Hash())

	hs.timeout.SoftStart()

	hs.mut.Lock()
	defer hs.mut.Unlock()
	hs.CurrentBlock = block
	hs.PreCommitQC = qc

}

// OnReceiveCommitVote is called when a pre-commit vote is received.
func (hs *BasicHotStuff) OnReceiveCommitVote(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceiveCommitVote: view:%d", msg.GetView())

	//// leader has already moved to next view
	//if msg.GetView() < uint64(hs.CurrentView) {
	//	log.Warnf("OnReceiveCommitVote: vote from view %d is too low, current view has moved to %d ", msg.GetView(), hs.CurrentView)
	//	return
	//}

	if !hs.MatchingMsg(msg, commonpb.MessageType_CommitVote) {
		log.Errorf("OnReceiveCommitVote: Commit vote msg does not match")
		return
	}

	pcPb := msg.GetPartialCert()
	if pcPb == nil {
		log.Errorf("OnReceiveCommitVote: partial certificate is nil")
		return
	}

	pc := basichotstuffpb.PartialCertFromProto(pcPb)
	block := basichotstuffpb.BlockFromProto(msg.GetBlock())

	//if hs.CurrentBlock.Hash() != pc.BlockHash() {
	//	log.Warnf("OnReceiveCommitVote: Could not find block for vote: %.8s.", pc.BlockHash())
	//	return
	//}

	//if hs.CurrentBlock.View() <= hs.HighQC.View() {
	//	// too old
	//	log.Warnf("OnReceiveCommitVote: block too old: %.8s.", pc.BlockHash())
	//	return
	//}

	if !hs.crypto.VerifyPartialCert(block, pc) {
		log.Info("OnReceiveCommitVote: Vote could not be verified!")
		return
	}

	log.Debugf("OnReceiveCommitVote: Vote PC verified: %.8s", pc.BlockHash())

	hs.mut.Lock()
	defer hs.mut.Unlock()

	hs.CurrentBlock = block

	// store vote
	votes := hs.verifiedCommitVotes[pc.BlockHash()]
	votes = append(votes, pc)
	hs.verifiedCommitVotes[pc.BlockHash()] = votes

	if len(votes) < crypto.QuorumSize {
		//hs.mut.Unlock()
		return
	}

	log.Debugf("OnReceiveCommitVote: get vote size: %d", len(votes))

	qc, err := hs.crypto.CreateQuorumCert(block, votes)
	if err != nil {
		log.Info("OnReceiveCommitVote: could not create QC for block: ", err)
		//hs.mut.Unlock()
		return
	}

	// store qc
	hs.CommitQC = qc

	// clean votes after create QC
	delete(hs.verifiedCommitVotes, pc.BlockHash())

	// store block
	hs.BlockChain.Store(block)

	// clean block
	hs.BlockChain.Clean(block)

	// send decide

	decideMsg := &basichotstuffpb.Msg{
		Type:        commonpb.MessageType_Decide,
		View:        uint64(hs.CurrentView),
		Block:       msg.GetBlock(),
		PartialCert: nil,
		QC:          basichotstuffpb.QuorumCertToProto(qc),
		ReplicaId:   uint32(hs.Conf.Id),
	}

	hs.GetNodesCfg().Decide(context.Background(), decideMsg)

	// exec cmd
	log.Infof("OnReceiveCommitVote: exec cmd: %s %s", msg.GetBlock().Hash, block.Command())

	// metric
	//p := time.Now()
	//go hs.metric.Put(model.MetricChanInfo{
	//	Hash:        hs.CurrentBlock.Hash(),
	//	View:        hs.CurrentView,
	//	ProposeTime: nil,
	//	CommitTime:  &p,
	//})

	// view number + 1
	hs.CurrentView += 1

	// reset currentBlock
	cmd := string(hs.CurrentBlock.Command())
	hs.CurrentBlock = nil
	hs.CmdCache.Finish(cmd)

	//hs.mut.Unlock()

	//hs.mut.Lock()
	// send new view
	hs.SendNewView()
	//hs.mut.Unlock()

	// todo: if
	// send response after new-view
	hs.SendResponse(cmd)

	hs.timeout.Stop()

}

// OnReceiveDecide is called to decide on a block.
func (hs *BasicHotStuff) OnReceiveDecide(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceiveDecide: view:%d", msg.GetView())

	if !hs.MatchingMsg(msg, commonpb.MessageType_Decide) {
		log.Errorf("OnReceiveDecide: msg does not match")
		return
	}

	qcPb := msg.GetQC()
	if qcPb == nil {
		log.Errorf("OnReceivePreCommit: could not find QC")
		return
	}

	qc := basichotstuffpb.QuorumCertFromProto(qcPb)
	block := basichotstuffpb.BlockFromProto(msg.GetBlock())

	if !hs.crypto.VerifyQuorumCert(block, qc) {
		log.Warnf("OnReceiveDecide: invalid quorum certificate for block: %s", msg.GetType().String())
		return
	}

	//if hs.CurrentBlock.Hash() != qc.BlockHash() {
	//	log.Warnf("OnReceiveDecide: Could not find block for vote: %.8s.", qc.BlockHash())
	//	return
	//}

	// Ensure the block is proposed by the expected leader
	if hs.GetLeader() != block.Proposer() {
		log.Warnf("OnReceiveDecide: block was not proposed by the expected leader: %d, got: %d", hs.GetLeader(), block.Proposer())
		return
	}

	// exec cmd
	log.Infof("OnReceiveDecide: exec cmd: %s %s", msg.GetBlock().Hash, block.Command())

	hs.CmdCache.Finish(string(block.Command()))

	// store block
	hs.BlockChain.Store(block)

	// clean block
	hs.BlockChain.Clean(block)

	// metric
	//p := time.Now()
	//go hs.metric.Put(model.MetricChanInfo{
	//	Hash:        block.Hash(),
	//	View:        hs.CurrentView,
	//	ProposeTime: nil,
	//	CommitTime:  &p,
	//})

	hs.mut.Lock()
	defer hs.mut.Unlock()

	// view number + 1
	hs.CurrentView = types.View(msg.View + 1)
	log.Infof("OnReceiveDecide: set new view %d", hs.CurrentView)

	// reset block
	cmd := string(block.Command())
	hs.CurrentBlock = nil

	// leader start change new view
	if hs.GetLeader() == hs.Conf.Id {
		hs.ViewChanging = true
	}

	// todo: update highqc ? is it right?
	hs.HighQC = qc
	hs.CommitQC = qc

	// send new view to next leader
	hs.SendNewView()

	// send response
	hs.SendResponse(cmd)

	hs.timeout.Stop()

}

func (hs *BasicHotStuff) SendNewView() {
	// send new view to next leader

	l := hs.GetLeader()
	if l == hs.Conf.Id {
		// new view to myself

		//hs.mut.Lock()

		//hs.highQCTmp = append(hs.highQCTmp, hs.PrepareQC)
		hs.highQCTmp[hs.CurrentView] = append(hs.highQCTmp[hs.CurrentView], hs.PrepareQC)

		hs.ViewChanging = true

		hs.ProcessCurrentViewQueue(hs.CurrentView)
		//hs.ProcessCurrentViewQueue(hs.CurrentView + 1)

		length := len(hs.highQCTmp[hs.CurrentView])
		//log.Infof("highQCTmp: %v", hs.highQCTmp[hs.CurrentView])
		log.Infof("SendNewView: send new view %d to myself, View Changing... get %d qc", hs.CurrentView, length)

		if length < crypto.QuorumSize {
			return
		}

		hs.processNewView()

		//hs.ProcessQueue(hs.CurrentView)

		//hs.mut.Unlock()
		return
	}

	log.Infof("OnSendNewView: sending new view to leader: %d", l)

	hs.GetLeaderNode().NewView(context.Background(), &basichotstuffpb.Msg{
		Type:        commonpb.MessageType_NewView,
		View:        uint64(hs.CurrentView),
		Block:       nil,
		PartialCert: nil,
		QC:          basichotstuffpb.QuorumCertToProto(hs.PrepareQC),
	})
	//hs.Ready <- struct{}{}
}

// OnReceiveNewView is called when a new view message is received.
func (hs *BasicHotStuff) OnReceiveNewView(msg *basichotstuffpb.Msg) {
	//log.Errorf("OnReceiveNewView: error receive new view: %d, current: %d", msg.GetView(), hs.CurrentView)

	//hs.mut.Lock()
	//defer hs.mut.Unlock()

	//if msg.GetView() < uint64(hs.CurrentView) {
	//	log.Warnf("OnReceiveNewView: received old new view(%d), current(%d), ignore", msg.GetView(), hs.CurrentView)
	//	return
	//}

	//// todo: bug: a new view msg will be processed before decide msg.
	//if msg.GetView() > uint64(hs.CurrentView) {
	//	log.Infof("OnReceiveNewView: received new view (%d) before current decide (%d), put it into pending...", msg.GetView(), hs.CurrentView)
	//	hs.PendingMessages[types.View(msg.GetView())] = append(hs.PendingMessages[types.View(msg.GetView())], msg)
	//	return
	//}

	//// process pending new views
	//if data, ok := hs.PendingMessages[types.View(msg.GetView())]; ok {
	//	for i, pMsg := range data {
	//
	//		log.Infof("OnReceiveNewView: process %d pending msg...", i)
	//
	//		qcPb := pMsg.GetQC()
	//		if qcPb == nil {
	//			log.Errorf("OnReceiveNewView: could not find QC")
	//			continue
	//		}
	//
	//		qc := basichotstuffpb.QuorumCertFromProto(qcPb)
	//
	//		hs.highQCTmp[hs.CurrentView] = append(hs.highQCTmp[hs.CurrentView], qc)
	//	}
	//}

	log.Infof("OnReceiveNewView: receive new view: %d", msg.GetView())

	//if msg.GetView() < uint64(hs.CurrentView) {
	//	log.Warnf("OnReceivePreCommitVote: vote from view %d is too low, current view has moved to %d ", msg.GetView(), hs.CurrentView)
	//	return
	//}

	if hs.finishedViewChange[hs.CurrentView] == true {
		log.Infof("OnReceiveNewView: already finished view change: %d", msg.GetView())
		return
	}

	qcPb := msg.GetQC()
	if qcPb == nil {
		log.Errorf("OnReceiveNewView: could not find QC")
		return
	}

	qc := basichotstuffpb.QuorumCertFromProto(qcPb)

	t := hs.highQCTmp[hs.CurrentView]
	t = append(t, qc)
	hs.highQCTmp[hs.CurrentView] = t

	hs.ViewChanging = true
	//log.Infof("highQCTmp: %v", hs.highQCTmp[hs.CurrentView])
	log.Infof("OnReceiveNewView: View Changing... get %d qc", len(hs.highQCTmp[hs.CurrentView]))

	if len(hs.highQCTmp[hs.CurrentView]) < crypto.QuorumSize {
		return
	}

	hs.processNewView()
}

func (hs *BasicHotStuff) processNewView() {

	maxQC := hs.PrepareQC

	for _, tmp := range hs.highQCTmp[hs.CurrentView] {
		if tmp.View() > maxQC.View() {
			maxQC = tmp
		}
	}

	hs.HighQC = maxQC

	// clean hs.highQCTmp
	delete(hs.highQCTmp, hs.CurrentView)

	// clean votes
	cleanFunc := func(m map[types.Hash][]types.PartialCert) {
		for k := range m {
			delete(m, k)
		}
	}
	cleanFunc(hs.verifiedPrepareVotes)
	cleanFunc(hs.verifiedPreCommitVotes)
	cleanFunc(hs.verifiedCommitVotes)
	log.Debug("processNewView: clean all votes")

	hs.finishedViewChange[hs.CurrentView] = true

	hs.ViewChanging = false
	log.Infof("processNewView: new view %d finished, next view len: %d", hs.CurrentView, len(hs.MsgQueue.Get(hs.CurrentView+1)))

	// 通知handleReq
	hs.Ready <- struct{}{}

	//// try to chase new msg
	//cView := hs.CurrentView
	//hs.ProcessQueue(cView)
	//
	//if hs.MsgQueue.Get(cView+1) != nil && len(hs.MsgQueue.Get(cView+1)) != 0 {
	//	// continue process Queue
	//	log.Infof("processNewView: continue processNewView: %d", cView+1)
	//	hs.ProcessQueue(cView + 1)
	//} else {
	//	log.Infof("processNewView: broadcast view %d change finished", cView)
	//	hs.ViewChangeCond.Broadcast() // 唤醒等待请求
	//}

}
func (hs *BasicHotStuff) MatchingMsg(msg *basichotstuffpb.Msg, msgType commonpb.MessageType) bool {
	return msg.GetType() == msgType && msg.GetView() == uint64(hs.CurrentView)
}

func (hs *BasicHotStuff) SafeNode(block *model.Block) bool {

	// liveness
	if block.View() > hs.LockedQC().View() {
		return true
	}

	log.Debug("liveness condition failed")

	// safety
	lockedBlock, ok := hs.BlockChain.Get(hs.LockedQC().BlockHash())

	if !ok {
		log.Error("failed to get locked block")
		return false
	}

	if hs.BlockChain.Extends(block, lockedBlock) {
		return true
	}

	log.Debug("safety condition failed")

	return false
}

// GetLeaderAddress get leader address
func (hs *BasicHotStuff) GetLeaderAddress() string {
	leaderIdx := int(hs.GetLeader())
	return fmt.Sprintf("%s:%d", hs.gConf.Replica[leaderIdx].Host, hs.gConf.Replica[leaderIdx].Port)
}

// GetLeaderNode get leader node
func (hs *BasicHotStuff) GetLeaderNode() *basichotstuffpb.Node {
	leaderAddress := hs.GetLeaderAddress()
	a1, _ := normalizeAddr(leaderAddress)

	for _, node := range hs.GetNodes() {
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

func (hs *BasicHotStuff) VoteMyself(pc *types.PartialCert, voteType commonpb.MessageType) {
	if hs.GetLeader() != hs.Conf.Id {
		return
	}

	//hs.mut.Lock()
	//defer hs.mut.Unlock()

	switch voteType {
	case commonpb.MessageType_PrepareVote:
		votes := hs.verifiedPrepareVotes[pc.BlockHash()]
		votes = append(votes, *pc)
		hs.verifiedPrepareVotes[pc.BlockHash()] = votes
	case commonpb.MessageType_PreCommitVote:
		votes := hs.verifiedPreCommitVotes[pc.BlockHash()]
		votes = append(votes, *pc)
		hs.verifiedPreCommitVotes[pc.BlockHash()] = votes
	case commonpb.MessageType_CommitVote:
		votes := hs.verifiedCommitVotes[pc.BlockHash()]
		votes = append(votes, *pc)
		hs.verifiedCommitVotes[pc.BlockHash()] = votes
	default:
		log.Warnf("PrepareVoteMyself: unknown vote type: %v", voteType)
	}
}

func (hs *BasicHotStuff) SendResponse(cmd string) {

	res := &clientpb.Response{
		Result:    "OK",
		Cmd:       cmd,
		ReplicaId: uint32(hs.Conf.Id),
	}

	log.Infof("try get client: %v", hs.GetClient())

	hs.GetClient().SendResponse(context.Background(), res)

	log.Infof("Sending response: %s", res.String())

}
