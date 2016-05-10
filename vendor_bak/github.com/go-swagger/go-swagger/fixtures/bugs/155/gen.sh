#!/bin/sh

go install ../../../cmd/swagger
rm -rf generated
swagger generate server -f swagger.yml  -t generated
