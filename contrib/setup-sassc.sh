#!/bin/bash
set -e

cd /tmp

rm -rf libsass
rm -rf sassc

git clone --depth=1 git://github.com/sass/libsass.git
git clone --depth=1 git://github.com/sass/sassc.git

export SASS_LIBSASS_PATH=/tmp/libsass

cd sassc
make
sudo make install