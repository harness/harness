This document provides a brief overview of Drone's build process, so that you can build and run Drone directly from source. For more detail, please see the `.drone.yml` and `Makefile`.

### Requirements
 
* Make
* Go 1.4+
* Libsqlite3

### Build and Test

We use `make` to build and package Drone. You can execute the following commands to build compile Drone locally:

```bash
make deps
make
make test
```

The `all` directive compiles binary files to the `bin/` directory and embeds all static content (ie html, javascript, css files) directly into the binary for simplified distribution.

The `test` directive runs the `go vet` tool for simple linting and executes the suite of unit tests.

**NOTE** if you experience slow compile times you can `go install` the `go-sqlite3` library from the vendored dependencies. This will prevent Drone from re-compiling on every build:

```bash
go install github.com/drone/drone/Godeps/_workspace/src/github.com/mattn/go-sqlite3
```

### Run

To run Drone you can invoke `make run`. This will start Drone with the `--debug` flag which instructs Drone to server static content from disk. This is helpful if you are doing local development and editing the static JavaScript or CSS files.


### Distribute

To generate a debian package:

```bash
make dist
```

To generate a Docker container:

```bash
docker build --rm=true -t drone/drone .
```