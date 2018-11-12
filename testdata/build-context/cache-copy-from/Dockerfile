FROM debian:8 as phase1
RUN mkdir /mine
RUN touch /mine/a.txt
RUN touch /mine/b.txt #!COMMIT
RUN mkdir /mine/c
RUN touch /mine/c/1.txt
RUN touch /mine/c/2.txt
RUN touch /mine/c/3.txt

FROM debian:8 as phase2
RUN mkdir /mine
RUN touch /mine/a.txt
RUN touch /mine/b.txt #!COMMIT
COPY --from=phase1 /mine/c/ /mine/c/
ENTRYPOINT ["/bin/sh", "-c", "cat /mine/c/1.txt && cat /mine/c/2.txt && cat /mine/c/3.txt"]
