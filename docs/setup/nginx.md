# Nginx

Using a proxy server is **not necessary**. Drone serves most static content from a CDN and uses the Go standard library's high-performance net/http package to serve dynamic content.

If using Nginx to proxy traffic to Drone, please ensure you have version 1.3.13 or greater. You also need to configure nginx to write `X-Forwarded_*` headers:

```
location / {
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $remote_addr;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header Host $http_host;
    proxy_set_header Origin "";

    proxy_pass http://127.0.0.1:8000;
    proxy_redirect off;
    proxy_http_version 1.1;
    proxy_buffering off;

    chunked_transfer_encoding off;
}
```

Our installation instructions recommend running Drone in a Docker container with port `:80` published. When behind a reverse proxy, you should run the Drone Docker container with `--publish=8000:8000`. This will publish Drone to port `:8000` allowing you to proxy from `:80`.
