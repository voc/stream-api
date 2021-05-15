package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Network struct {
	Name      string   `yaml:"name"`
	Endpoints []string `yaml:"endpoints"`
}

type SourceConfig struct {
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
}

type TranscodeConfig struct {
	Enable     bool   `yaml:"enable"`
	Capacity   int    `yaml:"capacity"`
	ConfigPath string `yaml:"configPath"`
	Sink       string `yaml:"sink"`
}

type FanoutConfig struct {
	Enable     bool   `yaml:"enable"`
	ConfigPath string `yaml:"configPath"`
	Sink       string `yaml:"sink"`
}

type MonitorConfig struct {
	Enable  bool   `yaml:"enable"`
	Address string `yaml:"address"`
}

type Config struct {
	Network   Network
	Sources   []SourceConfig
	Transcode TranscodeConfig
	Fanout    FanoutConfig
	Monitor   MonitorConfig
}

// Parse parses the config from a yaml file at path
func Parse(path string) (Config, error) {
	cfg := Config{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}
