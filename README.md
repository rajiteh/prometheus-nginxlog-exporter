NGINX-to-Prometheus log file exporter
=====================================

Helper tool that continuously reads an NGINX log file and exports metrics to
[Prometheus](prom).

Usage
-----

You can either use a simple configuration, using command-line flags, or create
a configuration file with a more advanced configration.

Use the command-line:

    ./nginx-log-exporter \
      -format="<FORMAT>" \
      -listen-port=4040 \
      -application=nginx \
      [PATHS-TO-LOGFILES...]

Use the configuration file:

    ./nginx-log-exporter -config-file /path/to/config.hcl

Collected metrics
-----------------

This exporter collects the following metrics. This collector can listen on
multiple log files at once and publish metrics in different applications. Each
metric uses the labels `method` (containing the HTTP request method) and
`status` (containing the HTTP status code).

- `nginx_http_response_count_total` - The total amount of processed HTTP requests/responses.
- `nginx_http_response_size_bytes` - The total amount of transferred content in bytes.
- `nginx_http_upstream_time_seconds` - A summary vector of the upstream
  response times in seconds. Logging these needs to be specifically enabled in
  NGINX using the `$upstream_response_time` variable in the log format.
- `nginx_http_response_time_seconds` - A summary vector of the total
  response times in seconds. Logging these needs to be specifically enabled in
  NGINX using the `$request_time` variable in the log format.

Additional labels can be configured in the configuration file (see below).

Configuration file
------------------

You can specify a configuration file to read at startup. The configuration file
is expected to be in [HCL](hcl) format. Here's an example file:

```hcl
listen {
  port = 4040
  address = "10.1.2.3"
}

application "app-1" {
  format = "$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\" \"$http_x_forwarded_for\""
  log_files = [
    "/var/log/nginx/app1/access.log"
  ]
  labels {
    app = "application-one"
    environment = "production"
    foo = "bar"
  }
}

application "app-2" {
  format = "$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\" \"$http_x_forwarded_for\" $upstream_response_time"
  log_files = [
    "/var/log/nginx/app2/access.log"
  ]
}
```

Running the collector
---------------------

### Systemd

You can find an example unit file for this service [in this repository](systemd/prometheus-nginxlog-exporter.service). Simply copy the unit file to `/etc/systemd/system`:

    $ wget -O /etc/systemd/system/prometheus-nginxlog-exporter.service https://raw.githubusercontent.com/martin-helmich/prometheus-nginxlog-exporter/master/systemd/prometheus-nginxlog-exporter.service
    $ systemctl enable prometheus-nginxlog-exporter
    $ systemctl start prometheus-nginxlog-exporter

The shipped unit file expects the binary to be located in `/usr/local/bin/prometheus-nginxlog-exporter` and the configuration file in `/etc/prometheus-nginxlog-exporter.hcl`. Adjust to your own needs.

Credits
-------

- [tail](https://github.com/hpcloud/tail), MIT license
- [gonx](https://github.com/satyrius/gonx), MIT license
- [Prometheus Go client library](https://github.com/prometheus/client_golang), Apache License
- [HashiCorp configuration language](hcl), Mozilla Public License

[prom]: https://prometheus.io/
[hcl]: https://github.com/hashicorp/hcl
