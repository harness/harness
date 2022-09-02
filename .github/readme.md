# [Drone](https://www.drone.io/) <img src="https://github.com/drone/brand/blob/master/screenshots/screenshot_build_success.png" style="max-width:100px;" />

**Welcome to the Drone codebase, we are thrilled to have you here!**

## Table of Contents

- [What is Drone?](#what-is-drone)
- [Contributing to Drone](#contributing-to-drone)
- [Community and Support](#community-and-support)
- [Documentation and Other Links](#documentation-and-other-links)
- [Example `.drone.yml` build file](#example-droneyml-build-file)

## What is Drone?

Drone is a continuous delivery system built on container technology. Drone uses a simple YAML build file to define and execute build pipelines inside Docker containers.

## Contributing To Drone

Drone's codebase is opensource and any kind of contribution is highly encouraged. Here are some resources to get you stsrted

- Find out the rules before you contribute [here](https://github.com/harness/drone/blob/0c764871d7a86bb92855a1776f6a0e0db061fec4/.github/contributing.md).
- Before you raise your PR please checkout the Pull Request template [here](https://github.com/harness/drone/blob/0c764871d7a86bb92855a1776f6a0e0db061fec4/.github/pull_request_template.md).
- If you come up with a bug related to security then feel free to go through the Security Policies and Procedures [here](https://github.com/harness/drone/blob/0c764871d7a86bb92855a1776f6a0e0db061fec4/.github/security.md).

### Release Procedure:

  Run the changelog generator.

  ```BASH
  docker run -it --rm -v "$(pwd)":/usr/local/src/your-app githubchangeloggenerator/github-changelog-generator -u drone -p drone -t <secret github token>
  ```

  You can generate a token by logging into your GitHub account and going to Settings -> Personal access tokens.

  Next we tag the PR's with the fixes or enhancements labels. If the PR does not fulfill the requirements, do not add a label.

  _Before moving on make sure to update the version file_ `version/version.go && version/version_test.go`.

  Run the changelog generator again with the future version according to semver.

  ```BASH
  docker run -it --rm -v "$(pwd)":/usr/local/src/your-app githubchangeloggenerator/github-changelog-generator -u harness -p drone -t <secret token> --future-release v1.0.0
  ```

  Create your pull request for the release. Get it merged then tag the release.

### Code Of Conduct:

Drone followes the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md).

## Documentation and Other Links

- [Setup Documentation](http://docs.drone.io/installation/) will help you install and configure the Drone Server and one or many Runners.<br>
- [Usage Documentation](http://docs.drone.io/getting-started/) can help you get started with the different types of pipelines/builds.
- [Plugin Index](http://plugins.drone.io/) shows you an extensive list of community plugins to customize your build pipeline.
- You can get help from [here](https://discourse.drone.io).
- We have two versions. To build the Enterprise Edition, look over [here](https://github.com/drone/drone/blob/master/BUILDING) and to build the Community Edition, go through [this](https://github.com/drone/drone/blob/master/BUILDING_OSS)

### Example `.drone.yml` build file

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

## Community and Support

[Harness Community Slack](https://join.slack.com/t/harnesscommunity/shared_invite/zt-y4hdqh7p-RVuEQyIl5Hcx4Ck8VCvzBw) - Join the #drone slack channel to connect with our engineers.
</br>
[Harness Community Forum](https://community.harness.io/) - Ask questions, find answers, and help other users.
</br>
[Report A Bug](https://community.harness.io/c/bugs/17) - Found a bug? Please report in our forum under Drone Bugs. Please provide screenshots and steps to reproduce.
</br>
[Events](https://www.meetup.com/harness/) - Keep up to date with Drone events and check out previous events [here](https://www.youtube.com/watch?v=Oq34ImUGcHA&list=PLXsYHFsLmqf3zwelQDAKoVNmLeqcVsD9o).

[⬆ Back to Top](#table-of-contents)
