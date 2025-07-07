package server

import (
	"github.com/relab/gorums"
	"hxy352/src/types"
)

type Server struct {
	id        types.ID
	gorumsSrv *gorums.Server
}

//// NewServer creates a new Server.
//func NewServer() *Server {
//
//	srv := &Server{}
//	srv.gorumsSrv = gorums.NewServer()
//	basichotstuffpb.RegisterBasicHotStuffServer(srv.gorumsSrv, &serviceImpl{srv})
//	return srv
//}
