#!/bin/sh
# uploading ffmpeg client
user=$1
pass=$2
ffmpeg -hide_banner -i rtmp://loopinglouie.fem.tu-ilmenau.de/rtmp/istuff_live -c copy \
    -f dash \
    -index_correction 1 \
    -http_persistent 1 -timeout 2 -use_timeline 1 \
    -window_size 10 -extra_window_size 2 \
    -media_seg_name 'chunk-stream$RepresentationID$-$Number$.$ext$' \
    -auth_type basic \
    -method PUT "http://${user}:${pass}@localhost:8080/dash/live/out.mpd"
