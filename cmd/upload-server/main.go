package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/voc/stream-api/upload"
	"github.com/voc/stream-api/util"
)

func main() {
	config := defaultConfig()
	configPath := flag.String("config", "config.toml", "Set path to auth config")
	debug := flag.Bool("debug", false, "sets log level to debug")
	metricsAddr := flag.String("metrics", "localhost:9275", "Enable metrics server on this address")
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

	if *metricsAddr != "" {
		reg := prometheus.NewRegistry()
		config.Server.Registerer = reg
		reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

		// Start metrics server
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

		go func() {
			l, err := net.Listen("tcp", *metricsAddr)
			if err != nil {
				log.Error().Err(err).Str("addr", *metricsAddr).Msg("failed to listen for metrics")
				return
			}
			metricsSrv := &http.Server{Handler: mux}
			log.Info().Str("addr", *metricsAddr).Msg("metrics server listening")
			if err := metricsSrv.Serve(l); err != nil && err != http.ErrServerClosed {
				log.Error().Err(err).Msg("metrics server failed")
			}
		}()
	}

	auth := upload.NewStaticAuth(config.Auth)
	server, err := upload.NewServer(auth, config.Server)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}

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
