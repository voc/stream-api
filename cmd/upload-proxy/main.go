package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	conf := Config{}
	configPath := flag.String("config", "config.toml", "Set path to proxy config")
	authConfigPath := flag.String("auth-config", "", "Set path to separate auth config (optional)")
	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.StringVar(&conf.ListenAddress, "addr", ":8080", "Set listen address")
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if err := conf.Load(*configPath, *authConfigPath); err != nil {
		log.Fatal().Err(err).Msg("config parse failed")
	}

	// Run proxy
	if err := run(ctx, conf, *configPath, *authConfigPath); err != nil {
		log.Fatal().Err(err).Msg("proxy run failed")
	}
}

func run(parentCtx context.Context, conf Config, configPath string, authConfigPath string) error {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()
	proxy, err := NewProxy(ctx, conf)
	if err != nil {
		return fmt.Errorf("proxy init failed: %w", err)
	}
	log.Info().Msgf("listening on %s", conf.ListenAddress)

	signalReload := make(chan os.Signal, 1)
	signal.Notify(signalReload, syscall.SIGHUP)

outer:
	for {
		select {
		case <-ctx.Done():
			break outer
		case err := <-proxy.Errors():
			log.Error().Err(err).Msg("server failed")
			cancel()
			break outer
		case <-signalReload:
			log.Info().Msg("reloading config")
			var newConf Config
			if err := newConf.Load(configPath, authConfigPath); err != nil {
				log.Error().Err(err).Msg("config parse failed")
				break
			}
			_ = proxy.UpdateConfig(ctx, newConf)
		}
	}

	proxy.Wait()
	return nil
}
