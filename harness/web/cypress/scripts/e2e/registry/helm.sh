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

echo "Start of helm e2e script"
echo ""

echo "helm login"
helm registry login $LOCAL_HOST --username $USERNAME --password $PASSWORD --insecure

echo "helm pull $ARTIFACT --version $VERSION"
helm pull $ARTIFACT --version $VERSION
echo ""

ARTIFACT_NAME=${ARTIFACT##*/}
ARTIFACT_FILE_NAME="$ARTIFACT_NAME-$VERSION.tgz"
echo $ARTIFACT_FILE_NAME

echo "helm push $ARTIFACT_FILE_NAME oci://$LOCAL_HOST/$SPACE_REF/$REGISTRY"
helm push $ARTIFACT_FILE_NAME oci://$LOCAL_HOST/$SPACE_REF/$REGISTRY
echo ""

echo "rm -rf $ARTIFACT_FILE_NAME"
rm -rf $ARTIFACT_FILE_NAME
echo ""

echo "helm pull oci://$LOCAL_HOST/$SPACE_REF/$REGISTRY/$ARTIFACT_NAME --version $VERSION"
helm pull oci://$LOCAL_HOST/$SPACE_REF/$REGISTRY/$ARTIFACT_NAME --version $VERSION
echo ""

echo "rm -rf $ARTIFACT_FILE_NAME"
rm -rf $ARTIFACT_FILE_NAME
echo ""

echo "helm logout"
helm registry logout $LOCAL_HOST