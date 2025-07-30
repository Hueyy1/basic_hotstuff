package cmd

import (
	"fmt"
	"github.com/relab/gorums"
	"github.com/spf13/cobra"
	"hxy352/src/crypto/keygen"
	"hxy352/src/log"
	"hxy352/src/proto/basichotstuffpb"
	"hxy352/src/server"
	"hxy352/src/types"
	"net"
	//_ "net/http/pprof"
)

var hsAnnotations = map[string]string{"app": "hotstuff", "isGraceful": "true"}

func newBasicHotStuffServiceCmd() *cobra.Command {
	var id int
	var faultNumber int
	var totalNumber int
	var pacemakerLoaded bool

	cmd := &cobra.Command{
		Use:         "bhs",
		Long:        "basic hotstuff",
		RunE:        startBasicHotStuffService,
		Annotations: hsAnnotations,
	}
	cmd.Flags().IntVarP(&id, "id", "i", 0, "id of the replica, start from 0")
	cmd.Flags().IntVarP(&faultNumber, "fault_number", "f", 0, "fault_number, start from 0 to 5")
	cmd.Flags().IntVarP(&totalNumber, "total_number", "t", 4, "total_number, start from 4 to 16")
	cmd.Flags().BoolVarP(&pacemakerLoaded, "pacemaker_loaded", "p", false, "pacemaker_loaded, true or false, default false")
	return cmd
}

func init() {
	rootCmd.AddCommand(newBasicHotStuffServiceCmd())
}

func startBasicHotStuffService(cmd *cobra.Command, _ []string) (err error) {
	gCfg := LoadConfig()
	log.Debugf("config is %+v", gCfg)
	id, _ := cmd.Flags().GetInt("id")
	gCfg.Id = types.ID(id)

	gCfg.FaultNumber, _ = cmd.Flags().GetInt("fault_number")
	gCfg.TotalNumber, _ = cmd.Flags().GetInt("total_number")
	gCfg.PacemakerLoaded, _ = cmd.Flags().GetBool("pacemaker_loaded")

	log.Infof(
		"starting basic hotstuff service: fault_number=%d, total_number=%d, pacemaker_loaded=%v",
		gCfg.FaultNumber,
		gCfg.TotalNumber,
		gCfg.PacemakerLoaded,
	)

	// for pprof
	//if id == 0 {
	//	go func() {
	//		http.ListenAndServe("localhost:6060", nil)
	//	}()
	//}

	err = keygen.LoadPemFile(&gCfg)
	if err != nil {
		panic(err)
	}

	log.Debug(gCfg.ReplicaConf)

	currentReplica := gCfg.ReplicaConf[id]

	addr := fmt.Sprintf("%s:%d", currentReplica.Conf.Host, currentReplica.Conf.Port)
	log.Infof("Created replica at %s", addr)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panic(err)
	}
	gorumsSrv := gorums.NewServer()
	srv := server.NewBasicHotStuffImpl(&currentReplica, &gCfg)
	basichotstuffpb.RegisterBasicHotStuffServer(gorumsSrv, srv)

	go srv.Consensus.Run()

	gorumsSrv.Serve(lis)

	return nil
}
