#!/bin/bash

#nerdctl is a direct replaceform for docker. You don't have to install anything as it comes pre-installed 
# lima does automatic portforwarding, so prom should be accessible on your mac at localhost:9090 when done 
# The go app exposed an HTTP port at 2112, which should be accessible on your mac at locahost:2112/metrics 


# Get the IP address of eth0 on the host machine 
IP_ADDRESS=$(ip -4 addr show eth0 | grep -oP '(?<=inet\s)\d+(\.\d+){3}')

# Replace the IP address in the prometheus.yml file
CONFIG_FILE="prom_config/prometheus.yml"
sed -i "s/[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+:2112/${IP_ADDRESS}:2112/g" "$CONFIG_FILE"

echo "Updated prometheus.yml with IP address: $IP_ADDRESS"
sleep 3

nerdctl run --rm -p 9090:9090 -v "$PWD/prom_config:/etc/prometheus" prom/prometheus



