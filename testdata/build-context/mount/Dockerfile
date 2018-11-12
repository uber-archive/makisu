FROM 127.0.0.1:5002/debian:8 AS phase1
ENV A=1
RUN cp /tmp/test.txt /done.txt #!COMMIT

FROM 127.0.0.1:5002/debian:8 AS phase2
ENV A=1
RUN cp /tmp/test.txt /done.txt #!COMMIT
RUN rm /done.txt

FROM 127.0.0.1:5002/debian:8 AS phase3
COPY --from=phase1 /done.txt /done.txt

ENTRYPOINT ["/bin/sh", "-c", "cat /done.txt"]
