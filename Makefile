.DEFAULT_GOAL := build

build:
	make -C monitor/frontend build
	go build ./cmd/...
.PHONY: build

clean:
	make -C monitor/frontend clean
	go clean
.PHONY: clean