#!/bin/bash
set -e

cd /tmp

# cleanup previously downloaded and unpacked files.
rm -rf sqlite-autoconf-3081101.tar.gz
rm -rf sqlite-autoconf-3081101

# download sqlite
curl -O https://www.sqlite.org/2015/sqlite-autoconf-3081101.tar.gz
tar xzf sqlite-autoconf-3081101.tar.gz

# build and install
cd sqlite-autoconf-3081101
./configure -prefix=/scratch/usr/local
make
make install
