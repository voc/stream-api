# stream-api
Distributed media stream processing using an etcd backend.

The goal is coordinating live stream transcoding and monitoring across multiple machines in a fault-tolerant manner.

## Build
### Build Dependencies
  - go >= 1.16
  - npm
  - node
  - make

### Run build
```
make
```

## TODO
- add auth-backend service for sources
- add auth-frontend to monitor
- integrate streaming scripts/icecast config?
