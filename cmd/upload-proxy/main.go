package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/voc/stream-api/util"
)

func run(parentCtx context.Context, conf *Config) error {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()
	var sinks []*Sink
	for _, sinkConfig := range conf.Sinks {
		sink, err := NewSink(sinkConfig)
		if err != nil {
			return fmt.Errorf("sink init failed: %w", err)
		}
		sinks = append(sinks, sink)
		log.Info().Str("sink", sink.url.Host).Str("basePath", sink.url.Path).Msg("added sink")
	}
	proxy, err := NewProxy(ctx, conf.ListenAddress, sinks)
	if err != nil {
		return fmt.Errorf("proxy init failed: %w", err)
	}
	log.Info().Msgf("listening on %s", conf.ListenAddress)

	select {
	case <-ctx.Done():
	case err := <-proxy.Errors():
		log.Error().Err(err).Msg("server failed")
		cancel()
	}

	proxy.Wait()
	return nil
}

func main() {
	conf := Config{}
	config := flag.String("config", "config.toml", "Set path to proxy config")
	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.StringVar(&conf.ListenAddress, "addr", ":8080", "Set listen address")
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	util.HandleSignal(ctx, cancel)

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	data, err := os.ReadFile(*config)
	if err != nil {
		log.Fatal().Err(err).Msg("config read failed")
	}
	err = toml.Unmarshal(data, &conf)
	if err != nil {
		log.Fatal().Err(err).Msg("config parse failed")
	}

	// Run proxy
	if run(ctx, &conf); err != nil {
		log.Fatal().Err(err)
	}
}
