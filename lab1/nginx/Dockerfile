FROM nginx
RUN apt update && apt install -y iproute2 iputils-ping tcpdump iptables dnsutils curl apache2-utils
COPY start.sh /root/start.sh
ENTRYPOINT []
CMD /root/start.sh