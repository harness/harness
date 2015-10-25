# Machine Management

The `drone node create` command lets your register new remote servers with Drone. This command should be used in conjunction with [docker-machine](https://github.com/docker/machine). Note that you can alternatively register and manage servers in the UI.

## Environment Variables

The `drone node create` command expects the following environment variables:

* `DOCKER_HOST` - docker deamon address
* `DOCKER_TLS_VERIFY` - docker daemon supports tlsverify
* `DOCKER_CERT_PATH` - docker certificate directory

## Instructions

Create or configure a new server using `docker-machine`:

```
docker-machine create \
    --digitalocean-size 2gb \
    --driver digitalocean \
    --digitalocean-access-token <token> \
    my-drone-worker
```

This writes setup instructions to the console:

```
export DOCKER_TLS_VERIFY="1"
export DOCKER_HOST="tcp://123.456.789.10:2376"
export DOCKER_CERT_PATH="/home/octocat/.docker/machine/machines/my-drone-worker"
export DOCKER_MACHINE_NAME="my-drone-worker"
# Run this command to configure your shell: 
# eval "$(docker-machine env my-drone-worker)"
```

Run the following command (from the above output) to configure your shell:

```
eval "$(docker-machine env my-drone-worker)"
```

Register the newly created or configured machine with Drone. Once registered, Drone will immediately begin sending builds to the server.

```
drone node create
```
