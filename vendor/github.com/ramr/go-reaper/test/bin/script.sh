#!/bin/bash

#
#  Usage: $0 <num-threads> <worker-args>
#         where:  <num-threads> = Number of threads - default 5.
#                 <worker-args> = <num-workers>  <sleep-time>
#                 <num-workers> = number of workers - default 10.
#                 <sleep-time>  = max sleep time in seconds - default 5.
#                                 This is used by the workers to randomly
#                                 sleep between 0 and the max sleep time.
#
#  Example: ./bin/script.sh 2 3 4
#           num-workers: 2   === run 2 threads
#           worker-args: 3 4 === create 3 workers (which become orphans) per
#                                thread with a max sleep time of 4 seconds.
#
SCRIPT_DIR=$(dirname "${BASH_SOURCE[0]}")
NTIMES=${1:-"5"}
shift

echo  "  -  $0 started with $NTIMES parallel threads ..."

for i in $(seq $NTIMES); do
  nohup bash -c "$SCRIPT_DIR/worker.sh $@" < /dev/null &> /dev/null &
  pid=$!
  echo "  - Started background worker - pid=$pid"
done
