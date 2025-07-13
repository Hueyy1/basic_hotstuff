package server

import (
	"context"
	"github.com/relab/gorums"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"hxy352/src/proto/basichotstuffpb"
	"hxy352/src/proto/commonpb"
	"log"
	"testing"
)

//func TestSendNewViewReq(t *testing.T) {
//	ca := req.GetCertificateAuthority()
//	cp := x509.NewCertPool()
//	cp.AppendCertsFromPEM(ca)
//	for _, opts := range req.GetClients() {
//		w.metricsLogger.Log(opts)
//
//		c := client.Config{
//			TLS:           opts.GetUseTLS(),
//			RootCAs:       cp,
//			MaxConcurrent: opts.GetMaxConcurrent(),
//			PayloadSize:   opts.GetPayloadSize(),
//			Input:         io.NopCloser(rand.Reader),
//			ManagerOptions: []gorums.ManagerOption{
//				gorums.WithDialTimeout(opts.GetConnectTimeout().AsDuration()),
//			},
//			RateLimit:        opts.GetRateLimit(),
//			RateStep:         opts.GetRateStep(),
//			RateStepInterval: opts.GetRateStepInterval().AsDuration(),
//			Timeout:          opts.GetTimeout().AsDuration(),
//		}
//		mods := modules.NewBuilder(hotstuff.ID(opts.GetID()), nil)
//		mods.Add(eventloop.New(1000))
//
//		if w.measurementInterval > 0 {
//			clientMetrics := metrics.GetClientMetrics(w.metrics...)
//			mods.Add(clientMetrics...)
//			mods.Add(metrics.NewTicker(w.measurementInterval))
//		}
//
//		mods.Add(w.metricsLogger)
//		mods.Add(logging.New("cli" + strconv.Itoa(int(opts.GetID()))))
//		cli := client.New(c, mods)
//		cfg, err := getConfiguration(req.GetConfiguration(), true)
//		if err != nil {
//			return nil, err
//		}
//		err = cli.Connect(cfg)
//		if err != nil {
//			return nil, err
//		}
//		cli.Start()
//		w.metricsLogger.Log(&types.StartEvent{Event: types.NewClientEvent(opts.GetID(), time.Now())})
//		w.clients[hotstuff.ID(opts.GetID())] = cli
//	}
//}

func TestNewRequest(t *testing.T) {
	mgr := basichotstuffpb.NewManager(
		gorums.WithGrpcDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)
	// Get all all available node ids, 3 nodes
	addrs := []string{
		"127.0.0.1:8001",
	}
	// Create a configuration including all nodes
	allNodesConfig, err := mgr.NewConfiguration(gorums.WithNodeList(addrs))
	if err != nil {
		log.Fatalln("error creating read config:", err)
	} // Test state
	state := &basichotstuffpb.Msg{
		Type: commonpb.MessageType_Unknown,
		Request: &basichotstuffpb.Request{
			Cmd:           "12345444444",
			ClientAddress: "127.0.0.1:9000",
		},
	}

	// Invoke Write RPC on all nodes in config
	for _, node := range allNodesConfig.Nodes() {
		node.ReceiveRequestFromClient(context.Background(), state)
	}
}
