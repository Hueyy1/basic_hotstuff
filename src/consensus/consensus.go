package consensus

import (
	"context"
	"hxy352/src/crypto"
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/proto/basichotstuffpb"
	"hxy352/src/proto/clientpb"
	"hxy352/src/proto/commonpb"
	"hxy352/src/service"
	"hxy352/src/types"
	"strconv"
	"sync"
	"time"
)

type HotStuff interface {
	PutMsg(msg *basichotstuffpb.Msg)
	PutReq(req *basichotstuffpb.Request)
	Run()

	HandleMsg()
	ProcessCurrentViewQueue(cView types.View)

	MatchingMsg(msg *basichotstuffpb.Msg, msgType commonpb.MessageType) bool
	SafeNode(block *model.Block) bool

	SendPrepare(cmd *basichotstuffpb.Request)
	OnReceivePrepare(msg *basichotstuffpb.Msg)
	OnReceivePrepareVote(msg *basichotstuffpb.Msg)
	OnReceivePreCommit(msg *basichotstuffpb.Msg)
	OnReceivePreCommitVote(msg *basichotstuffpb.Msg)
	OnReceiveCommit(msg *basichotstuffpb.Msg)
	OnReceiveCommitVote(msg *basichotstuffpb.Msg)
	OnReceiveDecide(msg *basichotstuffpb.Msg)
	SendNewView()
	OnReceiveNewView(msg *basichotstuffpb.Msg)

	SendResponse(cmd string)
}

type HotStuffImpl struct {
	Conf        *model.ReplicaConf
	gConf       *model.Config
	crypto      crypto.Crypto
	NodeManager *service.NodeManager

	CmdCache *service.CmdCache

	Ready chan struct{} // current

	MsgChan  chan *basichotstuffpb.Msg
	MsgQueue *service.MessageQueueService

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

	finishedPrepareVotes   map[types.Hash]bool
	finishedPreCommitVotes map[types.Hash]bool
	finishedCommitVotes    map[types.Hash]bool

	highQCTmp map[types.View][]types.QuorumCert
}

func NewHotStuffImpl(conf *model.ReplicaConf, gConf *model.Config) *HotStuffImpl {
	nm := service.NewNodeManager(gConf)

	return &HotStuffImpl{
		Conf:       conf,
		gConf:      gConf,
		BlockChain: service.NewBlockChain(),

		NodeManager: nm,

		CmdCache: service.NewCmdCache(),
		Ready:    make(chan struct{}),

		timeout: service.NewTimeoutService(1 * time.Second),

		MsgChan:  make(chan *basichotstuffpb.Msg, 1000),
		MsgQueue: service.NewMessageQueueService(),

		crypto: crypto.New(conf, gConf, nm.QuorumSize, nm.TimeoutQuorumSize),

		CurrentView: 1, // Initial view number

		verifiedPrepareVotes:   make(map[types.Hash][]types.PartialCert),
		verifiedPreCommitVotes: make(map[types.Hash][]types.PartialCert),
		verifiedCommitVotes:    make(map[types.Hash][]types.PartialCert),

		finishedPrepareVotes:   make(map[types.Hash]bool),
		finishedPreCommitVotes: make(map[types.Hash]bool),
		finishedCommitVotes:    make(map[types.Hash]bool),

		highQCTmp: make(map[types.View][]types.QuorumCert),

		finishedViewChange: make(map[types.View]bool),
	}
}

func (hs *HotStuffImpl) PutMsg(msg *basichotstuffpb.Msg) {
	hs.MsgChan <- msg
}

func (hs *HotStuffImpl) PutReq(req *basichotstuffpb.Request) {
	hs.CmdCache.Enqueue(req)
}

func (hs *HotStuffImpl) LockedQC() types.QuorumCert {
	return hs.PreCommitQC
}

func (hs *HotStuffImpl) GetLeaderId() types.ID {
	return hs.NodeManager.GetLeaderByView(hs.CurrentView)
}

func (hs *HotStuffImpl) GetLeaderNode() *basichotstuffpb.Node {
	return hs.NodeManager.GetLeaderNode(hs.CurrentView)
}

// CreateLeaf is called to create a new leaf block.
func (hs *HotStuffImpl) CreateLeaf(parentHash types.Hash, cert types.QuorumCert, cmd types.Command) *model.Block {
	return model.NewBlock(
		parentHash, // todo: if hs.HighQC.BlockHash() or qc, _ := cert.QC()
		cert,       // todo if highQC
		cmd,
		hs.CurrentView,
		hs.Conf.Id,
	)
}

func (hs *HotStuffImpl) VoteMyself(pc *types.PartialCert, voteType commonpb.MessageType) {
	if hs.GetLeaderId() != hs.Conf.Id {
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

func (hs *HotStuffImpl) MatchingMsg(msg *basichotstuffpb.Msg, msgType commonpb.MessageType) bool {
	return msg.GetType() == msgType && msg.GetView() == uint64(hs.CurrentView)
}

func (hs *HotStuffImpl) SafeNode(block *model.Block) bool {

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

// SendPrepare is called to propose a new block.
func (hs *HotStuffImpl) SendPrepare(cmd *basichotstuffpb.Request) {

	hs.mut.Lock()
	defer hs.mut.Unlock()

	// check if leader, otherwise send to leader
	l := hs.GetLeaderId()
	if l != hs.Conf.Id {
		log.Warnf("current replica is not leader, current leader is %d, wait... current view: %d", l, hs.CurrentView)
		return
	}

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

	hs.NodeManager.GetNodesCfg().Prepare(context.Background(), prepareMsg)

	log.Infof("SendPrepare: send prepare msg: %+v", prepareMsg)

	// vote myself

	pc, err := hs.crypto.CreatePartialCert(block)

	if err != nil {
		log.Errorf("SendPrepare: failed to create partial certificate: %v", err)
	} else {
		log.Infof("SendPrepare: vote myself: %s", hs.CurrentBlock)
		hs.VoteMyself(&pc, commonpb.MessageType_PrepareVote)
	}

	hs.timeout.SoftStart()

	return
}

// OnReceivePrepare is called when a prepare message is received.
func (hs *HotStuffImpl) OnReceivePrepare(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceivePrepare: view:%d", msg.GetView())

	if !hs.MatchingMsg(msg, commonpb.MessageType_Prepare) {
		log.Errorf("OnReceivePrepare: msg does not match")
		return
	}

	pbBlock := msg.GetBlock()
	block := basichotstuffpb.BlockFromProto(pbBlock)

	qcPb := msg.GetQC()
	if qcPb == nil {
		log.Errorf("OnReceivePrepare: partial certificate is nil")
		return
	}

	// Ensure the block is proposed by the expected leader
	leader := hs.GetLeaderId()
	if leader != block.Proposer() {
		log.Warnf("OnReceivePrepare: block was not proposed by the expected leader: %d, got: %d", leader, block.Proposer())
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

	if hs.NodeManager.IsFaultNode() {
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

	node := hs.GetLeaderNode()

	log.Debugf("leader node: %v", leader)

	node.PrepareVote(context.Background(), prepareVote)

	log.Infof("OnReceivePrepare: sent prepare vote for block: %s", block.Hash())

	hs.timeout.SoftStart()

}

// OnReceivePrepareVote is called when a prepare vote is received.
func (hs *HotStuffImpl) OnReceivePrepareVote(msg *basichotstuffpb.Msg) {
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

	if hs.finishedPrepareVotes[pc.BlockHash()] {
		log.Infof("OnReceivePrepareVote: view %d votes have finished, ignore", msg.GetView())
		return
	}

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

	if len(votes) < hs.NodeManager.QuorumSize {
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
	hs.finishedPrepareVotes[pc.BlockHash()] = true

	// send pre-commit

	hs.NodeManager.GetNodesCfg().PreCommit(context.Background(), &basichotstuffpb.Msg{
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

func (hs *HotStuffImpl) OnReceivePreCommit(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceivePreCommit: view:%d", msg.GetView())

	if !hs.MatchingMsg(msg, commonpb.MessageType_PreCommit) {
		log.Errorf("OnReceivePreCommit: msg does not match")
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
		log.Warnf("OnReceivePreCommit: invalid quorum certificate for block: %s", msg.GetType().String())
		return
	}

	// Ensure the block is proposed by the expected leader
	leader := hs.GetLeaderId()
	if leader != block.Proposer() {
		log.Warnf("OnReceivePreCommit: block was not proposed by the expected leader: %d, got: %d", leader, block.Proposer())
		return
	}

	pc, err := hs.crypto.CreatePartialCert(block)

	if err != nil {
		log.Errorf("OnReceivePreCommit: failed to create partial certificate: %.8s: %v", msg.GetBlock().Hash, err)
		return
	}

	pCert := basichotstuffpb.PartialCertToProto(pc)

	// Send preCommit vote
	if hs.NodeManager.IsFaultNode() {
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

	log.Infof("leader node sent preCommitVote: %v, view:%d, id:%d", hs.GetLeaderNode(), preCommitVote.View, preCommitVote.ReplicaId)

	hs.GetLeaderNode().PreCommitVote(context.Background(), preCommitVote)

	log.Infof("OnReceivePreCommit sent preCommit vote for block: %s", block.Hash())

	hs.timeout.SoftStart()

	hs.mut.Lock()
	defer hs.mut.Unlock()
	hs.PrepareQC = qc
	hs.CurrentBlock = block
}

// OnReceivePreCommitVote is called when a pre-commit vote is received.
func (hs *HotStuffImpl) OnReceivePreCommitVote(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceivePreCommitVote: view:%d", msg.GetView())

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

	if hs.finishedPreCommitVotes[pc.BlockHash()] {
		log.Infof("OnReceivePreCommitVote: view %d votes have finished, ignore", msg.GetView())
		return
	}

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

	if len(votes) < hs.NodeManager.QuorumSize {
		return
	}

	log.Debugf("OnReceivePreCommitVote: get vote size: %d", len(votes))

	qc, err := hs.crypto.CreateQuorumCert(hs.CurrentBlock, votes)
	if err != nil {
		log.Errorf("OnReceivePreCommitVote: could not create QC for block: %v", err)
		return
	}

	// store qc
	hs.PreCommitQC = qc

	// clean votes after create QC
	delete(hs.verifiedPreCommitVotes, pc.BlockHash())
	hs.finishedPreCommitVotes[pc.BlockHash()] = true

	// send commit

	commitMsg := &basichotstuffpb.Msg{
		Type:        commonpb.MessageType_Commit,
		View:        uint64(hs.CurrentView),
		Block:       msg.GetBlock(),
		PartialCert: nil,
		QC:          basichotstuffpb.QuorumCertToProto(qc),
		ReplicaId:   uint32(hs.Conf.Id),
	}

	hs.NodeManager.GetNodesCfg().Commit(context.Background(), commitMsg)

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
func (hs *HotStuffImpl) OnReceiveCommit(msg *basichotstuffpb.Msg) {
	log.Infof("OnReceiveCommit: view:%d", msg.GetView())

	if !hs.MatchingMsg(msg, commonpb.MessageType_Commit) {
		log.Errorf("OnReceiveCommit: msg does not match")
		return
	}

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

	// Ensure the block is proposed by the expected leader
	leader := hs.GetLeaderId()
	if leader != block.Proposer() {
		log.Warnf("OnReceiveCommit: block was not proposed by the expected leader: %d, got: %d", leader, block.Proposer())
		return
	}

	pc, err := hs.crypto.CreatePartialCert(block)

	if err != nil {
		log.Errorf("OnReceiveCommit: failed to create partial certificate: %.8s: %v", msg.GetBlock().Hash, err)
		return
	}

	// Send preCommit vote
	pCert := basichotstuffpb.PartialCertToProto(pc)

	if hs.NodeManager.IsFaultNode() {
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

func (hs *HotStuffImpl) SendResponse(cmd string) {

	res := &clientpb.Response{
		Result:    "OK",
		Cmd:       cmd,
		ReplicaId: uint32(hs.Conf.Id),
	}

	client := hs.NodeManager.GetClient()

	log.Debugf("try get client: %v", client)

	client.SendResponse(context.Background(), res)

	cmdInt, _ := strconv.Atoi(res.Cmd)

	log.Infof("Sending response: %d", cmdInt)

}
