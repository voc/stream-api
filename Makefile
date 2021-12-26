.DEFAULT_GOAL := build

stream-api:
	CGO_ENABLED=0 go build ./cmd/stream-api
.PHONY: stream-api

upload-server:
	CGO_ENABLED=0 go build ./cmd/upload-server
.PHONY: upload-server

upload-proxy:
	CGO_ENABLED=0 go build ./cmd/upload-proxy
.PHONY: upload-proxy

all: stream-api upload-server upload-proxy
.PHONY: all

frontend:
	make -C monitor/frontend build
.PHONY: frontend

reallyall: frontend all
.PHONY: reallyall
