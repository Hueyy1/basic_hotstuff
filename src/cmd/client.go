package cmd

import (
	"context"
	"fmt"
	"github.com/relab/gorums"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"hxy352/src/log"
	"hxy352/src/proto/basichotstuffpb"
	"hxy352/src/proto/clientpb"
	"hxy352/src/server"
	"net"
	"strconv"
	"time"
)

func newBasicHotStuffClientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "bhs-client",
		Long:        "basic hotstuff client",
		RunE:        startBasicHotStuffClient,
		Annotations: hsAnnotations,
	}
	return cmd
}

func init() {
	rootCmd.AddCommand(newBasicHotStuffClientCmd())
}

func startBasicHotStuffClient(cmd *cobra.Command, _ []string) (err error) {
	gCfg := LoadConfig()
	log.Debugf("config is %+v", gCfg)

	// start client server

	addr := fmt.Sprintf("%s:%d", gCfg.Client.Host, gCfg.Client.Port)
	log.Infof("Created Client Server at %s", addr)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panic(err)
	}
	gorumsSrv := gorums.NewServer()
	srv := server.NewClientImpl()
	clientpb.RegisterClientServer(gorumsSrv, srv)

	go func() {
		if err := gorumsSrv.Serve(lis); err != nil {
			log.Panic(err)
		}
	}()

	// let replicas get ready...
	time.Sleep(2 * time.Second)

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

	// client start to send request to replica

	mgr := basichotstuffpb.NewManager(
		gorums.WithGrpcDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)
	// Get all all available node ids, 3 nodes

	addrs := []string{
		fmt.Sprintf("%s:%d", gCfg.Replica[0].Host, gCfg.Replica[0].Port),
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

	i := 1

	for {
		req := &basichotstuffpb.Request{
			Cmd: strconv.Itoa(i),
		}
		node.SendRequest(context.Background(), req)
		log.Infof("Sending request to %v: %s", node.Address(), req.String())

		_ = <-srv.Chan

		time.Sleep(time.Second)
		i++
	}

	return nil
}
