package server

import (
	"github.com/relab/gorums"
	"hxy352/src/consensus"
	"hxy352/src/crypto"
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/proto/basichotstuffpb"
	"hxy352/src/types"
	"sync"
)

type BasicHotStuffImpl struct {
	crypto     crypto.Crypto
	blockChain model.BlockChain

	mut         sync.RWMutex // to protect the following
	highQC      types.QuorumCert
	currentView types.View

	Consensus *consensus.BasicHotStuff
}

func NewBasicHotStuffImpl(conf *model.ReplicaConf, gConf *model.Config) *BasicHotStuffImpl {
	return &BasicHotStuffImpl{
		//crypto: crypto.New(),
		Consensus: consensus.NewBasicHotStuff(conf, gConf),
	}
}

//func (s *BasicHotStuffImpl) Run(port int) {
//	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
//	if err != nil {
//		log.Panic(err)
//	}
//	gorumsSrv := gorums.NewServer()
//	srv := NewBasicHotStuffImpl()
//	basichotstuffpb.RegisterBasicHotStuffServer(gorumsSrv, srv)
//	gorumsSrv.Serve(lis)
//}

func (s *BasicHotStuffImpl) NewView(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("NewView request: %+v", msg)

	s.Consensus.OnReceiveNewView(msg)

	//v := types.View(0)
	//
	//// todo: how to wait n - f new-view messages
	//
	//if service.MsgValidate.MatchingMsg(request, basichotstuffpb.BasicMessageType_NewView, s.CurrentView()-1) == false {
	//	log.Errorf("NewView message does not match: %+v", request)
	//	return
	//}
	//
	//var qc types.QuorumCert
	//
	//if request.GetQC() == nil {
	//	log.Warnf("NewView message with nil QC: %+v", request)
	//	return
	//} else {
	//	qc = basichotstuffpb.QuorumCertFromProto(request.QC)
	//}
	//
	//if !s.crypto.VerifyQuorumCert(qc) {
	//	log.Errorf("Quorum certificate verification failed, req: %+v", request)
	//	return
	//}
	//
	//// todo: 是否使用下面的update
	//// 是否应该使用request.QC.View  > s.highQC.View()
	//if qc.View() > s.highQC.View() {
	//	s.highQC = qc
	//}
	//
	//// create proposal
	//model.NewBlock(s.highQC)

	//s.UpdateHighQC(qc)

	//if qc.View() >= v {
	//	v = qc.View()
	//}
	//
	//if v < s.CurrentView() {
	//	return
	//}
	//
	//newView := v + 1
	//s.currentView = newView
	//log.Infof("move to view %d", newView)

	return
}

//func (s *BasicHotStuffImpl) UpdateHighQC(qc types.QuorumCert) {
//
//	newBlock, ok := s.blockChain.Get(qc.BlockHash())
//	if !ok {
//		log.Info("updateHighQC: Could not find block referenced by new QC!")
//		return
//	}
//
//	if newBlock.View() > s.highQC.View() {
//		s.highQC = qc
//		log.Debug("HighQC updated")
//	}
//}
//
//// CurrentView returns the current view.
//func (s *BasicHotStuffImpl) CurrentView() types.View {
//	s.mut.RLock()
//	defer s.mut.RUnlock()
//	return s.currentView
//}
//
////func (s *BasicHotStuffImpl) CreateLeaf(parent, cmd) {
////
////}

func (s *BasicHotStuffImpl) Prepare(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("Prepare raw request: %+v", msg)

	s.Consensus.OnReceivePrepare(msg)

}

func (s *BasicHotStuffImpl) PrepareVote(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("PrepareVote raw request: %+v", msg)

	s.Consensus.OnReceivePrepareVote(msg)

}

func (s *BasicHotStuffImpl) PreCommit(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("PreCommit raw request: %+v", msg)

	s.Consensus.OnReceivePreCommit(msg)

}

func (s *BasicHotStuffImpl) PreCommitVote(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("PreCommitVote request: %+v", msg)

	s.Consensus.OnReceivePreCommitVote(msg)
}

func (s *BasicHotStuffImpl) Commit(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("Commit request: %+v", msg)

	s.Consensus.OnReceiveCommit(msg)

}

func (s *BasicHotStuffImpl) CommitVote(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("CommitVote request: %+v", msg)

	s.Consensus.OnReceiveCommitVote(msg)

}

func (s *BasicHotStuffImpl) Decide(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("Decide request: %+v", msg)

	s.Consensus.OnReceiveDecide(msg)
}

func (s *BasicHotStuffImpl) ReceiveRequestFromClient(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {

	req := msg.GetRequest()

	log.Debugf("Got request msg, content:%s", req.String())

	if req == nil {
		log.Warnf("ReceiveRequestFromClient request is nil")
		return
	}

	// check if leader, otherwise send to leader
	if s.Consensus.GetLeader() != s.Consensus.Conf.Id {
		log.Warnf("current replica is not leader, current leader is %d, resend to leader", s.Consensus.GetLeader())
		// send to leader
		s.Consensus.Unicast(msg)
		return
	}

	// todo: check if view changing, if so, return
	//if bhs.CurExec.Node != nil || bhs.View.ViewChanging {
	//	return
	//}

	//pbBlock := s.Consensus.SendPrepare(req)
	s.Consensus.SendPrepare(req)

	// vote self
	//s.Consensus.OnReceivePrepareVote(pbBlock)
}
