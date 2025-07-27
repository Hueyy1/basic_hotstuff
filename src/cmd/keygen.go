package cmd

import (
	//"fmt"
	//"github.com/relab/gorums"
	"github.com/spf13/cobra"
	"hxy352/src/crypto/keygen"
)

func newGenerateCertsCmd() *cobra.Command {
	//var faultNumber int
	cmd := &cobra.Command{
		Use:  "generate-certs",
		Long: "generate certs",
		RunE: startGenerateCerts,
	}
	//cmd.Flags().IntVarP(&faultNumber, "fault_number", "f", 0, "fault_number, start from 0 to 5")
	return cmd
}

func init() {
	rootCmd.AddCommand(newGenerateCertsCmd())
}

func startGenerateCerts(cmd *cobra.Command, _ []string) (err error) {
	gCfg := LoadConfig()

	//faultNumber, _ := cmd.Flags().GetInt("fault_number")
	//gCfg.FaultNumber = faultNumber

	keygen.GenerateKeyChain(&gCfg)

	return nil
}
