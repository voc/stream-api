# streaming docs

The voc streaming infra is structured using dedicated machines for each stage of the streaming pipeline. Currently the stages are:
  - Encoding - capture and initial encoding of the live stream (either on VOC-owned hardware using voctomix or from 3rd-party encoders)
  - [Ingest](./docs/ingest.md) - receive and relay incoming streams
  - [Transcoding](./docs/transcoding.md) - convert streams to multiple formats and bitrates
  - [Origin](./docs/origin.md) - serve as the primary source for the HTTP streams
  - [Edge](./docs/edge.md) - distribute streams to end users
  - [Loadbalancer](./docs/loadbalancer.md) - redirect users to edge servers
  - [Monitoring](./docs/monitoring.md)