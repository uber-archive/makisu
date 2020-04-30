ARG version_default=v1

FROM alpine:latest as base1
ARG version_default
ENV version=$version_default

FROM alpine:latest as base2
ARG version_default
ENV version2=$version_default

ENTRYPOINT if [ -z "$version" -a "$version2" = "v2" ]; then echo "This is correct"; exit 0; else exit 1; fi