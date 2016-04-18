#!/bin/sh

json2yaml fixtures/json -s -r
cp -a fixtures/json/* fixtures/yaml
find fixtures/yaml -name '*.json' -exec rm -rf {} \;
find fixtures/json -name '*.yaml' -exec rm -rf {} \;