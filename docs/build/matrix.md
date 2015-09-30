# Matrix Builds

Drone uses the `matrix` section of the `.drone.yml` to define the build matrix. Drone executes a build for each permutation in the matrix, allowing you to build and test a single commit against many configurations.

Below is an example `.drone.yml` that tests a single commit against multiple versions of Go and Redis, resulting in a total of 6 different build permutations:

```yaml
build:
  image: golang:$$GO_VERSION
  commands:
    - go get
    - go build
    - go test

compose:
  redis:
    image: redis:$$REDIS_VERSION

matrix:
  GO_VERSION:
    - 1.4
    - 1.3
  REDIS_VERSION:
    - 2.6
    - 2.8
    - 3.0
```

## Matrix Variables

Matrix variables are injected into the `.drone.yml` file using the `$$` syntax, performing a simple find / replace. Matrix variables are also injected into your build container as environment variables.

This is an example `.drone.yml` file before injecting the matrix parameters:

```yaml
build:
  image: golang:$$GO_VERSION
  commands:
    - go get
    - go build
    - go test

compose:
  redis:
    image: redis:$$REDIS_VERSION

    matrix:
      GO_VERSION:
        - 1.4
      REDIS_VERSION:
        - 3.0
```

And this is the `.drone.yml` file after injecting the matrix parameters:

```yaml
build:
  image: golang:1.4
  environment:
    - GO_VERSION=1.4
    - REDIS_VERSION=3.0
  commands:
    - go get
    - go build
    - go test
    compose:
      redis:
        image: redis:3.0
```

## Matrix Deployments

Matrix builds execute the same `.drone.yml` multiple times, but with different parameters. This means that publish and deployment steps are executed multiple times as well, which is typically undesired. To restrict a publish or deployment step to a single permutation you can add the following condition:

```yaml
deploy:
  heroku:
    app: foo
    when:
      matrix:
        GO_VERSION: 1.4
        REDIS_VERSION: 3.0
```
