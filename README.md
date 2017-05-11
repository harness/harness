[![Build Status](http://beta.drone.io/api/badges/drone/drone/status.svg)](http://beta.drone.io/drone/drone)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/drone/drone?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

Drone is a Continuous Integration platform built on container technology. Every build is executed inside an ephemeral Docker container, giving developers complete control over their build environment with guaranteed isolation.

Browse the code at https://sourcegraph.com/github.com/drone/drone

### Documentation

Documentation is published to [docs.drone.io](http://docs.drone.io)

### Community, Help

Contributions, questions, and comments are welcomed and encouraged. We ask that you please post user support questions to the [community forum](http://discourse.drone.io/), before creating Github issues. Drone developers hang out in the [drone/drone](https://gitter.im/drone/drone) room on Gitter.

### Installation

Please see our [installation guide](http://docs.drone.io/installation/) to install the official Docker image.

### From Source

Clone the repository to your Go workspace:

```
export PATH=$PATH:$GOPATH/bin

git clone git://github.com/drone/drone.git $GOPATH/src/github.com/drone/drone
cd $GOPATH/src/github.com/drone/drone
```

Commands to build from source:

```sh
make deps          # Download required dependencies
make gen           # Generate code
make build_static  # Build the binary
```

If you are having trouble building this project please reference its `.drone.yml` file. Everything you need to know about building Drone is defined in that file.
