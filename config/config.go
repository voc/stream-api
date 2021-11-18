package config

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

type TLSConfig struct {
	CertFile      string `yaml:"certFile"`
	KeyFile       string `yaml:"keyFile"`
	TrustedCAFile string `yaml:"trustedCaFile"`
}

type Network struct {
	Name      string     `yaml:"name"`
	Endpoints []string   `yaml:"endpoints"`
	TLS       *TLSConfig `yaml:"tls"`
}

type SourceConfig struct {
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
}

type PublisherConfig struct {
	Enable   bool           `yaml:"enable"`
	Sources  []SourceConfig `yaml:"sources"`
	Interval time.Duration  `yaml:"interval"`
	Timeout  time.Duration  `yaml:"timeout"`
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

type AuthConfig struct {
	Enable  bool   `yaml:"enable"`
	Address string `yaml:"address"`
}

type Config struct {
	Network   Network
	Publisher PublisherConfig
	Transcode TranscodeConfig
	Fanout    FanoutConfig
	Monitor   MonitorConfig
}

// Parse parses the config from a yaml file at path
func Parse(path string) (Config, error) {
	// default config
	cfg := Config{
		Publisher: PublisherConfig{
			Interval: time.Second * 3,
			Timeout:  time.Second * 15,
		},
	}
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
