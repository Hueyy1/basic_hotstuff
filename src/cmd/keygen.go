package cmd

import (
	//"fmt"
	//"github.com/relab/gorums"
	"github.com/spf13/cobra"
	"hxy352/src/crypto/keygen"
)

func newGenerateCertsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "generate-certs",
		Long: "generate certs",
		RunE: startGenerateCerts,
	}
	return cmd
}

func init() {
	rootCmd.AddCommand(newGenerateCertsCmd())
}

func startGenerateCerts(cmd *cobra.Command, _ []string) (err error) {
	gCfg := LoadConfig()

	keygen.GenerateKeyChain(&gCfg)

	return nil
}
