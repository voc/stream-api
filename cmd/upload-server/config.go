package main

import (
	"io/ioutil"

	"github.com/pelletier/go-toml"
	"github.com/rs/zerolog/log"

	"github.com/voc/stream-api/upload"
)

type Config struct {
	Server upload.ServerConfig
	Auth   upload.AuthConfig
}

func defaultConfig() Config {
	return Config{
		Server: upload.ServerConfig{
			Addr:            "localhost:8080",
			OutputPath:      "/tmp/upload",
			MaxPlaylistSize: 256 * 1024,
			MaxSegmentSize:  100 * 1024 * 1024,
			PlaylistSize:  10,
		},
	}
}

func parseConfig(path string, conf *Config) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = toml.Unmarshal(data, conf)
	if err != nil {
		return err
	}
	log.Info().Msgf("reading config from %s", path)
	return nil
}
