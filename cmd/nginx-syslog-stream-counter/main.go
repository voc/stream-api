package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/voc/stream-api/logreceiver"
	"golang.org/x/sys/unix"
)

// maxSegmentDuration is the maximum duration of a segment to expect
const maxSegmentDuration = time.Second * 5

func main() {
	if err := run(); err != nil {
		slog.Error("collector failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	slidingWindow := flag.Duration("sliding-window-duration", time.Second*30, "duration of the sliding window for the counting")
	prometheusListen := flag.String("prometheus-listen", ":9273", "listen address of the prometheus endpoint")
	socket := flag.String("socket", "/var/log/relay.sock", "syslog socket")
	debug := flag.Bool("debug", false, "enable debug mode")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, unix.SIGTERM)
	defer cancel()

	logOpts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	if *debug {
		logOpts.Level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, logOpts)))

	reg := prometheus.NewPedanticRegistry()

	// Add the standard process and Go metrics to the custom registry.
	reg.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)

	_, err := logreceiver.NewParser(ctx, logreceiver.ParserConfig{
		SocketPath:        *socket,
		MetricsRegisterer: reg,
		SlidingWindow:     *slidingWindow,
		MinSegments:       int(*slidingWindow/maxSegmentDuration) / 2,
	})
	if err != nil {
		return err
	}

	// serve metrics
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	server := http.Server{
		Handler: mux,
	}

	listener, err := net.Listen("tcp", *prometheusListen)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", *prometheusListen, err)
	}
	defer listener.Close()
	slog.Info("serving metrics on", "addr", *prometheusListen)

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to serve prometheus metrics", "err", err)
			cancel()
		}
	}()
	<-ctx.Done()
	return nil
}
