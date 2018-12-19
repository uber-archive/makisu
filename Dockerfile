FROM golang:1.11 AS builder

RUN mkdir -p /go/src/github.com/uber/makisu
WORKDIR /go/src/github.com/uber/makisu

ADD Makefile .
RUN make ext-tools/Linux/dep

ADD Gopkg.toml Gopkg.lock ./
ADD .git ./.git
ADD cli ./cli
ADD bin ./bin
ADD lib ./lib
RUN make bins

RUN apt-get update && apt-get install -y ca-certificates

FROM scratch
COPY --from=builder /go/src/github.com/uber/makisu/bin/makisu/makisu /makisu-internal/makisu
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /makisu-internal/certs/cacerts.pem
ENTRYPOINT ["/makisu-internal/makisu"]
