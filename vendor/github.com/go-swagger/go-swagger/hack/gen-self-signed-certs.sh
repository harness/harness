#!/bin/sh

# generate CA
openssl genrsa -out myCA.key 2048
openssl req -x509 -new -key myCA.key -out myCA.crt -days 730 -subj /CN="Go Swagger"

# generate server cert and key
openssl genrsa -out mycert1.key 2048
openssl req -new -out mycert1.req -key mycert1.key -subj /CN="goswagger.local"
openssl x509 -req -in mycert1.req -out mycert1.crt -CAkey myCA.key -CA myCA.crt -days 365 -CAcreateserial -CAserial serial
