# Build

Drone uses the `build` section of the `.drone.yml` to describe your Docker build environment and your build and test instructions. The following is an example build definition:

```yaml
build:
  image: golang
  environment:
    - GOBIN=/drone/bin
  commands:
    - go get
    - go build
    - go test
```

## Build options

The build configuration options:

* `image` - any valid Docker image name
* `pull` - if true, will always attempt to pull the latest image
* `environment` - list of environment variables, declared in `name=value` format
* `privileged` - if true, runs the container with extended privileges [1]
* `volumes` - list of bind mounted volumes on the host machine [1]
* `net` - sets the container [network mode](https://docs.docker.com/articles/networking/#container-networking) [1]
* `commands` - list of build commands

[1] Some build options are disabled for security reasons, including `volumes`, `privileged` and `net`. To enable these options, a system administrator must white-list your repository as trusted. This can be done via the repository settings screen.

## Build image

The `image` attribute supports any valid Docker image name:

```yaml
# Docker library image
image: golang

# Docker library image, with tag
image: golang:1.4

# Docker image, full name, with tag
image: library/golang:1.4

# fully qualified Docker image URI, with tag
image: index.docker.io/library/golang:1.4
```

## Skipping builds

Skip a build by including the text `[CI SKIP]` in your commit message.
