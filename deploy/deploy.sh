#!/bin/bash

PRIVILEGED=true

curl -v -X PUT $MARATHON_API_URL/v2/apps/shurenyun-$TASKENV-$SERVICE -H Content-Type:application/json -d \
'{
      "id": "shurenyun-'$TASKENV'-'$SERVICE'",
      "cpus": '$CPUS',
      "mem": '$MEM',
      "instances": '$INSTANCES',
      "constraints": [["hostname", "LIKE", "'$DEPLOYIP'"], ["hostname", "UNIQUE"]],
      "container": {
                     "type": "DOCKER",
                     "docker": {
                                     "image": "'$SERVICE_IMAGE'",
                                     "network": "HOST",
                                     "forcePullImage": '$FORCEPULLIMAGE',
                                     "privileged": '$PRIVILEGED'
                                },
                        "volumes": [
                            {
                                      "containerPath": "/var/run/docker.sock",
                                      "hostPath": "/var/run/docker.sock",
                                      "mode": "RW"
                                }
                        ]
                   },
      "env": {
                "NO_BAMBOO": "true"
             },
      "uris": [
               "'$CONFIGSERVER'/config/demo/config/registry/docker.tar.gz"
       ]
}'
