FROM debian:8 as phase1
RUN mkdir /mine
RUN touch /mine/a.txt

FROM debian:8 as phase2
RUN mkdir /mine
COPY --from=phase1 / /
RUN touch /mine/b.txt
ENTRYPOINT ["/bin/sh", "-c", "ls / && ls /mine && cat /mine/b.txt && cat /mine/a.txt"]
