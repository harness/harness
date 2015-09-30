#!/bin/bash

cd /tmp

curl -O https://www.sqlite.org/2015/sqlite-autoconf-3081101.tar.gz
tar xzf sqlite-autoconf-3081101.tar.gz
cd sqlite-autoconf-3081101
cd sqlite-3.6.421
./configure -prefix=/scratch/usr/local
make
make install
