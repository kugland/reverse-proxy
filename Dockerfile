FROM golang:1.12.4-alpine

WORKDIR /app

#EXPOSE 80

ADD reverse-proxy /app/reverse-proxy

ENTRYPOINT [ "./reverse-proxy" ]