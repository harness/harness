# Upgrading

> Warning. There is no automated migration from 0.3 to 0.4 due to substantial changes in database structure.

Drone is built continuously, with updates available daily. In order to upgrade Drone you must first stop and remove your running Drone instance:

```
sudo docker stop drone
sudo docker rm drone
```

Pull the latest Drone image:

```
sudo docker pull drone/drone:0.4
```

Re-run the container using the latest Drone image:

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