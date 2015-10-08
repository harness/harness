# Overview

In order to configure your build, you must include a `.drone.yml` file in the root of your repository. This section provides a brief overview of the `.drone.yml` configuration file format.

Example `.drone.yml` for a Go repository:

```yaml
build:
  image: golang
  commands:
    - go get
    - go build
    - go test
```

A more comprehensive example with linked service containers, deployment plugins and notification plugins:

```yaml
build:
  image: golang
  commands:
    - go get
    - go build
    - go test

compose:
  cache:
    image: redis
  database:
    image: mysql

deploy:
  heroku:
    app: pied_piper
    when:
      branch: master

notify:
  slack:
    channel: myteam
```
