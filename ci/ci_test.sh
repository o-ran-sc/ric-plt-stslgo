#!/bin/sh

set -e

export TIMESERIESDB_SERVICE_TOKEN=$(cat ~/.influxdbv2/configs | grep -m 1 token | awk '{ print $3; exit }' | xargs)

go test -v ./... --run Test*
