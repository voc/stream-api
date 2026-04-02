## Edge stage
The edge machines run nginx to serve the HTTP streams to end users. 

### Origin Discovery
The edge machines run consul-template to discover the available origin servers from the Consul backend and update the nginx configuration accordingly.

### Edge Monitoring
They also run a local telegraf to scrape metrics from nginx and forward them to the monitoring system.