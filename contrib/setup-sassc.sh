#!/bin/bash
set -e

cd /tmp

# cleanup previously downloaded and unpacked files.
rm -rf libsass
rm -rf sassc

# download the latest build of sassc
git clone --depth=1 git://github.com/sass/libsass.git
git clone --depth=1 git://github.com/sass/sassc.git

export SASS_LIBSASS_PATH=/tmp/libsass

# build the sassc binary
cd sassc
make

# isntall the sassc binary
install -t /usr/local/bin bin/sassc
