package main

import "github.com/voc/stream-api/upload"

type Config struct {
	ListenAddress string
	Sinks         []*upload.Sink
}
