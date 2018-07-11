FROM alpine:3.7

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 https://github.com/aaechain/go-aaechain && \
  (cd go-aaeereum && make gaae) && \
  cp go-aaeereum/build/bin/gaae /gaae && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-aaeereum && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/gaae"]
