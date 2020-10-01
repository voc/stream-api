package main

import (
	// "crypto/tls"
	"log"
	"flag"
	"context"
	"os"
	"os/signal"
	"time"
	// "github.com/symptog/jitsi-colibri-exporter/collector"
	"bitbucket.fem.tu-ilmenau.de/scm/~ischluff/stream-api/config"
	"bitbucket.fem.tu-ilmenau.de/scm/~ischluff/stream-api/client"
	"bitbucket.fem.tu-ilmenau.de/scm/~ischluff/stream-api/service"
)

type cancelFunc func(){}

func handleSignal(ctx context.Context, cancel cancelFunc) {
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for {
			select {
				case <-ctx.Done():
					return
				case s := <-c:
					log.Println("signal", s)
					cancel()
			}
		}
	}()
}

func main() {
	var configPath = flag.String("config", "config.yml", "path to configuration file")
	// var action = flag.String("action", "watch", "action: (watch|write)")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	handleSignal(ctx, cancel)
	defer cancel()

	cfg, err := config.Parse(configPath)
	if err != nil {
		log.Fatal(err)
	}

	registry := service.NewRegistry(cfg.Services)

	cli := client.NewClient(ctx, cfg.Network)
	cli.Publish(registry.Services())

	cli.Watch()

	// assigner := service.NewAssigner(ctx, client, cfg)

	// if *action == "watch" {
	// 	client.Watch(ctx)
	// } else {
	// 	client.Write(ctx)
	// }

	// Wait for graceful shutdown
	<-ctx.Done()
	time.Sleep(500*time.Millisecond)
}