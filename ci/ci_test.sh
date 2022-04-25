#!/bin/sh

set -e

# Run the TimeSeriesDB(TimeSeriesDB) service and set the values here as per the service and then the GO functions are tested as part of Dockerfile
# So, keeping these commented
#export TIMESERIESDB_SERVICE_HOST=10.104.76.22
#export TIMESERIESDB_SERVICE_PORT_HTTP=8086

export GO111MODULE=on
go mod download
go test -v ./... --run Test*
