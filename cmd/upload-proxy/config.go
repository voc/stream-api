package main

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
)

type Config struct {
	ListenAddress string
	Sinks         []SinkConfig
}

func (c *Config) ReadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	err = toml.Unmarshal(data, c)
	if err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}
