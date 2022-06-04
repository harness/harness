# [Drone](https://www.drone.io/) <img src="https://github.com/drone/brand/blob/master/screenshots/screenshot_build_success.png" style="max-width:100px;" />
**Welcome to the Drone codebase, we are thrilled to have you here!** 

## What is Drone?
Drone is a continuous delivery system built on container technology. Drone uses a simple YAML build file, to define and execute build pipelines inside Docker containers. 
 
## Table of Contents

- [What is Drone?](#what-is-drone)
- [Table of Contents](#table-of-contents)
- [Community and Support](#community-and-support)
- [Contributing](#contributing)
- [Code of Conduct](#code-of-conduct)
- [Setup Documentation](#setup-documentation)
- [Usage Documentation](#usage-documentation)
- [Example `.drone.yml` build file](#example-droneyml-build-file)
- [Plugin Index](#plugin-index)
- [Documentation and Other Links](#documentation-and-Other-Links)

## Community and Support
[Harness Community Slack](https://join.slack.com/t/harnesscommunity/shared_invite/zt-y4hdqh7p-RVuEQyIl5Hcx4Ck8VCvzBw) - Join the #drone slack channel to connect with our engineers and other users running Drone CI.
</br>
[Harness Community Forum](https://community.harness.io/) - Ask questions, find answers, and help other users.
</br>
[Report A Bug](https://community.harness.io/c/bugs/17) - Find a bug? Please report in our forum under Drone Bugs. Please provide screenshots and steps to reproduce. 
</br>
[Events](https://www.meetup.com/harness/) - Keep up to date with Drone events and check out previous events [here](https://www.youtube.com/watch?v=Oq34ImUGcHA&list=PLXsYHFsLmqf3zwelQDAKoVNmLeqcVsD9o).

## Contributing

We encourage you to contribute to Drone! Whether that's joining in on the community slack or discourse, or contributing pull requests / documentation changes or raising issues.

## Code of Conduct

Drone follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).

### Setup Documentation

This section of the [documentation](http://docs.drone.io/installation/) will help you install and configure the Drone Server and one or many Runners. A runner is a standalone daemon that polls the server for pending pipelines to execute.

### Usage Documentation

Our [documentation](http://docs.drone.io/getting-started/) can help you get started with the different types of pipelines/builds. There are different runners / plugins / extensions designed for different use cases to help make an efficient and simple build pipeline

### Plugin Index

Plugins are used in build steps to perform actions, eg send a message to slack or push a container to a registry. We have an extensive list of community plugins to customize your build pipeline, you can find those [here](http://plugins.drone.io/).

### Example `.drone.yml` build file. 

This build file contains a single pipeline (you can have multiple pipelines too) that builds a go application. The front end with npm. Publishes the docker container to a registry and announces the results to a slack room.

```YAML
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

## Documentation and Other Links

* Setup Documentation [docs.drone.io/installation](http://docs.drone.io/installation/)
* Usage Documentation [docs.drone.io/getting-started](http://docs.drone.io/getting-started/)
* Plugin Index [plugins.drone.io](http://plugins.drone.io/)
* Getting Help [discourse.drone.io](https://discourse.drone.io)
* Build the Enterprise Edition [BUILDING](https://github.com/drone/drone/blob/master/BUILDING)
* Build the Community Edition [BUILDING_OSS](https://github.com/drone/drone/blob/master/BUILDING_OSS)

## Building from source

We have two versions available: the [Enterprise Edition](https://github.com/drone/drone/blob/master/BUILDING) and the [Community Edition](https://github.com/drone/drone/blob/master/BUILDING_OSS)

## Release procedure

Run the changelog generator.

```BASH
docker run -it --rm -v "$(pwd)":/usr/local/src/your-app githubchangeloggenerator/github-changelog-generator -u drone -p drone -t <secret github token>
```

You can generate a token by logging into your GitHub account and going to Settings -> Personal access tokens.

Next we tag the PR's with the fixes or enhancements labels. If the PR does not fulfill the requirements, do not add a label.

**Before moving on make sure to update the version file `version/version.go && version/version_test.go`.**

Run the changelog generator again with the future version according to semver.

```BASH
docker run -it --rm -v "$(pwd)":/usr/local/src/your-app githubchangeloggenerator/github-changelog-generator -u harness -p drone -t <secret token> --future-release v1.0.0
```

Create your pull request for the release. Get it merged then tag the release.

[⬆ Back to Top](#table-of-contents)
