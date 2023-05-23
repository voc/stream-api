package main

import (
	"context"
	"testing"
	"time"
)

func TestSmoke(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := run(ctx, &Config{
		ListenAddress: "127.0.0.1:0",
		Sinks: []SinkConfig{{
			Address: "http://1.2.3.4:5678/upload",
		}},
	})
	if err != nil {
		t.Error(err)
	}
}
