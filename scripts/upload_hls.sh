#!/bin/sh
# uploading ffmpeg client
ffmpeg -hide_banner -i rtmp://loopinglouie.fem.tu-ilmenau.de/rtmp/istuff_live -c copy \
    -map 0:a -map 0:v -map 0:a -map 0:v \
    -b:v 2000k \
    -f hls \
    -http_persistent 1 \
    -hls_flags delete_segments \
    -master_pl_name native_hd.m3u8 \
    -var_stream_map "v:0,a:0 v:1,a:1" \
    -master_pl_publish_rate 10 \
    -method PUT 'http://localhost:8080/hls/live/out_%v.m3u8'
