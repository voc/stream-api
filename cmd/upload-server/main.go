package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/voc/stream-api/upload"
	"github.com/voc/stream-api/util"
)

func main() {
	config := defaultConfig()
	configPath := flag.String("config", "config.toml", "Set path to auth config")
	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.StringVar(&config.Server.Addr, "addr", config.Server.Addr, "Set listen address")
	flag.StringVar(&config.Server.OutputPath, "path", config.Server.OutputPath, "Set upload storage path")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	err := parseConfig(*configPath, &config)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read config")
	}
	log.Debug().Msgf("config: %+v", config)

	auth := upload.NewStaticAuth(config.Auth)
	server := upload.NewServer(auth, config.Server)
	log.Info().Msgf("listening on %s", config.Server.Addr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-ctx.Done():
		case err := <-server.Errors():
			log.Error().Err(err).Msg("server failed")
			cancel()
		}
	}()

	util.GracefulShutdown(ctx, func() {
		server.Stop()
	}, time.Second*2)
}
