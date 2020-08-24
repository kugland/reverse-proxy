FROM golang as build-env
WORKDIR /app
ADD . /app

RUN cd /app && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o rproxy

# FROM golang:alpine
FROM scratch
ADD config.yaml /app/config.yaml
COPY --from=build-env /app/rproxy /app/rproxy
WORKDIR /app
ENTRYPOINT [ "./rproxy"]
