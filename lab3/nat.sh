#!/bin/bash
sudo docker exec edge_router /bin/bash -c 'iptables -t nat -F; iptables -t filter -F'
sudo docker exec edge_router /bin/bash -c 'iptables -t nat -A POSTROUTING -s 10.0.0.0/16 -o eth1 -j MASQUERADE'
sudo docker exec edge_router /bin/bash -c 'iptables -P FORWARD DROP'
sudo docker exec edge_router /bin/bash -c 'iptables -A FORWARD -m state --state ESTABLISHED,RELATED -j ACCEPT'
sudo docker exec edge_router /bin/bash -c 'iptables -A FORWARD -m state --state NEW -i eth0 -j ACCEPT'
sudo docker exec edge_router /bin/bash -c 'iptables -A FORWARD -m state --state NEW -i eth1 -d 172.16.123.128/28 -j ACCEPT'