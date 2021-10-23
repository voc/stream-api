package main

import (
	"context"
	"flag"
	"log"

	"github.com/voc/stream-api/upload"
	"github.com/voc/stream-api/util"
)

func main() {
	path := flag.String("path", "/tmp/upload", "Set upload storage path")
	config := flag.String("config", "auth.toml", "Set path to auth config")
	addr := flag.String("addr", ":8080", "Set listen address")
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	util.HandleSignal(ctx, cancel)
	auth, err := upload.NewStaticAuth(*config)
	if err != nil {
		log.Fatal(err)
	}
	server := upload.NewServer(ctx, *addr, *path, auth)

	select {
	case <-ctx.Done():
	case err := <-server.Errors():
		log.Println("Server failed:", err)
		cancel()
	}

	server.Wait()
}
