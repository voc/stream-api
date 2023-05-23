package main

type Config struct {
	ListenAddress string
	Sinks         []SinkConfig
}
