ARG GO_VERSION

FROM golang:${GO_VERSION} AS builder

RUN mkdir -p /workspace/github.com/uber/makisu
WORKDIR /workspace/github.com/uber/makisu

ADD Makefile .
ADD go.mod ./go.mod
ADD go.sum ./go.sum
RUN make vendor
ADD .git ./.git
ADD bin ./bin
ADD lib ./lib
ADD mocks ./mocks
RUN make lbins

FROM scratch
COPY --from=builder /workspace/github.com/uber/makisu/bin/makisu/makisu.linux /makisu-internal/makisu
ADD ./assets/cacerts.pem /makisu-internal/certs/cacerts.pem

ENTRYPOINT ["/makisu-internal/makisu"]
