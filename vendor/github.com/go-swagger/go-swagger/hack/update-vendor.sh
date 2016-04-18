#!/bin/zsh

# go to project root
cd `git rev-parse --show-toplevel`

gvt update --all

# not interested in the test files thank you.
rm -rf vendor/**/*_test.go

# remove some items that are problematic for a continuous build and not actually in use
rm -rf vendor/github.com/tylerb/graceful/tests
