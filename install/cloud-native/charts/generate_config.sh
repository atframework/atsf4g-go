#!/bin/bash
cd "$(dirname $0)"

../../atdtool/bin/atdtool template ./ -o ../.. --values ../values/default --set global.world_id=1