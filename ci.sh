#!/bin/sh
set -e

# only execute this script as part of the pipeline.
[ -z $CI ] && exit 1

# only execute the script when github token exists.
[ -z $SSH_KEY ] && exit 1

# write a netrc file for authorization.
mkdir -p $HOME/.ssh
echo -n "$SSH_KEY" > $HOME/.ssh/id_rsa
chmod 600 $HOME/.netrc

# clone the extras project.
set +x
git clone git@github.com:drone/drone-enterprise.git extras

# build a static binary with the build number and extra features.
go build -ldflags '-extldflags "-static" -X github.com/drone/drone/version.VersionDev=build.${DRONE_BUILD_NUMBER}' -tags extras -o release/drone github.com/drone/drone/drone
