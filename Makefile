stream-api:
        HOME=$$(pwd) git config --global http.sslVerify false
        if [ ! -f $$(pwd)/go/bin/go ]; then \
                wget --no-check-certificate https://go.dev/dl/go1.17.5.linux-amd64.tar.gz;\
                tar zxf $$(pwd)/go1.17.5.linux-amd64.tar.gz;\
        fi
        mkdir -p $$(pwd)/gopath
        CGO_ENABLED=0 HOME=$$(pwd) GOROOT=$$(pwd)/go GOPATH=$$(pwd)/gopath $$(pwd)/go/bin/go build -o stream-api ./cmd/stream-api

upload-server:
        HOME=$$(pwd) git config --global http.sslVerify false
        if [ ! -f $$(pwd)/go/bin/go ]; then \
                wget --no-check-certificate https://go.dev/dl/go1.17.5.linux-amd64.tar.gz;\
                tar zxf $$(pwd)/go1.17.5.linux-amd64.tar.gz;\
        fi
        mkdir -p $$(pwd)/gopath
        CGO_ENABLED=0 HOME=$$(pwd) GOROOT=$$(pwd)/go GOPATH=$$(pwd)/gopath $$(pwd)/go/bin/go build -o upload-server ./cmd/upload-server

upload-proxy:
        HOME=$$(pwd) git config --global http.sslVerify false
        if [ ! -f $$(pwd)/go/bin/go ]; then \
                wget --no-check-certificate https://go.dev/dl/go1.17.5.linux-amd64.tar.gz;\
                tar zxf $$(pwd)/go1.17.5.linux-amd64.tar.gz;\
        fi
        mkdir -p $$(pwd)/gopath
        CGO_ENABLED=0 HOME=$$(pwd) GOROOT=$$(pwd)/go GOPATH=$$(pwd)/gopath $$(pwd)/go/bin/go build -o upload-proxy ./cmd/upload-proxy

frontend:
        make -C monitor/frontend build

build: stream-api upload-server upload-proxy
