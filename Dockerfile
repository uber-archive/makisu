FROM golang:1.11 AS builder

RUN mkdir -p /go/src/github.com/uber/makisu
WORKDIR /go/src/github.com/uber/makisu

ADD Makefile .
RUN make ext-tools/Linux/dep

ADD Gopkg.toml Gopkg.lock ./
ADD .git ./.git
ADD bin ./bin
ADD lib ./lib
RUN make lbins

FROM scratch
COPY --from=builder /go/src/github.com/uber/makisu/bin/makisu/makisu.linux /makisu-internal/makisu
ADD ./assets/cacerts.pem /makisu-internal/certs/cacerts.pem
ENTRYPOINT ["/makisu-internal/makisu"]
