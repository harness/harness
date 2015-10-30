# Install

> This is an early preview of the command line utility. Contributors wanted.

Drone provides a simple command line utility that allows you interact with the Drone server from the command line. This section describes the setup and installation process.

## System Requirements

This tool requires Docker 1.6 or higher. If you are using Windows or Mac we recommend installing the [Docker Toolbox](https://www.docker.com/docker-toolbox).

## Linux

Download and install the x64 linux binary:

```
curl http://downloads.drone.io/drone-cli/drone_linux_amd64.tar.gz | tar zx
sudo install -t /usr/local/bin drone
```

## OSX

Download and install using Homebrew:

```
brew tap drone/drone
brew install drone
```

Or manually download and install the binary:

```
curl http://downloads.drone.io/drone-cli/drone_darwin_amd64.tar.gz | tar zx
sudo cp drone /usr/local/bin
```

## Setup

In order to communicate with the Drone server you must provide the server url:

```
export DRONE_SERVER=<http://>
```

In order to authorize communications you must also provide your access token:

```
export DRONE_TOKEN=<token>
```

You can retrieve your access token from the user profile screen in Drone.
