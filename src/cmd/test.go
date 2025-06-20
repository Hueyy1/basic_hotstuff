package cmd

import (
	"github.com/spf13/cobra"
	"hxy352/src/log"
)

func newTestConfCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "test-conf",
		Long: "test config",
		RunE: startTestConf,
	}
	return cmd
}

func init() {
	rootCmd.AddCommand(newTestConfCmd())
}

func startTestConf(_ *cobra.Command, _ []string) (err error) {
	gCfg := LoadConfig()
	log.Infof("config is %+v", gCfg)
	return nil
}
