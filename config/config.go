package config

import (
	"gopkg.in/yaml.v2"
)

type Network struct {
	Name string
	Endpoints []string
}

type Sources struct {

}

type Services struct {

}

type Config struct {
	Network Network
	Sources Sources
	Services Services
}

func Parse(string path) (Config, error) {
	cfg := Config{}
	err := yaml.Unmarshal([]byte(data), &cfg)
	if err != nil {
		return error
	}

	return cfg
}