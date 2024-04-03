FROM nginx
RUN apt update && apt install -y iproute2 iputils-ping tcpdump iptables dnsutils curl
COPY start.sh /root/start.sh
COPY nginx.conf /etc/nginx/conf.d/default.conf
ENTRYPOINT []
CMD /root/start.sh