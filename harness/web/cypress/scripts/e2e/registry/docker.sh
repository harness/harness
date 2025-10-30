# Copyright 2024 Harness Inc. All rights reserved.
# Use of this source code is governed by the PolyForm Shield 1.0.0 license
# that can be found in the licenses directory at the root of this repository, also available at
# https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

# Extract named variables from arguments

#!/bin/bash
while [ "$1" != "" ]; do
  case $1 in
    --space_ref )           shift
                            SPACE_REF=$1
                            ;;
    --registry )            shift
                            REGISTRY=$1
                            ;;
    --artifact )            shift
                            ARTIFACT=$1
                            ;;
    --version )             shift
                            VERSION=$1
                            ;;
    * )                     echo "Invalid parameter: $1"
                            exit 1
  esac
  shift
done

set -a
source .env
set +a

echo "Host: $DOCKER_LOCAL_HOST"
echo "Username: $USERNAME"

echo "Start of docker e2e script"
echo ""

echo "docker login $DOCKER_LOCAL_HOST"
docker login $DOCKER_LOCAL_HOST --username $USERNAME --password $PASSWORD

echo "docker pull $ARTIFACT:$VERSION"
docker pull $ARTIFACT:$VERSION
echo ""

echo "docker tag $ARTIFACT:$VERSION $DOCKER_LOCAL_HOST/$SPACE_REF/$REGISTRY/$ARTIFACT:$VERSION"
docker tag $ARTIFACT:$VERSION $DOCKER_LOCAL_HOST/$SPACE_REF/$REGISTRY/$ARTIFACT:$VERSION
echo ""

echo "docker push $DOCKER_LOCAL_HOST/$SPACE_REF/$REGISTRY/$ARTIFACT:$VERSION"
docker push $DOCKER_LOCAL_HOST/$SPACE_REF/$REGISTRY/$ARTIFACT:$VERSION
echo ""

echo "docker rmi $DOCKER_LOCAL_HOST/$SPACE_REF/$REGISTRY/$ARTIFACT:$VERSION"
docker rmi $DOCKER_LOCAL_HOST/$SPACE_REF/$REGISTRY/$ARTIFACT:$VERSION
echo ""

echo "docker pull $DOCKER_LOCAL_HOST/$SPACE_REF/$REGISTRY/$ARTIFACT:$VERSION"
docker pull $DOCKER_LOCAL_HOST/$SPACE_REF/$REGISTRY/$ARTIFACT:$VERSION
echo ""

echo "docker logout"
docker logout $DOCKER_LOCAL_HOST