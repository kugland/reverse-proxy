FROM scratch
WORKDIR /app
ADD reverse-proxy /app/reverse-proxy
ADD reverse-proxy.log /app/logs/reverse-proxy.log
ENTRYPOINT [ "./reverse-proxy"]