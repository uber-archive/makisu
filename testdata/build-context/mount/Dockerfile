FROM alpine:latest AS phase1
ENV A=1
RUN cp /tmp/test.txt /done.txt #!COMMIT


FROM alpine:latest AS phase2
ENV A=1
RUN cp /tmp/test.txt /done.txt #!COMMIT
RUN rm /done.txt


FROM alpine:latest AS phase3

# Generate a few empty layers.
RUN mkdir /test #!COMMIT
WORKDIR /test #!COMMIT
RUN ls /test #!COMMIT

COPY --from=phase1 /done.txt /done.txt
ENTRYPOINT ["/bin/sh", "-c", "cat /done.txt"]
