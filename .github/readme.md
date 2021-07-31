# [Drone](https://www.drone.io/) <img src="https://github.com/drone/brand/blob/master/screenshots/screenshot_build_success.png" style="max-width:100px;" />
**Welcome to the Drone codebase, we are thrilled to have you here!** 


## What is Drone?
Drone is a continuous delivery system built on container technology. Drone uses a simple YAML configuration file, a superset of docker-compose, to define and execute Pipelines inside Docker containers. we have two versions available: the [Enterprise Edition](https://github.com/drone/drone/blob/master/BUILDING) and the [Community Edition](https://github.com/drone/drone/blob/master/BUILDING_OSS)

_Please note the official Docker images run the Drone Enterprise distribution. If you would like to run the Community Edition you can build from source by following the instructions in [BUILDING_OSS](https://github.com/drone/drone/blob/master/BUILDING_OSS)._
 
## Table of Contents

- [What is Drone?](#what-is-drone)
- [Table of Contents](#table-of-contents)
- [Community](#community)
- [Contributing](#contributing)
- [Code of Conduct](#code-of-conduct)
- [Core Team](#core-team)
- [Setup Documentation](#setup-documentation)
- [Usage Documentation](#usage-documentation)
- [Sample Pipeline Configuration](#sample-pipeline-configuration)
- [Plugin Index](#plugin-index)

## Community

We have a place to voice your ideas, have discussion on features, or get help with any issue you may have, please visit the community on our [Discourse](https://discourse.drone.io/) as well as our [Slack](https://join.slack.com/t/harnesscommunity/shared_invite/zt-90wb0w6u-OATJvUBkSDR3W9oYX7D~4A).


## Contributing

We encourage you to contribute to Drone! Please check out our [Proposing Changes Guide](https://github.com/drone/proposal).

## Code of Conduct

Drone follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).

## Core Team

- Brad Rydzewski
- TP Honey
- Marko Gacesa
- Eoin McAfee
- Dan Wilson
- Marie Antons

### Setup Documentation

This section of the [documentation](http://docs.drone.io/installation/) will help you install and configure the Drone Server and one or many Runners. A runner is a standalone daemon that polls the server for pending pipelines to execute.

### Usage Documentation

Pipelines help you automate steps in your software delivery process, such as initiating code builds, running automated tests, and deploying to a staging or production environment. Our [documentation](http://docs.drone.io/getting-started/) can help you get started with the different types of pipelines. Each optimized for different use cases and runtime environments.


### Plugin Index

We have an extensive registry of community plugins to customize your continuous delivery pipeline, you can find those [here](http://plugins.drone.io/).

<br>
<img src="https://github.com/drone/brand/blob/master/screenshots/screenshot_build_success.png" style="max-width:100px;" />

### Sample Pipeline Configuration

```yaml
name: default

kind: pipeline
type: docker

steps:
- name: backend
  image: golang
  commands:
    - go get
    - go build
    - go test

- name: frontend
  image: node:6
  commands:
    - npm install
    - npm test

- name: publish
  image: plugins/docker
  settings:
    repo: octocat/hello-world
    tags: [ 1, 1.1, latest ]
    registry: index.docker.io

- name: notify
  image: plugins/slack
  settings:
    channel: developers
    username: drone
```

[⬆ Back to Top](#table-of-contents)