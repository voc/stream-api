package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"net/url"

	"github.com/pelletier/go-toml"
	"github.com/voc/stream-api/upload"
	"github.com/voc/stream-api/util"
)

func main() {
	conf := Config{}
	config := flag.String("config", "config.toml", "Set path to proxy config")
	flag.StringVar(&conf.ListenAddress, "addr", ":8080", "Set listen address")
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	util.HandleSignal(ctx, cancel)

	data, err := ioutil.ReadFile(*config)
	if err != nil {
		log.Fatal("config read failed:", err)
	}
	err = toml.Unmarshal(data, &conf)
	if err != nil {
		log.Fatal("config parse failed:", err)
	}
	for _, sink := range conf.Sinks {
		url, err := url.Parse(sink.Address)
		if err != nil {
			log.Println("invalid address", sink.Address)
			continue
		}
		sink.URL = *url
	}
	proxy := upload.NewProxy(ctx, conf.ListenAddress, conf.Sinks)

	select {
	case <-ctx.Done():
	case err := <-proxy.Errors():
		log.Println("Server failed:", err)
		cancel()
	}

	proxy.Wait()
}
