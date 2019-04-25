FROM golang:1.12.4-alpine

WORKDIR /app

EXPOSE 80

ADD reverse-proxy /app

ENTRYPOINT [ "./reverse-proxy" ]