#!/bin/bash

NUM=$1
RATE=$2
PAYLOAD=$3
TIME=$4

for ((i = 0; i < NUM; i++)); do
{
  host1=$(jq ".nodes[$i].PublicIpAddress" clients.json)
  host=${host1//\"/}
  url1=$(jq ".nodes[$i].ServerURL" clients.json)
  url=${url1//\"/}
  port=6000
  user='ubuntu'
  key="~/.ssh/aws"
  id=$i
  node="node"$id

expect <<-END
spawn ssh -oStrictHostKeyChecking=no -i $key $user@$host "cd;cd client;./client -rate=$RATE -payload=$PAYLOAD -port=6100 -time=$TIME -output client$i.log > client$i.output &"
expect EOF
exit
END
} &
done

wait
