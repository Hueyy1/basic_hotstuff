package model

type Config struct {
	Zap      ZapConfig         `yaml:"zap" mapstructure:"zap"`
	Http     HttpConfig        `yaml:"http" mapstructure:"http"`
	FilePath map[string]string `yaml:"file_path" mapstructure:"file_path"`
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
