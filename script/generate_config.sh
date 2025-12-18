#!/bin/bash
cd "$(dirname $0)"

if [ -f "./atdtool/bin/atdtool" ]; then
  chmod +x ./atdtool/bin/atdtool
  ./atdtool/bin/atdtool template ./deploy/charts -o ./ --values ./deploy/values/default,./deploy/values/dev --set global.world_id=1
else
  chmod +x ./atdtool/atdtool
  ./atdtool/atdtool template ./deploy/charts -o ./ --values ./deploy/values/default,./deploy/values/dev --set global.world_id=1
fi