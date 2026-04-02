## Ingest stage
The ingest machines run streaming software to receive incoming streams, authenticate them, and relay them to the transcoding stage.

### Streaming servers
Currently we support the following protocols:
 - RTMP - using nginx with [rtmp module](https://github.com/arut/nginx-rtmp-module)
 - SRT - using [srtrelay](https://github.com/voc/srtrelay)


### Stream registration
The ingest stage runs the [stream-api](../cmd/stream-api) binary with the [publish](../publish/) module enabled.
This scrapes the apis of nginx-rtmp and srtrelay to discover incoming streams, and registers them in the Consul backend.

The registration is placed in consul kv with the key `stream/{stream_id}`
and a json value describing the stream source.

See the [stream package](../stream/) for the schema of the stream registration and the available fields.

### Further reading
See the [transcoding stage](./transcoding.md) next.