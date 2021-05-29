package monitor

import (
	"context"
	"net/http"
	"sync"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/voc/stream-api/config"
)

type clientMap map[*websocket.Conn]bool

type server struct {
	upgrader websocket.Upgrader
	done     sync.WaitGroup

	// update channels
	addClient    chan *websocket.Conn
	removeClient chan *websocket.Conn
	updates      <-chan map[string]interface{}

	// local state
	transcoders       *transcodersJson
	fanouts           *fanoutsJson
	streams           *streamsJson
	streamTranscoders *streamTranscodersJson
	state             map[string]interface{}
}

func newServer(ctx context.Context, updates <-chan map[string]interface{}, conf config.MonitorConfig) *server {
	s := &server{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		updates:      updates,
		addClient:    make(chan *websocket.Conn, 1),
		removeClient: make(chan *websocket.Conn, 1),
		state:        make(map[string]interface{}),
	}
	s.done.Add(1)
	go s.run(ctx, &conf)
	return s
}

func (s *server) Wait() {
	s.done.Done()
}

func (s *server) run(parentContext context.Context, conf *config.MonitorConfig) {
	defer s.done.Done()

	router := mux.NewRouter()
	router.HandleFunc("/", indexHandler()).Methods("GET")
	router.HandleFunc("/ws", s.wsHandler)
	router.PathPrefix("/").Handler(http.FileServer(http.FS(static)))

	srv := &http.Server{Addr: conf.Address, Handler: router}

	s.done.Add(1)
	go func() {
		defer s.done.Done()
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal().Msgf("ListenAndServe(): %v", err)
		}
	}()

	clients := make(clientMap)
	for {
		select {
		case <-parentContext.Done():
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := srv.Shutdown(ctx)
			if err != nil {
				log.Error().Err(err).Msg("server shutdown")
			}
		case ws := <-s.addClient:
			ws.WriteJSON(s.state)
			clients[ws] = true
		case ws := <-s.removeClient:
			delete(clients, ws)
		// generic send
		case update := <-s.updates:
			for k, v := range update {
				s.state[k] = v
			}
			s.broadcast(clients, update)
		}
	}
}

type templateData struct {
	Prefix string
	Errors []error
}

func indexHandler() http.HandlerFunc {
	data, err := static.ReadFile("frontend/public/index.html")
	if err != nil {
		log.Fatal().Err(err).Msg("index read")
	}
	tmpl, err := template.New("index").Parse(string(data))
	if err != nil {
		log.Fatal().Err(err).Msg("index template")
	}

	tmplData := templateData{}
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, tmplData)
		// w.Write(data)
	}
}

func (s *server) broadcast(clients clientMap, v interface{}) {
	for ws := range clients {
		err := ws.WriteJSON(v)
		if err != nil {
			log.Error().Err(err).Msg("write")
		}
	}
}

func (s *server) wsHandler(w http.ResponseWriter, r *http.Request) {
	c, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("ws upgrade failed")
		return
	}
	defer c.Close()

	// register client
	s.addClient <- c
	defer func() { s.removeClient <- c }()

	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Error().Err(err).Msg("ws read")
			}
			break
		}
	}
}
