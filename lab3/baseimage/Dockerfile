FROM ubuntu:20.04
RUN apt update && apt install -y iproute2 iputils-ping tcpdump iptables dnsutils curl apache2-utils
COPY sleep.sh /root/sleep.sh
CMD /root/sleep.sh