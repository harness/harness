#!/bin/bash

# Tweak PATH for Travis
export PATH=$PATH:$HOME/gopath/bin

OPTIONS="-config=test-integration/dbconfig.yml -env mysql_noflag"

set -ex

sql-migrate status $OPTIONS | grep -q "Make sure that the parseTime option is supplied"
