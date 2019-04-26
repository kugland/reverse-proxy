#!/bin/bash
docker run --rm -it -v "$PWD":/go/src/reverse-proxy -w /go/src/reverse-proxy -e GOOS=linux -e GOARCH=386 golang:1.12.4 go build -v