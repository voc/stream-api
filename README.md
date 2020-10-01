# stream-api

Distributed work management
==========================
db scheme:
services/dist/live.ber.c3voc.de: {capacity: 10}
services/dist/live2.ber.c3voc.de: {capacity: 15}

services/transcode/speedy.lan.c3voc.de: {capacity: 3}
services/transcode/loop-transcoder.c3voc.de: {capacity: 1, stream_filter: "^sloop$"}
services/record/deinemutter.fem.tu-ilmenau.de: {stream_filter: "^sloop$"}

streams/s1 {key: "s1", slots: ["dist", "transcode", "fanout", "record"]}


global2 highlevel description:

= processing chain:


local2:
services:
  source:
    plugin: poll
    source: true
    stream_vars:
      url: http://ingest.c3voc.de:8000
  plugin component
  write->services/source

  assigner component
  watch->services/
  for each stream:
    watch->streams
    write->streams/sX

  dist:
    plugin: static
    stream_vars:
      url: http://live.ber.c3voc.de:7999

  write->services/dist
  watch->streams
    -> activate when we are assigned
  for each stream:
    may write streams/sX/dist

  transcode:
    plugin:
      name: exec
      path: /transcode_vaapi.sh
      env:
        key: ${stream:key}
        pull_url: ${source:url}/${stream:key}
        push_prefix: ${dist_url}
    global_vars:
      capacity: 1
      stream_filter: "^foobar$"

  write->services/transcode
  watch->streams
    -> activate when we are assigned
  for each stream:
    may write streams/sX/transcode



## Usage

### defining sources
There are actually two different forms of plugins, standard processing plugins and source plugins.

Sources announce streams to the api, while the standard plugins provide some kind of ... for the stream.

The current source plugins are "poll" and "static"

The chain attribute decides which processing blocks the stream should have.

```yaml
# sources this node should provide
sources:
  - plugin: "poll"
    vars:
      url: "http://ingest.c3voc.de:8000"
    chain:
      - dist
      - transcode
      - fanout_h264
      - fanout_vpx
      - fanout_audio
      - fanout_thumbnail
    stream:
      url: "http://ingest.c3voc.de:8000"
```

### defining services
```yaml
# services this node should provide
services:

```


## Internals
### DB use
- keys use colon separated "path"-segments
- values are json

- workers register their services under service:<type>:<hostname>
  - optional values are url, capacity, used, filter
  - e.g: service:transcode:speedy.lan.c3voc.de = {"capacity": 3, "used": 1, "filter": "^s[1-9]$"}
- streams are created under stream:<key>
  - required values are key, chain
  - may contain additional user values
  - e.g.: stream:s1 = {"key": "s1", "chain": ["dist", "transcode", "fanout"]}
- stream workers 
