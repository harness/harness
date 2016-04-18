#!/bin/bash
set -e -o pipefail

 mkdir -p /drone/{testresults,coverage,dist}
 go test -race -timeout 20m -v $(go list ./... | grep -v vendor) | go-junit-report -dir /drone/testresults

# Run test coverage on each subdirectories and merge the coverage profile.
echo "mode: ${GOCOVMODE-count}" > profile.cov

repo_pref="${CI_BUILD_DIR##${GOPATH/%:*/}/src/}/"

# Standard go tooling behavior is to ignore dirs with leading underscores
# skip generator for race detection and coverage
for dir in $(go list ./... | grep -v -E 'vendor|generator')
do
  pth="${dir//*$repo_pref}"
  go test -covermode=${GOCOVMODE-count} -coverprofile=${pth}/profile.out $dir
  if [ -f $pth/profile.out ]
  then
      cat $pth/profile.out | tail -n +2 >> profile.cov
      # rm $pth/profile.out
  fi
done

go tool cover -func profile.cov
gocov convert profile.cov | gocov report
gocov convert profile.cov | gocov-html > /drone/coverage/coverage-${CI_BUILD_NUM-"0"}.html
