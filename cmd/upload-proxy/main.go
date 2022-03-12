package main

import (
	"context"
	"flag"
	"io/ioutil"
	"net/url"
	"os"

	"github.com/pelletier/go-toml"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/voc/stream-api/upload"
	"github.com/voc/stream-api/util"
)

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

	data, err := ioutil.ReadFile(*config)
	if err != nil {
		log.Fatal().Err(err).Msg("config read failed")
	}
	err = toml.Unmarshal(data, &conf)
	if err != nil {
		log.Fatal().Err(err).Msg("config parse failed")
	}
	for _, sink := range conf.Sinks {
		url, err := url.Parse(sink.Address)
		if err != nil {
			log.Error().Str("address", sink.Address).Msg("invalid address")
			continue
		}
		sink.URL = *url
		log.Info().Str("sink", sink.URL.Host).Str("basePath", sink.URL.Path).Msg("added sink")
	}
	proxy := upload.NewProxy(ctx, conf.ListenAddress, conf.Sinks)
	log.Info().Msgf("listening on %s", conf.ListenAddress)

	select {
	case <-ctx.Done():
	case err := <-proxy.Errors():
		log.Error().Err(err).Msg("server failed")
		cancel()
	}

	proxy.Wait()
}
