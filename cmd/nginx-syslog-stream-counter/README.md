# Usage
```
log_format json_logs escape=json '{'
    '"time_local": "$time_local",'
    '"remote_addr": "$remote_addr",'                            # client IP
    '"method": "$request_method",'                              # request method, usually “GET” or “POST” 
    '"protocol": "$server_protocol",'                           # request protocol, usually “HTTP/1.0”, “HTTP/1.1”, “HTTP/2.0”, or “HTTP/3.0” 
    '"uri": "$uri",'                                            # current URI in request
    '"status": "$status",'                                      # response status code
    '"bytes_sent": "$bytes_sent", '                             # the number of bytes sent to a client
    '"request_length": "$request_length", '                     # request length (including headers and body)
    '"connection_requests": "$connection_requests",'            # number of requests made in connection
    '"upstream": "$upstream_addr", '                            # upstream backend server for proxied requests
    '"upstream_connect_time": "$upstream_connect_time", '       # upstream handshake time incl. TLS
    '"upstream_header_time": "$upstream_header_time", '         # time spent receiving upstream headers
    '"upstream_response_time": "$upstream_response_time", '     # time spend receiving upstream body
    '"upstream_response_length": "$upstream_response_length", ' # upstream response length
    '"upstream_cache_status": "$upstream_cache_status", '       # cache HIT/MISS where applicable
    '"ssl_protocol": "$ssl_protocol", '                         # TLS protocol
    '"ssl_cipher": "$ssl_cipher", '                             # TLS cipher
    '"scheme": "$scheme", '                                     # http or https
    '"user_agent": "$http_user_agent"'
'}';

access_log syslog:server=unix:/var/log/relay.sock json_logs;
```

# Customization
The code must be able to derive the sub playlist from the request path. 
