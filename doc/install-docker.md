An official Drone image is available in the [Docker registry](https://registry.hub.docker.com/u/drone/drone/). This is the recommended way to install and run Drone on non-Ubuntu environments. Pull the latest Drone image to get started:

```bash
sudo docker pull drone/drone
```

An example command to run your Drone instance with GitHub enabled:

```bash
sudo docker run -d \
	-v /var/lib/drone:/var/lib/drone \
	-e DRONE_GITHUB_CLIENT=c0aaff74c060ff4a950d \
	-e DRONE_GITHUB_SECRET=1ac1eae5ff1b490892f5 \
	-p 80:80 --name=drone drone/drone
```

### Persistence

When running Drone inside Docker we recommend mounting a volume for your sqlite database. This ensures the database is not lost if the container is accidentally stopped and removed. Below is an example that mounts `/var/lib/drone` on your host machine:

```bash
sudo docker run \
	-v /var/lib/drone:/var/lib/drone \
	--name=drone drone/drone
```

### Configuration

When running Drone inside Docker we recommend using environment variables to configure the system. All configuration attributes in the `drone.toml` may also be provided as environment variables. Below demonstrates how to configure GitHub from environment variables:

```bash
sudo docker run \
	-e DRONE_GITHUB_CLIENT=c0aaff74c060ff4a950d \
	-e DRONE_GITHUB_SECRET=1ac1eae5ff1b490892f5 \
	--name=drone drone/drone
```

### Logging

When running Drone inside Docker the logs are sent to stdout / stderr. You can view the log output by running the following command:

```bash
docker logs drone
```
