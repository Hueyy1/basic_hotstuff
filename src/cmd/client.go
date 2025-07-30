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
	var faultNumber int
	var totalNumber int
	var pacemakerLoaded bool

	cmd := &cobra.Command{
		Use:         "bhs-client",
		Long:        "basic hotstuff client",
		RunE:        startBasicHotStuffClient,
		Annotations: hsAnnotations,
	}
	cmd.Flags().IntVarP(&faultNumber, "fault_number", "f", 0, "fault_number, start from 0 to 5")
	cmd.Flags().IntVarP(&totalNumber, "total_number", "t", 4, "total_number, start from 4 to 16")
	cmd.Flags().BoolVarP(&pacemakerLoaded, "pacemaker_loaded", "p", false, "pacemaker_loaded, true or false, default false")
	return cmd
}

func init() {
	rootCmd.AddCommand(newBasicHotStuffClientCmd())
}

func startBasicHotStuffClient(cmd *cobra.Command, _ []string) (err error) {
	gCfg := LoadConfig()
	log.Debugf("config is %+v", gCfg)

	gCfg.FaultNumber, _ = cmd.Flags().GetInt("fault_number")
	gCfg.TotalNumber, _ = cmd.Flags().GetInt("total_number")
	gCfg.PacemakerLoaded, _ = cmd.Flags().GetBool("pacemaker_loaded")

	log.Infof(
		"starting basic hotstuff client: fault_number=%d, total_number=%d, pacemaker_loaded=%v",
		gCfg.FaultNumber,
		gCfg.TotalNumber,
		gCfg.PacemakerLoaded,
	)

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
