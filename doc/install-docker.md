# Installation

To quickly tryout Drone we have a [Docker image](https://registry.hub.docker.com/u/drone/drone/) that includes everything you need to get started. Simply run the commend below, substituted with your GitHub credentials:

```bash
sudo docker run \
	--volume /var/lib/drone:/var/lib/drone \
	--volume /var/run/docker.sock:/var/run/docker.sock \
	--env DRONE_REMOTE="github://client_id=1ac1eae5ff1b490892f5&client_secret=c0aaff74c060ff4a950d" \
	--restart=always \
	--publish=80:8000 \
	--detach=true \
	--name=drone \
	drone/drone:latest
```

Drone is now running (in the background) on `http://localhost:80`

## Docker options

Here are some of the Docker options, explained:

* `--restart=always` starts Drone automatically during system init process
* `--publish=80:8000` runs Drone on port `80`
* `--detach=true` starts Drone in the background
* `--volume /var/lib/drone:/var/lib/drone` mounted volume to persist sqlite database
* `--volume /var/run/docker.sock:/var/run/docker.sock` mounted volume to access Docker and spawn builds

## Drone settings

Drone uses environment variables for runtime settings and configuration, such as GitHub, GitLab, plugins and more. These settings can be provided to Docker using `--env` command as seen above.

## Standalone Install

Running Drone inside Docker is recommended but by no means required. Drone compiles to a single binary file with zero dependencies. To simplify support, however, we no longer ship a binary distribution and encourage everyone to run Drone inside Docker.
