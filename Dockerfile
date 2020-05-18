FROM golang as build-env
WORKDIR /app
ADD ./vendor/* /go/src/github.com/
ADD . /app

RUN cd /app && go build -o builtapp

FROM scratch
WORKDIR /app
COPY --from=build-env /app/builtapp /reverse-proxy
ENTRYPOINT [ "./reverse-proxy"]