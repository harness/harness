#!/bin/sh

examples=`git rev-parse --show-toplevel`/examples

# go to project root
cd "${examples}/generated"
rm -rf cmd models restapi
swagger generate server -A Petstore

cd "${examples}/todo-list"
rm -rf client cmd models restapi
swagger generate client -A TodoList -f ./swagger.yml
swagger generate server -A TodoList -f ./swagger.yml

cd "${examples}/task-tracker"
rm -rf client cmd models restapi
swagger generate client -A TaskTracker -f ./swagger.yml
swagger generate server -A TaskTracker -f ./swagger.yml

cd "${examples}/tutorials/todo-list/server-1"
rm -rf cmd models restapi
swagger generate server -A TodoList -f ./swagger.yml

cd "${examples}/tutorials/todo-list/server-2"
rm -rf cmd models restapi
swagger generate server -A TodoList -f ./swagger.yml

 cd "${examples}/tutorials/todo-list/server-complete"
 swagger generate server -A TodoList -f ./swagger.yml
