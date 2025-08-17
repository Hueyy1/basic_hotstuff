package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"hxy352/src/log"
	"hxy352/src/model"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var rootCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		isGraceful := cmd.Annotations["isGraceful"]
		if isGraceful == "true" {
			GracefulExist()
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, err := fmt.Fprintln(os.Stderr, err)
		if err != nil {
			return
		}
		os.Exit(1)
	}

}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {

	viper.SetConfigName("config-map.yaml")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs/")
	viper.AddConfigPath("../configs/")

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
	conf := LoadConfig()
	err := log.InitLogger(&conf.Zap)
	if err != nil {
		panic(err)
	}
}

func LoadConfig() model.Config {
	cfg := model.Config{}
	err := viper.Unmarshal(&cfg)
	if err != nil {
		panic(err)
	}

	return cfg
}

func GracefulExist() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-c
		log.Info("get a signal %s", s.String())
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			log.Info("exit")
			time.Sleep(time.Second)
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}
}
