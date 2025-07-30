package server

import (
	"context"
	"fmt"
	"github.com/relab/gorums"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/proto/basichotstuffpb"
	"hxy352/src/proto/clientpb"
	"hxy352/src/service"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

type ClientImpl struct {
	//Consensus      *consensus.BasicHotStuff
	Chan           chan bool
	conf           *model.Config
	nodes          []*basichotstuffpb.Node
	allNodesConfig *basichotstuffpb.Configuration
	timeout        service.TimeoutService

	NodeManager *service.NodeManager

	mutex   sync.Mutex
	resMap  map[string][]string
	doneMap map[string]struct{}

	metric *service.MetricService
}

func NewClientImpl(conf *model.Config) *ClientImpl {
	c := &ClientImpl{
		Chan:    make(chan bool),
		resMap:  make(map[string][]string),
		doneMap: make(map[string]struct{}),
		conf:    conf,
		timeout: service.NewTimeoutService(1 * time.Second),

		NodeManager: service.NewNodeManager(conf),

		metric: service.NewMetricService(conf),
	}
	c.initClient()
	c.timeout.Reset()
	c.timeout.Stop()
	return c
}

func (s *ClientImpl) SendResponse(ctx gorums.ServerCtx, res *clientpb.Response) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	cmdInt, _ := strconv.Atoi(res.Cmd)
	log.Infof("Get Response cmd: %d", cmdInt)

	if _, ok := s.doneMap[res.Cmd]; ok {
		log.Infof("Done for Cmd: %d", cmdInt)
		return
	}

	v, ok := s.resMap[res.Cmd]
	if !ok {
		s.resMap[res.Cmd] = []string{res.GetResult()}
	} else {
		v = append(v, res.GetResult())

		if len(v) < s.NodeManager.F+1 {
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
		gorums.WithSendBufferSize(1000),
		gorums.WithGrpcDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)
	// Get all all available node ids, 3 nodes

	var addrs []string

	log.Infof("Total nodes size: %d", s.NodeManager.TotalNodesNumber)

	for _, rep := range s.conf.Replica[0:s.NodeManager.TotalNodesNumber] {
		addrs = append(addrs, fmt.Sprintf("%s:%d", rep.Host, rep.Port))
	}

	var nodes []*basichotstuffpb.Node
	for {
		allNodesConfig, err := mgr.NewConfiguration(gorums.WithNodeList(addrs))
		if err != nil {
			log.Warnf("error creating read config:%v, sleep 2s", err)
			time.Sleep(2 * time.Second)
			continue
		}
		s.allNodesConfig = allNodesConfig
		nodes = allNodesConfig.Nodes()
		break
	}

	s.nodes = nodes
}

func (s *ClientImpl) SendRequests() {

	i := 1

	for {

		if i > 20 {
			// wait for metric file finished
			time.Sleep(1 * time.Second)
			os.Exit(0)
		}

		requestTime := time.Now()

		req := &basichotstuffpb.Request{
			//Cmd: strconv.Itoa(i),
			Cmd: fmt.Sprintf("%0100d", i), // 定长100
		}

		s.allNodesConfig.SendRequest(context.Background(), req)
		log.Infof("Sending request %d", i)
		s.timeout.SoftStart()

		select {
		case <-s.Chan:
			s.timeout.Stop()
			i++

			rt := time.Since(requestTime)
			go s.metric.Put(model.MetricChanInfo{
				//TraceId:     req.Cmd,
				PayloadSize: getRequestSize(req),
				Duration:    rt,
			})
			continue
		case <-s.timeout.Timeout():
			log.Warnf("Timeout received, sleep 2s, resend cmd %s !!!", req.Cmd)
			s.timeout.Stop()

			time.Sleep(2 * time.Second)
			//os.Exit(0)
		}
	}
}

func getRequestSize(msg *basichotstuffpb.Request) int {
	data, _ := proto.Marshal(msg)
	return len(data)
}
