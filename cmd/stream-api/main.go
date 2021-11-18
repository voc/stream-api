package main

import (
	// "crypto/tls"
	"context"
	"flag"
	"os"
	"os/signal"

	"net/http"
	_ "net/http/pprof"

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
	profile := flag.String("profile", "", "set pprof address")
	flag.StringVar(&name, "name", name, "set network name (defaults to fqdn)")
	// var action = flag.String("action", "watch", "action: (watch|write)")
	flag.Parse()
	exitCode := 0

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if *profile != "" {
		go http.ListenAndServe(*profile, nil)
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
	log.Debug().Interface("config", cfg.Network).Msgf("Creating client")
	cli, err := client.NewConsulClient(cliCtx, cfg.Network)
	if err != nil {
		log.Fatal().Err(err).Msg("client:")
	}

	// setup service context
	ctx, cancel := context.WithCancel(context.Background())
	handleSignal(ctx, cancel)
	defer cancel()
	var services []Service

	// shutdown app on etcd client errors
	go func() {
		errChan := cli.Errors()
		select {
		case err := <-errChan:
			log.Error().Err(err).Msgf("client:")
			exitCode = 255
			log.Debug().Msg("cancel after client error")
			cancel()
		case <-cliCtx.Done():
			return
		}
	}()

	// setup monitor
	// if cfg.Monitor.Enable {
	// 	log.Debug().Msgf("Creating monitor %v", cfg.Monitor)
	// 	services = append(services, monitor.New(ctx, cfg.Monitor, cli))
	// }

	// setup publisher
	if cfg.Publisher.Enable {
		log.Debug().Msgf("Creating publisher %v", cfg.Publisher)
		services = append(services, publish.New(ctx, &cfg.Publisher, cli, name))
	}

	// setup transcoder
	if cfg.Transcode.Enable {
		log.Debug().Msgf("Creating transcoder %v", cfg.Transcode)
		services = append(services, transcode.New(ctx, cfg.Transcode, cli, name))
	}

	// // setup fanout
	// if cfg.Fanout.Enable {
	// 	log.Debug().Msgf("Creating fanout %v", cfg.Fanout)
	// 	services = append(services, fanout.New(ctx, cfg.Fanout, cli, name))
	// }

	// Wait for graceful shutdown
	<-ctx.Done()
	for _, service := range services {
		service.Wait()
	}
	cliCancel()
	log.Debug().Msgf("exitcode: %d", exitCode)
	os.Exit(exitCode)
}
