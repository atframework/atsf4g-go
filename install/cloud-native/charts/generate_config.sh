#!/bin/bash
cd "$(dirname $0)"

servers=$(ls -l | awk '/^d/ {print $NF}')

for svr in ${servers}; do
  if [ $svr = "libapp" -o $svr = "app" ]; then
    continue
  fi
  if [[ ! -e "$svr/Chart.yaml" ]]; then
    continue
  fi
  echo $svr
  ../../atdtool/atdtool template ./$svr/ -o ../../$svr --values ../values/default
done