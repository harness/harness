# Server

Drone uses the `net/http` package in the Go standard library for high-performance `http` request processing. This section describes how to customize the default server configuration. This section is completely **optional**.

## Server Settings

This section lists all environment variables used to configure the server.

* `SERVER_ADDR` server address and port. Defaults to `:8000`
* `SERVER_KEY` ssl certificate key (key.pem)
* `SERVER_CERT` ssl certificate (cert.pem)

This example changes the default port to `:80`:

```bash
SERVER_ADDR=:80
```

## Server SSL

Drone uses the `ListenAndServeTLS` function in the Go standard library to accept `https` connections. If you experience any issues configuring `https` please contact us on [gitter](https://gitter.im/drone/drone). Please do not log an issue saying `https` is broken in Drone.

This example accepts `HTTPS` connections:

```bash
SERVER_ADDR=:443
SERVER_KEY=/path/to/key.pem
SERVER_CERT=/path/to/cert.pem
```

> When your certificate is signed by an authority, the certificate should be the concatenation of the server's certificate followed by the CA certificate.

When running Drone inside Docker, you'll need to mount a volume containing the certificate:

```bash
docker run
    --volume /path/to/cert.pem:/path/to/cert.pem \
    --volume /path/to/key.pem:/path/to/key.pem   \
```
