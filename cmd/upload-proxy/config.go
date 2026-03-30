package main

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
)

type Config struct {
	ListenAddress string
	Sinks         []SinkConfig
	Auth          AuthConfig
}

type SinkConfig struct {
	Address   string
	QueueSize int `toml:"queue-size"`
	AuthConfig
}

func (s *SinkConfig) ApplyDefaults() {
	if s.QueueSize == 0 {
		s.QueueSize = DefaultQueueSize
	}
	if s.Username != "" && (s.AuthType != AuthTypeBasic && s.AuthType != AuthTypeDigest) {
		s.AuthType = AuthTypeBasic
	}
}

type AuthConfig struct {
	Username string
	Password string
	AuthType AuthType `toml:"auth-type"`
}

type AuthType string

const (
	AuthTypeNone   AuthType = "none"
	AuthTypeBasic  AuthType = "basic"
	AuthTypeDigest AuthType = "digest"
)

func (c *Config) Load(path string, authPath string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	err = toml.Unmarshal(data, c)
	if err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	// load separate auth config if specified
	if authPath != "" {
		data, err := os.ReadFile(authPath)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		err = toml.Unmarshal(data, &c.Auth)
		if err != nil {
			return fmt.Errorf("unmarshal: %w", err)
		}
	}
	// apply auth config to all sinks if set
	for i := range c.Sinks {
		c.Sinks[i].ApplyDefaults()
		if c.Auth.Username != "" {
			c.Sinks[i].AuthConfig = c.Auth
		}
	}
	return nil
}
