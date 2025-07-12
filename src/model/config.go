package model

import "hxy352/src/types"

type Config struct {
	Zap         ZapConfig         `yaml:"zap" mapstructure:"zap"`
	Http        HttpConfig        `yaml:"http" mapstructure:"http"`
	FilePath    map[string]string `yaml:"file_path" mapstructure:"file_path"`
	Replica     []ReplicaConfig   `yaml:"replica" mapstructure:"replica"`
	ReplicaConf []ReplicaConf
	Id          types.ID
}

type ZapConfig struct {
	Level            string   `yaml:"level" mapstructure:"level"`
	OutputPaths      []string `yaml:"outputPaths" mapstructure:"outputPaths"`
	ErrorOutputPaths []string `yaml:"errorOutputPaths" mapstructure:"errorOutputPaths"`
	Encoding         string   `yaml:"encoding" mapstructure:"encoding"`
}

type HttpConfig struct {
	RunMode  string `yaml:"run_mode" mapstructure:"run_mode"`
	HttpPort int    `yaml:"http_port" mapstructure:"http_port"`
}

type ReplicaConfig struct {
	Id          int    `yaml:"id" mapstructure:"id"`
	Host        string `yaml:"host" mapstructure:"host"`
	Port        int    `yaml:"port" mapstructure:"port"`
	PrivateFile string `yaml:"privateFile" mapstructure:"privateFile"`
	PublicFile  string `yaml:"publicFile" mapstructure:"publicFile"`
}
