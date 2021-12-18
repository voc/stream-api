.DEFAULT_GOAL := build

stream-api:
	CGO_ENABLED=0 go build ./cmd/stream-api
.PHONY: stream-api

build:
	make -C monitor/frontend build
	go build ./cmd/...
.PHONY: build

clean:
	make -C monitor/frontend clean
	go clean
.PHONY: clean
