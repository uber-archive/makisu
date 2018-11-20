FROM busybox:latest
RUN echo "OK" > /test1

FROM debian:8
COPY --from=0 /test1 /test2
RUN cat /test2
ENTRYPOINT ["/bin/sh", "-c", "cat /test2"]
