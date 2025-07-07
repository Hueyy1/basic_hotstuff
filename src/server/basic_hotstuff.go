package server

import (
	"fmt"
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

func (s *BasicHotStuffImpl) NewView(ctx gorums.ServerCtx, req *basichotstuffpb.SyncInfo) {
	fmt.Println("NewView")
	log.Debugf("NewView request: %+v", req)

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
	log.Debugf("Prepare request: %+v", msg)

}

func (s *BasicHotStuffImpl) PrepareVote(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("PrepareVote request: %+v", msg)

}

func (s *BasicHotStuffImpl) PreCommit(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("PreCommit request: %+v", msg)

}

func (s *BasicHotStuffImpl) PreCommitVote(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("PreCommitVote request: %+v", msg)
}

func (s *BasicHotStuffImpl) Commit(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("Commit request: %+v", msg)

}

func (s *BasicHotStuffImpl) CommitVote(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("CommitVote request: %+v", msg)

}

func (s *BasicHotStuffImpl) Decide(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("Decide request: %+v", msg)

}

func (s *BasicHotStuffImpl) ReceiveRequestFromClient(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("ReceiveRequestFromClient request: %+v", msg)

	pbBlock := s.Consensus.SendPrepare(msg.GetRequest())

	// vote self
	s.Consensus.OnReceivePrepareVote(pbBlock)
}
