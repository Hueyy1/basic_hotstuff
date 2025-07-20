package server

import (
	"context"
	"fmt"
	"github.com/relab/gorums"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"hxy352/src/consensus"
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/proto/basichotstuffpb"
	"hxy352/src/proto/clientpb"
	"net"
	"strconv"
	"sync"
	"time"
)

const SuccessCount = 2

type ClientImpl struct {
	Consensus *consensus.BasicHotStuff
	Chan      chan bool
	conf      *model.Config
	node      *basichotstuffpb.Node

	mutex   sync.Mutex
	resMap  map[string][]string
	doneMap map[string]struct{}
}

func NewClientImpl(conf *model.Config) *ClientImpl {
	c := &ClientImpl{
		Chan:    make(chan bool),
		resMap:  make(map[string][]string),
		doneMap: make(map[string]struct{}),
		conf:    conf,
	}
	c.initClient()
	return c
}

func (s *ClientImpl) SendResponse(ctx gorums.ServerCtx, res *clientpb.Response) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	log.Infof("Get Response: %+v", res)

	if _, ok := s.doneMap[res.Cmd]; ok {
		log.Infof("Done for Cmd: %s", res.Cmd)
		return
	}

	v, ok := s.resMap[res.Cmd]
	if !ok {
		s.resMap[res.Cmd] = []string{res.GetResult()}
	} else {
		v = append(v, res.GetResult())

		if len(v) < SuccessCount {
			// continue waiting response from replicas
			s.resMap[res.Cmd] = v
		} else {
			// get enough responses
			delete(s.resMap, res.Cmd)
			s.doneMap[res.Cmd] = struct{}{}
			s.Chan <- true
		}
	}

	return
}

func (s *ClientImpl) WaitForServerReady() {
	addr := fmt.Sprintf("%s:%d", s.conf.Client.Host, s.conf.Client.Port)
	for {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			conn.Close()
			log.Infof("client server is ready")
			break
		}
		log.Infof("Waiting for client server to be ready...")
		time.Sleep(100 * time.Millisecond)
	}
}

func (s *ClientImpl) initClient() {
	mgr := basichotstuffpb.NewManager(
		gorums.WithGrpcDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)
	// Get all all available node ids, 3 nodes

	addrs := []string{
		fmt.Sprintf("%s:%d", s.conf.Replica[0].Host, s.conf.Replica[0].Port),
	}

	var node *basichotstuffpb.Node
	for {
		allNodesConfig, err := mgr.NewConfiguration(gorums.WithNodeList(addrs))
		if err != nil {
			log.Warnf("error creating read config:%v, sleep 2s", err)
			time.Sleep(2 * time.Second)
			continue
		}
		node = allNodesConfig.Nodes()[0]
		break
	}

	s.node = node
}

func (s *ClientImpl) SendRequests() {

	i := 1

	for {
		req := &basichotstuffpb.Request{
			Cmd: strconv.Itoa(i),
		}
		s.node.SendRequest(context.Background(), req)
		log.Infof("Sending request to %v: %s", s.node.Address(), req.String())

		_ = <-s.Chan

		time.Sleep(time.Second)
		i++
	}
}
