#!/bin/bash

websites=("http://example.com" "https://www.google.com" "https://www.wikipedia.org")
loop_count=20

# mess the traffic up with tc 
sudo tc qdisc add dev eth0 root netem loss 5% delay 100ms

#loop
for ((i = 1; i <= loop_count; i++)); do
    for site in "${websites[@]}"; do
        echo "Sending request to $site (iteration $i)"
        curl -sS "$site" > /dev/null
        sleep 1
        wget -O- "$site" > /dev/null # O redirects output to stdout coz I don't wanna save the file.
    done
done

#delete the tc

sudo tc qdisc del dev eth0 root

