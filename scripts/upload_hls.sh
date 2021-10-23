#!/bin/sh
# uploading ffmpeg client
user=$1
pass=$2
ffmpeg -hide_banner -i rtmp://loopinglouie.fem.tu-ilmenau.de/rtmp/istuff_live -c copy \
    -f hls \
    -http_persistent 1 -timeout 2 \
    -hls_flags delete_segments -auth_type basic \
    -method PUT "http://${user}:${pass}@localhost:8080/live/out.m3u8"
