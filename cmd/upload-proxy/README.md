# retrying proxy for video upload

This is a proxy for uploading segmented streams to one or multiple CDN servers.
It is mainly useful for tools like ffmpeg that do not support retrying http uploads.

Features:
- retries failed uploads
- uploads to multiple servers in parallel