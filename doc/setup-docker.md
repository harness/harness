Drone is configured to connect to the local Docker daemon. Drone will attempt to use the `DOCKER_HOST` environment variable to determine the daemon URL. If not set, Drone will attempt to use the default socket connection `unix:///var/run/docker.sock`.

You can modify the Docker daemon URL in the Drone configuration file:

```ini
[docker] 
nodes=[ 
   "unix:///var/run/docker.sock",
   "unix:///var/run/docker.sock"
]
```

### Concurrency

Each node is capable of processing a single build. Therefore, the below configuration will only execute one build at a time:

```ini
[docker] 
nodes=[ 
   "unix:///var/run/docker.sock"
]
```

In order to increase concurrency you can increase the number of nodes. The below configuration is capable of processing four builds at a time, all using the local Docker daemon:

```ini
[docker] 
nodes=[ 
   "unix:///var/run/docker.sock",
   "unix:///var/run/docker.sock",
   "unix:///var/run/docker.sock",
   "unix:///var/run/docker.sock"
]
```

### Distribution

As your installation grows you may need to distribute your builds across multiple servers. Since Docker exposes a REST API we can easily configure Drone to communicate with remote servers. First we'll need to generate an SSL certificate in order to secure communication across nodes.

We recommend using this Gist to generate keys:
https://gist.github.com/bradrydzewski/a6090115b3fecfc25280

This will generate the following files:

* ca.pem
* cert.pem 
* key.pem 
* server-cert.pem 
* server-key.pem

Update your Drone configuration to use the `cert.pem` and `key.pem` files and remote daemon URLs:

```ini
[docker]
cert="/path/to/cert.pem" 
key="/path/to/key.pem"
nodes = [
  "tcp://172.17.42.1:2376",
  "tcp://172.17.42.2:2376",
  "tcp://172.17.42.3:2376",
  "tcp://172.17.42.4:2376"
]
```

> Remember that you can add the same URL multiple times to increase concurrency!

Finally, you need to place the server key, certificate and ca on each remote server. You'll need to update the Docker daemon configuration on each remote server (in `/etc/init/drone-dart.conf`) and restart Docker:

```bash
# Use DOCKER_OPTS to modify the daemon startup options.
DOCKER_OPTS="--tlsverify --tlscacert=/etc/ssl/docker/ca.pem --tlscert=/etc/ssl/docker/server-cert.pem --tlskey=/etc/ssl/docker/server-key.pem -H=0.0.0.0:2376 -H unix:///var/run/docker.sock"
```

Lastly, we can verify that everything is configured correctly. We can try to connect to a remote Docker server from our Drone server using the following command:

```bash
sudo docker                     \
    --tls                       \
    --tlscacert=/path/to/ca.pem \
    --tlscert=/path/to/cert.pem \
    --tlskey=/path/to/key.pem   \
    -H=1.2.3.4:2376 version
```
