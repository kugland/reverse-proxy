FROM golang as build-env
WORKDIR /app
ADD ./vendor/* /go/src/github.com/
ADD . /app
ENV GOOS=linux 
ENV GOARCH=amd64 
ENV CGO_ENABLED=0

RUN cd /app && go build -o rproxy

# FROM golang:alpine
FROM scratch
ADD config.json /app/config.json
COPY --from=build-env /app/rproxy /app/rproxy
WORKDIR /app
ENTRYPOINT [ "./rproxy"]
