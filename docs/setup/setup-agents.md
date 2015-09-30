> **NOTE** this is an advanced feature and should not be required for most installs

If you are running a large build cluster (25+ servers) we recommend using build agents. A build agent is a daemon that is installed on each build server that polls the central Drone server for work. You can add and remove build agents at any time, without having to configure and re-start the central Drone server. This is a great option if you need to frequently scale your build infrastrucutre up or down by adding or removing servers.

You will need to configure the `/etc/drone/drone.toml` to enable build agents. This includes specifying a secret token (ie password) that will allow agents to authenticate to the central Drone server to poll for builds:

```ini
[agents]
secret = "c0aaff74c060ff4a950d"
```

And then on each build server, pull the Drone agent image:

```bash
docker pull drone/drone-agent
```

And start the Drone agent:

```bash
docker run -d 
	-v /var/run/docker.sock:/var/run/docker.sock
	-p 1991:1991 drone/drone-agent \
	--addr=http://localhost:8000 \
	--token=c0aaff74c060ff4a950d \
```

The Drone agent is started with port `1991` exposed allowing the central Drone server to communicate directly with the agent. We also mount the Docker socket, `/var/run/docker.sock`, allowing the agent to spawn build containers.

The Drone agent also requires the following command line arguments:

* **addr** address of the running Drone server
* **token** token used to authorize requestests to the Drone server

