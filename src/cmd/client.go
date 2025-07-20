package cmd

import (
	"fmt"
	"github.com/relab/gorums"
	"github.com/spf13/cobra"
	"hxy352/src/log"
	"hxy352/src/proto/clientpb"
	"hxy352/src/server"
	"net"
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
	srv := server.NewClientImpl(&gCfg)
	clientpb.RegisterClientServer(gorumsSrv, srv)

	go func() {
		if err := gorumsSrv.Serve(lis); err != nil {
			log.Panic(err)
		}
	}()

	// let replicas get ready...
	time.Sleep(2 * time.Second)

	srv.WaitForServerReady()

	// client start to send request to replica

	srv.SendRequests()

	return nil
}
