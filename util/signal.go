package util

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func HandleSignal(ctx context.Context, cancel context.CancelFunc) {
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case s := <-c:
				log.Println("caught signal", s)
				if s == syscall.SIGHUP {
					continue
				}
				cancel()
			}
		}
	}()
}
