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
	//var replicaNumber int
	var faultNumber int

	cmd := &cobra.Command{
		Use:         "bhs",
		Long:        "basic hotstuff",
		RunE:        startBasicHotStuffService,
		Annotations: hsAnnotations,
	}
	cmd.Flags().IntVarP(&id, "id", "i", 0, "id of the replica, start from 0")
	//cmd.Flags().IntVarP(&replicaNumber, "replica_number", "r", 4, "replica_number, start from 4 to 10")
	cmd.Flags().IntVarP(&faultNumber, "fault_number", "f", 0, "fault_number, start from 0 to 5")
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

	faultNumber, _ := cmd.Flags().GetInt("fault_number")
	gCfg.FaultNumber = faultNumber

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

	go srv.Consensus.HandleMsg()

	go srv.Consensus.HandleReq()

	gorumsSrv.Serve(lis)

	return nil
}
