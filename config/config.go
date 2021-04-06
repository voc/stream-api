package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Network struct {
	Name      string
	Endpoints []string
}

type SourceConfig struct {
	Type string
	URL  string
}

type TranscodeConfig struct {
	Enable     bool
	Capacity   int
	Name       string
	ConfigPath string
}

type Config struct {
	Network   Network
	Sources   []SourceConfig
	Transcode TranscodeConfig
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
