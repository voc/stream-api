stream-api:
	HOME=$$(pwd) git config --global http.sslVerify false
	if [ ! -f $$(pwd)/go/bin/go ]; then \
		wget --no-check-certificate https://go.dev/dl/go1.17.5.linux-amd64.tar.gz;\
		tar zxf $$(pwd)/go1.17.5.linux-amd64.tar.gz;\
	fi
	mkdir -p $$(pwd)/gopath
	CGO_ENABLED=0 HOME=$$(pwd) GOROOT=$$(pwd)/go GOPATH=$$(pwd)/gopath $$(pwd)/go/bin/go build -o stream-api ./cmd/stream-api
	CGO_ENABLED=0 HOME=$$(pwd) GOROOT=$$(pwd)/go GOPATH=$$(pwd)/gopath $$(pwd)/go/bin/go build -o upload-server ./cmd/upload-server
	CGO_ENABLED=0 HOME=$$(pwd) GOROOT=$$(pwd)/go GOPATH=$$(pwd)/gopath $$(pwd)/go/bin/go build -o upload-proxy ./cmd/upload-proxy

frontend:
	make -C monitor/frontend build

build: stream-api upload-server upload-proxy

install: stream-api
	mkdir -p $$(pwd)/debian/stream-api/usr/bin
	install -m 0755 stream-api $$(pwd)/debian/stream-api/usr/bin
	install -m 0755 upload-server $$(pwd)/debian/stream-api/usr/bin
	install -m 0755 upload-proxy $$(pwd)/debian/stream-api/usr/bin

