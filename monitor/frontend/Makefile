.DEFAULT_GOAL := build

build:
	HOME=$$(pwd) npm install --no-fund --no-optional --no-audit --unsafe-perm=true --allow-root --legacy-peer-deps --loglevel=error
    HOME=$$(pwd) npm run build

.PHONY: build