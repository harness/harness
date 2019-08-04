Drone is a Continuous Delivery system built on container technology. Drone uses a simple YAML configuration file, a superset of docker-compose, to define and execute Pipelines inside Docker containers. 

<br/>

<img src="https://github.com/drone/brand/blob/master/screenshots/screenshot_build_success.png" style="max-width:100px;" />

Sample Pipeline Configuration:

```yaml
pipeline:
  backend:
    image: golang
    commands:
      - go get
      - go build
      - go test

  frontend:
    image: node:6
    commands:
      - npm install
      - npm test

  publish:
    image: plugins/docker
    repo: octocat/hello-world
    tags: [ 1, 1.1, latest ]
    registry: index.docker.io

  notify:
    image: plugins/slack
    channel: developers
    username: drone
```

Documentation and Other Links:

* Setup Documentation [docs.drone.io/installation](http://docs.drone.io/installation/)
* Usage Documentation [docs.drone.io/user-guide](https://docs.drone.io/user-guide/)
* Plugin Index [plugins.drone.io](http://plugins.drone.io/)
* Getting Help [discourse.drone.io](https://discourse.drone.io)
