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
	"golang.org/x/sys/unix"
)

func main() {
	if err := run(); err != nil {
		slog.Error("exporter failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	metricsListen := flag.String("listen", "localhost:9274", "listen address for prometheus metrics and FFmpeg progress endpoint")
	debug := flag.Bool("debug", false, "enable debug logging")
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
	reg.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)
	metrics := NewMetrics(reg)
	progressHandler := NewProgressHandler(metrics)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.Handle("/progress/", progressHandler)

	listener, err := net.Listen("tcp", *metricsListen)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", *metricsListen, err)
	}
	slog.Info("listening on address", "addr", listener.Addr())
	defer listener.Close()

	server := &http.Server{
		Handler: mux,
	}

	slog.Info("serving metrics and progress endpoint", "addr", *metricsListen)

	// Run HTTP server in background
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server error", "err", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	slog.Info("shutdown signal received")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second*2)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("error during shutdown", "err", err)
		return err
	}
	return nil
}
