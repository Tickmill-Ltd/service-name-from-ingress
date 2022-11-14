FROM golang:1.19-rc-alpine

RUN apk add --update \
    ca-certificates \
    git \
  && rm -rf /var/cache/apk/*

RUN echo "nobody:x:65534:65534:Nobody:/:" > /etc_passwd

ENV GO111MODULE=on
ENV CGO_ENABLED=0

WORKDIR /go/src/github.com/Tickmill-Ltd/service-name-from-ingress
COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go install -ldflags='-d -s -w' -tags netgo -installsuffix netgo -v ./...

FROM scratch

COPY --from=0 /go/bin/service-name-from-ingress /bin/service-name-from-ingress
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=0 /etc_passwd /etc/passwd

ENTRYPOINT ["/bin/service-name-from-ingress"]
