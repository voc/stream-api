package util

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// GracefulShutdown waits for a signal or context close and then calls the shutdown function in a blocking fashion.
// If the shutdown function does not complete within the timeout, the function exits early.
func GracefulShutdown(ctx context.Context, handleShutdown func(), timeout time.Duration) {
	// Listen for signals
	s := make(chan os.Signal, 1)
	signal.Notify(s,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM)

	// Do graceful shutdown on signal or context close
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case sig := <-s:
			if sig != syscall.SIGHUP {
				break loop
			}
		}
	}
	done := make(chan struct{})
	go func() {
		handleShutdown()
		close(done)
	}()

	select {
	case <-done:
		log.Println("graceful shutdown complete")
	case <-time.After(timeout):
		log.Println("graceful shutdown timed out")
	}
}
