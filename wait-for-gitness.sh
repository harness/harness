#!/bin/bash
set -eux

declare -r HOST="localhost:3000/health"

wait-for-url() {
    echo "Testing $1"
    timeout -s TERM 180 bash -c \
    'while [[ "$(curl -s -o /dev/null -L -w ''%{http_code}'' ${0})" != "200" ]];\
    do echo "Waiting for ${0}" && sleep 2;\
    done' ${1}
    echo "OK!"
    curl $1
}
wait-for-url http://${HOST}
