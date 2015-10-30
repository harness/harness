# Docker

Drone uses the local Docker daemon (at `unix:///var/run/docker.sock`) to execute your builds with 2x concurrency. This section describes how to customize your Docker configuration and concurrency settings using the `DOCKER_*` environment variables.

Configure a single Docker host (1x build concurrency):

```bash
DOCKER_HOST="unix:///var/run/docker.sock"
```

## Concurrency

Configure Drone to run multiple, concurrent builds by increasing the number of registered Docker hosts. Each `DOCKER_HOST_*` environment variable will increase concurrency by 1.

Configure multiple Docker hosts (4x build concurrency):

```bash
DOCKER_HOST_1="unix:///var/run/docker.sock"
DOCKER_HOST_2="unix:///var/run/docker.sock"
DOCKER_HOST_3="unix:///var/run/docker.sock"
DOCKER_HOST_4="unix:///var/run/docker.sock"
```

Configure a single, external Docker host (1x build concurrency):

```bash
DOCKER_HOST="tcp://1.2.3.4:2376"

DOCKER_CA="/path/to/ca.pem"
DOCKER_CERT="/path/to/cert.pem"
DOCKER_KEY="/path/to/key.pem"
```

Configure multiple, external Docker hosts (4x build concurrency using 2 remote servers):

```bash
DOCKER_HOST_1="tcp://1.2.3.4:2376"
DOCKER_HOST_2="tcp://1.2.3.4:2376"

DOCKER_HOST_3="tcp://4.3.2.1:2376"
DOCKER_HOST_4="tcp://4.3.2.1:2376"

DOCKER_CA="/path/to/ca.pem"
DOCKER_CERT="/path/to/cert.pem"
DOCKER_KEY="/path/to/key.pem"
```

## Remote Servers

Connecting to remote Docker servers requires TLS authentication for security reasons. You will therefore need to generate your own self-signed certificates. For convenience, we've created the following gist to help generate a certificate: https://gist.github.com/bradrydzewski/a6090115b3fecfc25280

This will generate the following files:

* ca.pem
* cert.pem
* key.pem
* server-cert.pem
* server-key.pem

Tell Drone where to find the `cert.pem` and `key.pem`:

```bash
DOCKER_CERT="/path/to/cert.pem"
DOCKER_KEY="/path/to/key.pem"
```

When running Drone inside Docker, you'll need to mount the volume containing the certificate:

```bash
docker run
    --volume /path/to/cert.pem:/path/to/cert.pem \
    --volume /path/to/key.pem:/path/to/key.pem   \
```

Tell Docker where to find the certificate files. Install the certificates on every remote machine (in `/etc/ssl/docker/`) and update each Docker configuration file (at `/etc/init/drone-dart.conf`) accordingly:

```bash
# Use DOCKER_OPTS to modify the daemon startup options.
DOCKER_OPTS="--tlsverify --tlscacert=/etc/ssl/docker/ca.pem --tlscert=/etc/ssl/docker/server-cert.pem --tlskey=/etc/ssl/docker/server-key.pem -H=0.0.0.0:2376 -H unix:///var/run/docker.sock"
```

Verify that everything is configured correctly by connecting to a remote Docker server from our Drone server using the following command:

```bash
sudo docker                     \
    --tls                       \
    --tlscacert=/path/to/ca.pem \
    --tlscert=/path/to/cert.pem \
    --tlskey=/path/to/key.pem   \
    -H="1.2.3.4:2376" version
```
