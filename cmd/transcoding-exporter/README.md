# Transcoding Exporter

A Prometheus exporter service that collects metrics from FFmpeg transcoding jobs via HTTP progress updates.

## Overview

The transcoding exporter listens on an HTTP port for FFmpeg progress data and exports aggregated metrics for Prometheus scraping. It tracks per-job metrics like frames processed, bytes written, bitrate, and frame rate. The stream ID is extracted from the HTTP request path.

## Building

```bash
go build ./cmd/transcoding-exporter
```

## Integration with FFmpeg

FFmpeg sends progress information to the exporter via continuous HTTP POST/PUT requests. The stream ID is extracted from the URL path: `/progress/{stream_id}`

Configure FFmpeg with the `-progress` flag:

```bash
ffmpeg -i input.mp4 \
  -c:v libx264 -preset fast \
  -progress http://localhost:9274/progress/my-stream \
  output.m3u8
```

The progress data is sent as key=value pairs (one per line) in the HTTP request body.

### Progress format
Updates are sent periodically by FFmpeg and look like this:

```bash
# Example progress update from FFmpeg
progress=continue
frame=422
fps=31.25
stream_0_0_q=26.0
stream_0_1_q=28.0
stream_1_0_q=4.3
stream_2_0_q=1.9
bitrate=N/A
total_size=N/A
out_time_us=16800000
out_time_ms=16800000
out_time=00:00:16.800000
dup_frames=0
drop_frames=454
speed=1.24x
```