## Transcoding stage

The transcoding machines run a control loop that watches the Consul backend for new stream registrations from the ingest stage, and starts transcoding jobs for them. 

### Transcoding jobs
The control loop as well as the transcoding script are implemented in the [transcoding repository](https://forgejo.c3voc.de/voc/transcode)

The transcoding uses a single FFmpeg process per stream to generate multiple renditions + thumbnails + audio tracks. It also automatically makes use of  VAAPI hardware acceleration when available.

### Transcoding Output
The output format is HLS playlists and TS segments, as well as thumbnail images which are pushed via HTTP to a local [upload-proxy](../cmd/upload-proxy/). The upload-proxy then forwards these to the origin stage for serving.

### Upload Proxy
The upload-proxy provides authorization and retransmits for the uploads so the FFmpeg process doesn't fall over when there are network issues.

The upload-proxy is dynamically configured using consul-template to watch for upload-server instances in the Consul backend and update its configuration accordingly.

### Transcoding metrics
The transcoders run a prometheus exporter that provides metrics about the transcoding jobs, which are scraped by a local telegraf and forwarded to the monitoring system.

The exporter receives the FFmpeg progress information via a unix socket and exposes it in a prometheus-friendly format.

### Further reading
See the [origin stage](./origin.md) next.