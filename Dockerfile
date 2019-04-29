FROM golang:1.12.4-alpine

WORKDIR /app

EXPOSE 80

ADD reverse-proxy /app/reverse-proxy
ADD fullchain.pem /app/fullchain.pem
ADD privkey.pem /app/privkey.pem

#ENTRYPOINT [ "./reverse-proxy", "-tls", "-cert fullchain.pem", "-key privkey.pem" ]
CMD ./reverse-proxy -tls -cert fullchain.pem -key privkey.pem