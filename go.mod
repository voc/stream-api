module github.com/voc/stream-api

go 1.16

require (
	github.com/Showmax/go-fqdn v1.0.0
	github.com/coreos/bbolt v0.0.0-00010101000000-000000000000 // indirect
	github.com/coreos/etcd v3.3.27+incompatible
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/godbus/dbus v4.1.0+incompatible
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/hashicorp/consul/api v1.11.0
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/minio/pkg v1.1.5
	github.com/pelletier/go-toml v1.9.4
	github.com/pkg/errors v0.9.1
	github.com/quangngotan95/go-m3u8 v0.1.0
	github.com/rs/zerolog v1.25.0
	github.com/soheilhy/cmux v0.1.5 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20201229170055-e5319fda7802 // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	github.com/zencoder/go-dash v0.0.0-20201006100653-2f93b14912b2
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools/v3 v3.4.0
)

replace github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0

replace github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.5

replace github.com/coreos/etcd => go.etcd.io/etcd v3.3.27+incompatible

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0
