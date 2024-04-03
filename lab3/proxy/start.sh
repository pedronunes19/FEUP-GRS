#!/bin/bash
/sbin/ip r d default via 10.0.1.1
/sbin/ip r a default via 10.0.1.254

while true ; do /bin/sleep 5m; done