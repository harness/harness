#!/bin/bash
set -e -o pipefail

if [ ! -f `which swagger` ]; then
  echo "can't find swagger in the PATH"
  exit 1
fi

for dir in $(ls fixtures/canary)
do
  pushd fixtures/canary/$dir
  rm -rf client models restapi cmd
  swagger generate client
  go test ./...
  if [ $dir != 'kubernetes' ]; then
    swagger generate server
    go test ./...
  fi
  popd
done
