package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/voc/stream-api/client"
	"github.com/voc/stream-api/stream"
)

func decodeJSON(rd io.Reader, out interface{}) error {
	content, err := io.ReadAll(io.LimitReader(rd, 1048576))
	if err != nil {
		return err
	}
	if len(content) == 0 {
		return errors.New("empty payload")
	}

	err = json.Unmarshal(content, out)
	if err != nil {
		return err
	}
	return nil
}

func HandleGetAllStreamSettings(api client.KVAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		slug := params["slug"]
		log.Println("slug", slug)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var settings []stream.Settings
		var tmp stream.Settings
		data, err := api.GetWithPrefix(ctx, client.StreamSettingsPath(slug))
		if err != nil {
			log.Println("err", err)
			return
		}
		for _, field := range data {
			log.Println("got", string(field.Key), string(field.Value))
			err := json.Unmarshal(field.Value, &tmp)
			log.Println("foo", tmp, err)
			if err != nil {
				continue
			}

			settings = append(settings, tmp)
		}
		fmt.Printf("return %v\n", settings)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
	}
}

func HandleGetStreamSettings(api client.KVAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		slug := params["slug"]
		log.Println("slug", slug)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		data, err := api.Get(ctx, client.StreamSettingsPath(slug))
		if err != nil {
			log.Println("err", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}

func HandleSetStreamSettings(api client.KVAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var settings stream.Settings
		err := decodeJSON(r.Body, &settings)
		if err != nil {
			http.Error(w, fmt.Sprintf("parse failed: %s", err.Error()), http.StatusUnprocessableEntity)
			return
		}
		data, err := json.Marshal(settings)
		if err != nil {
			http.Error(w, fmt.Sprintf("marshal failed: %s", err.Error()), http.StatusInternalServerError)
		}
		log.Println("set", settings)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err = api.Put(ctx, client.StreamSettingsPath(settings.Slug), data)
		if err != nil {
			http.Error(w, fmt.Sprintf("put failed: %s", err.Error()), http.StatusInternalServerError)
			return
		}
	}
}
