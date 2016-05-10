#!/usr/bin/env make

all:	tests

test:	tests

tests:
	(go build testpid1.go; docker build -t reaper/test .; ./runtests.sh)



