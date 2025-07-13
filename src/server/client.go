package server

import (
	"github.com/relab/gorums"
	"hxy352/src/consensus"
	"hxy352/src/log"
	"hxy352/src/proto/clientpb"
)

type ClientImpl struct {
	Consensus *consensus.BasicHotStuff
	Chan      chan bool
}

func NewClientImpl() *ClientImpl {
	return &ClientImpl{Chan: make(chan bool)}
}

func (s *ClientImpl) SendResponse(ctx gorums.ServerCtx, res *clientpb.Response) {
	log.Infof("Get Response: %+v", res)
	s.Chan <- true
	return
}
