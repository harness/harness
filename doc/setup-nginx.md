Using a proxy server is not really necessary. Drone serves most static content from a CDN and uses the Go standard library’s high-performance net/http package to serve dynamic content. If using Nginx to proxy traffic to Drone you may use the following configuration:

```nginx
location / {
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $remote_addr;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header Host $http_host;
    proxy_set_header Origin "";

    proxy_pass http://127.0.0.1:8000;
    proxy_redirect off;
}
```

You may also want to change Drone’s default port when proxying traffic. You can change the port in the `/etc/drone/drone.toml` configuration file:

```toml
[server]
addr = ":8000"
```
