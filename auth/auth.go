package auth

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/voc/stream-api/client"
	"github.com/voc/stream-api/config"
)

// Auth subscribes to stream settings and responds to auth request over http
type Auth struct {
	watcher *watcher
	name    string
	api     client.ServiceAPI
	done    sync.WaitGroup
}

var defaultScrapeInterval = time.Second * 3

// New creates a new Auth
func New(ctx context.Context, api client.ServiceAPI, name string, conf config.AuthConfig) *Auth {
	a := &Auth{
		watcher: newWatcher(ctx, api),
		name:    name,
		api:     api,
	}

	// watch settings updates
	a.done.Add(1)
	go a.run(ctx, &conf)

	return a
}

func (a *Auth) Wait() {
	a.watcher.Wait()
	a.done.Wait()
}

func (a *Auth) run(parentContext context.Context, conf *config.AuthConfig) {
	defer a.done.Done()

	mux := http.NewServeMux()
	mux.HandleFunc("/", authHandler(a.watcher))

	srv := &http.Server{Addr: conf.Address, Handler: mux}

	a.done.Add(1)
	go func() {
		defer a.done.Done()
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal().Msgf("ListenAndServe(): %v", err)
		}
	}()

	<-parentContext.Done()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := srv.Shutdown(ctx)
	if err != nil {
		log.Error().Err(err).Msg("auth: server shutdown")
	}
}

func authHandler(watcher *watcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Error().Err(err).Msg("auth: parse")
			http.Error(w, "Invalid form", http.StatusUnauthorized)
			return
		}

		app := r.PostForm.Get("app")
		name := r.PostForm.Get("name")
		auth := r.PostForm.Get("auth")

		log.Debug().Msgf("auth: %s/%s, secret '%s'", app, name, auth)

		success := watcher.Auth(app, name, auth)
		if !success {
			log.Debug().Msgf("auth: %s/%s unauthorized", app, name)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Debug().Msgf("auth: %s/%s ok", app, name)
	}
}
