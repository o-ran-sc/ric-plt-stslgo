#!/bin/bash

# influxdb
wget https://dl.influxdata.com/influxdb/releases/influxdb2-2.2.0-amd64.deb
dpkg -i influxdb2-2.2.0-amd64.deb

service influxdb start
service influxdb status

# influx-cli
wget https://dl.influxdata.com/influxdb/releases/influxdb2-client-2.2.1-linux-amd64.tar.gz && \
tar xvzf influxdb2-client-2.2.1-linux-amd64.tar.gz && \
cp influxdb2-client-2.2.1-linux-amd64/influx /usr/local/bin/

influx ping

influx setup \
  --org influxdata \
  --bucket default \
  --username admin \
  --password 12345678 \
  --force  
