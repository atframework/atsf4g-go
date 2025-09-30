#!/bin/bash
cd "$(dirname $0)"

./atdtool/bin/atdtool template ./deploy/charts -o ./ --values ./deploy/values/default --set global.world_id=1