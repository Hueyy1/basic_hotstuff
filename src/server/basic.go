package server

import (
	"github.com/relab/gorums"
	"hxy352/src/consensus"
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/proto/basichotstuffpb"
)

type BasicHotStuffImpl struct {
	Consensus *consensus.BasicHotStuff
}

func NewBasicHotStuffImpl(conf *model.ReplicaConf, gConf *model.Config) *BasicHotStuffImpl {
	return &BasicHotStuffImpl{
		Consensus: consensus.NewBasicHotStuff(conf, gConf),
	}
}

func (s *BasicHotStuffImpl) NewView(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("NewView request: %+v", msg)

	s.Consensus.MsgChan <- msg

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

	s.Consensus.MsgChan <- msg

	return
}

func (s *BasicHotStuffImpl) PrepareVote(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("PrepareVote raw request: %+v", msg)

	s.Consensus.MsgChan <- msg

	return
}

func (s *BasicHotStuffImpl) PreCommit(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("PreCommit raw request: %+v", msg)

	s.Consensus.MsgChan <- msg

	return
}

func (s *BasicHotStuffImpl) PreCommitVote(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("PreCommitVote request: %+v", msg)

	s.Consensus.MsgChan <- msg

	return
}

func (s *BasicHotStuffImpl) Commit(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("Commit request: %+v", msg)

	s.Consensus.MsgChan <- msg

	return
}

func (s *BasicHotStuffImpl) CommitVote(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("CommitVote request: %+v", msg)

	s.Consensus.MsgChan <- msg

	return
}

func (s *BasicHotStuffImpl) Decide(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("Decide request: %+v", msg)

	s.Consensus.MsgChan <- msg

	return
}

func (s *BasicHotStuffImpl) SendRequest(ctx gorums.ServerCtx, req *basichotstuffpb.Request) {
	log.Infof("Got request msg, content:%s", req.String())

	if req == nil {
		log.Warnf("ReceiveRequestFromClient request is nil")
		return
	}

	s.Consensus.CmdCache.Enqueue(req)

	return

}

func (s *BasicHotStuffImpl) WishNextView(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("WishNextView request: %+v", msg)

	s.Consensus.MsgChan <- msg
}

func (s *BasicHotStuffImpl) Timeout(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("Timeout request: %+v", msg)

	s.Consensus.MsgChan <- msg
}

func (s *BasicHotStuffImpl) TimeoutVote(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("TimeoutVote request: %+v", msg)

	s.Consensus.MsgChan <- msg
}
