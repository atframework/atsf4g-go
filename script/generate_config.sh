#!/bin/bash
cd "$(dirname $0)"

chmod +x ./atdtool/bin/atdtool
./atdtool/bin/atdtool template ./deploy/charts -o ./ --values ./deploy/values/default,./deploy/values/dev --set global.world_id=1