#!/bin/sh

# only execute this script as part of the pipeline.
[ -z $CI ] && echo "missing ci enviornment variable" && exit 2

# only execute the script when github token exists.
[ -z $SSH_KEY ] && echo "missing ssh key" && exit 3

# write a netrc file for authorization.
mkdir /root/.ssh
echo -n "$SSH_KEY" > /root/.ssh/id_rsa
chmod 600 /root/.ssh/id_rsa

# clone the extras project.
set -e
set -x
git clone git@github.com:drone/drone-enterprise.git extras

# build a static binary with the build number and extra features.
go build -ldflags '-extldflags "-static" -X github.com/drone/drone/version.VersionDev=build.${DRONE_BUILD_NUMBER}' -tags extras -o release/drone github.com/drone/drone/drone
