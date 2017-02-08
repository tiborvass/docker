FROM    golang:1.7.5-alpine

RUN     apk add -U git go bash curl gcc musl-dev make

RUN     mkdir -p /go/src /go/bin /go/pkg
ENV     GOPATH=/go
ENV     PATH=$PATH:/go/bin
RUN     go get -u github.com/LK4D4/vndr
COPY    . /go/src/yaml
RUN     ( cd /go/src/yaml && vndr )

WORKDIR /go/src/github.com/docker/docker/
ENTRYPOINT ["yaml/generate.sh"]
