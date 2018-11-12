FROM debian:8 as phase1
COPY *.json /
RUN cat /1.json

FROM debian:8 as phase2
RUN mkdir /mine
COPY --from=phase1 /*.json /mine/
COPY *.txt /mine/
RUN cat /mine/1.txt

ENTRYPOINT ["/bin/sh", "-c", "ls / && ls /mine && cat /mine/1.json && cat /mine/1.txt && cat /mine/2.txt"]
