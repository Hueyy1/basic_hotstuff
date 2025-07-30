package server

import (
	"github.com/relab/gorums"
	"hxy352/src/consensus"
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/proto/basichotstuffpb"
	"strconv"
)

type BasicHotStuffImpl struct {
	gConf     *model.Config
	Consensus consensus.HotStuff
}

func NewBasicHotStuffImpl(conf *model.ReplicaConf, gConf *model.Config) *BasicHotStuffImpl {
	if gConf.PacemakerLoaded {
		return &BasicHotStuffImpl{
			gConf:     gConf,
			Consensus: consensus.NewBasicHotStuffWithCogsworth(conf, gConf),
		}
	} else {
		return &BasicHotStuffImpl{
			gConf:     gConf,
			Consensus: consensus.NewBasicHotStuff(conf, gConf),
		}
	}
}

func (s *BasicHotStuffImpl) NewView(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	if !s.gConf.PacemakerLoaded {
		log.Debugf("this method is implemented for cogsworth")
		return
	}

	log.Debugf("NewView request: %+v", msg)

	s.Consensus.PutMsg(msg)

	return
}

func (s *BasicHotStuffImpl) NewViewBasic(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	if s.gConf.PacemakerLoaded {
		log.Debugf("this method is implemented for basic hotstuff")
		return
	}

	log.Debugf("NewView request: %+v", msg)

	s.Consensus.PutMsg(msg)

	return
}

func (s *BasicHotStuffImpl) Prepare(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("Prepare raw request: %+v", msg)

	s.Consensus.PutMsg(msg)

	return
}

func (s *BasicHotStuffImpl) PrepareVote(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("PrepareVote raw request: %+v", msg)

	s.Consensus.PutMsg(msg)

	return
}

func (s *BasicHotStuffImpl) PreCommit(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("PreCommit raw request: %+v", msg)

	s.Consensus.PutMsg(msg)

	return
}

func (s *BasicHotStuffImpl) PreCommitVote(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("PreCommitVote request: %+v", msg)

	s.Consensus.PutMsg(msg)

	return
}

func (s *BasicHotStuffImpl) Commit(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("Commit request: %+v", msg)

	s.Consensus.PutMsg(msg)

	return
}

func (s *BasicHotStuffImpl) CommitVote(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("CommitVote request: %+v", msg)

	s.Consensus.PutMsg(msg)

	return
}

func (s *BasicHotStuffImpl) Decide(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("Decide request: %+v", msg)

	s.Consensus.PutMsg(msg)

	return
}

func (s *BasicHotStuffImpl) SendRequest(ctx gorums.ServerCtx, req *basichotstuffpb.Request) {
	cmdInt, _ := strconv.Atoi(req.Cmd)
	log.Infof("Got request msg, content:%d", cmdInt)

	s.Consensus.PutReq(req)

	return

}

func (s *BasicHotStuffImpl) WishNextView(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("WishNextView request: %+v", msg)

	s.Consensus.PutMsg(msg)
}

func (s *BasicHotStuffImpl) Timeout(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("Timeout request: %+v", msg)

	s.Consensus.PutMsg(msg)
}

func (s *BasicHotStuffImpl) TimeoutVote(ctx gorums.ServerCtx, msg *basichotstuffpb.Msg) {
	log.Debugf("TimeoutVote request: %+v", msg)

	s.Consensus.PutMsg(msg)
}
