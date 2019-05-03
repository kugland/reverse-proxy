FROM golang:1.12.4-alpine

WORKDIR /app

EXPOSE 80

ADD reverse-proxy /app/reverse-proxy
ADD fullchain.pem /app/fullchain.pem
ADD privkey.pem /app/privkey.pem
ADD reverse-proxy.log /app/logs/reverse-proxy.log

ENTRYPOINT [ "./reverse-proxy", "-tls", "-cert=fullchain.pem", "-key=privkey.pem", "-logfile=/app/logs/reverse-proxy.log" ]
