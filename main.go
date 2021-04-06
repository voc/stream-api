package main

import (
	// "crypto/tls"
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Showmax/go-fqdn"
	"github.com/voc/stream-api/client"
	"github.com/voc/stream-api/config"
	"github.com/voc/stream-api/publish"
	"github.com/voc/stream-api/transcode"
)

func handleSignal(ctx context.Context, cancel context.CancelFunc) {
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
				log.Info().Msgf("caught signal %s", s)
				cancel()
			}
		}
	}()
}

func getHostname() string {
	name, err := fqdn.FqdnHostname()
	if err != nil {
		log.Error().Err(err).Msg("fqdn")
		if err != fqdn.ErrFqdnNotFound {
			return name
		}

		name, err = os.Hostname()
		if err != nil {
			log.Fatal().Err(err).Msg("hostname")
		}
	}
	return name
}

type Service interface {
	Wait()
}

func main() {
	name := getHostname()
	configPath := flag.String("config", "config.yml", "path to configuration file")
	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.StringVar(&name, "name", name, "set network name (defaults to fqdn)")
	// var action = flag.String("action", "watch", "action: (watch|write)")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// parse config
	cfg, err := config.Parse(*configPath)
	if err != nil {
		log.Fatal().Err(err)
	}

	// setup etcd context
	cliCtx, cliCancel := context.WithCancel(context.Background())
	defer cliCancel()

	// connect to etcd
	cfg.Network.Name = name
	log.Debug().Msgf("Creating client %s", cfg.Network)
	cli := client.NewClient(cliCtx, cfg.Network)

	// setup service context
	ctx, cancel := context.WithCancel(context.Background())
	handleSignal(ctx, cancel)
	defer cancel()

	// setup publisher
	var services []Service
	if len(cfg.Sources) > 0 {
		log.Debug().Msgf("Creating publisher")
		services = append(services, publish.NewPublisher(ctx, cli, cfg.Sources))
	}

	// setup transcoder
	if cfg.Transcode.Enable {
		log.Debug().Msgf("Creating transcoder")
		cfg.Transcode.Name = name
		services = append(services, transcode.NewTranscoder(ctx, cli, cfg.Transcode))
	}

	// Wait for graceful shutdown
	<-ctx.Done()
	for _, service := range services {
		service.Wait()
	}
	cliCancel()
	cli.Wait()
}
