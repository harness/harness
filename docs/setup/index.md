# Installation

To quickly tryout Drone we have a [Docker image](https://registry.hub.docker.com/u/drone/drone/) that includes everything you need to get started. Simply run the commend below:

```
sudo docker pull drone/drone:0.4
```

And then run the container:

```
sudo docker run \
	--volume /var/lib/drone:/var/lib/drone \
	--volume /var/run/docker.sock:/var/run/docker.sock \
	--env-file /etc/drone/dronerc \
	--restart=always \
	--publish=80:8000 \
	--detach=true \
	--name=drone \
	drone/drone:0.4
```

Drone is now running (in the background) on `http://localhost:80`. Note that before running you should create the `--env-file` and add your Drone configuration (GitHub, Bitbucket, GitLab credentials, etc).

## Docker options

Here are some of the Docker options, explained:

* `--restart=always` starts Drone automatically during system init process
* `--publish=80:8000` runs Drone on port `80`
* `--detach=true` starts Drone in the background
* `--volume /var/lib/drone:/var/lib/drone` mounted volume to persist sqlite database
* `--volume /var/run/docker.sock:/var/run/docker.sock` mounted volume to access Docker and spawn builds
* `--env-file /etc/defaults/drone` loads an external file with environment variables. Used to configure Drone.

## Drone settings

Drone uses environment variables for runtime settings and configuration, such as GitHub, GitLab, plugins and more. These settings can be provided to Docker using an `--env-file` as seen above.

## Starting, Stopping, Logs

Commands to start, stop and restart Drone:

```
sudo docker start drone
sudo docker stop drone
sudo docker restart drone
```

And to view the Drone logs:

```
sudo docker logs drone
```
