FROM busybox:latest
RUN echo "OK" > /test1.txt

FROM busybox:latest
# Copy from default numbered alias.
COPY --from=0 /test1.txt /test2.txt
RUN cat /test2.txt

FROM busybox
# Copy from another image directly.
COPY --from=debian:8 /bin/bash /bash
RUN ls -la /bash

FROM alpine as phaseA
# Setup copy source.
RUN mkdir /src
RUN touch /src/a.txt
RUN touch /src/b.txt #!COMMIT
RUN mkdir /src/c && chown news:news /src/c
RUN touch /src/c/1.txt && chown news:news /src/c/1.txt && chmod 770 /src/c/1.txt
RUN touch /src/c/2.txt

FROM alpine
RUN mkdir /src
RUN touch /src/a.txt
RUN touch /src/b.txt #!COMMIT
# Copy files that were not committed from previous stage.
COPY --from=phaseA /src/c/ /src/c/
RUN cat /src/c/1.txt && cat /src/c/2.txt
RUN rm -rf /src/c/ #!COMMIT

FROM alpine AS phaseB
RUN mkdir /dst
# Copy to existing dir with changed permission.
RUN mkdir /dst/a && chown mail:mail /dst/a && chmod 777 /dst/a
COPY --from=phaseA /src/c/ /dst/a/
RUN mkdir /dst/b && chown daemon:daemon /dst/b && chmod 777 /dst/b
COPY --from=phaseA --archive /src/c/ /dst/b/
RUN mkdir /dst/c && chown games:games /dst/c && chmod 777 /dst/c
COPY --from=phaseA --chown=ftp:ftp /src/c/ /dst/c/
RUN mkdir /dst/d && chown man:man /dst/d && chmod 777 /dst/d
COPY --chown=ftp:ftp 1/ /dst/d/
# Copy to non-existing dir.
COPY --from=phaseA /src/c/ /dst/dst/a
COPY --from=phaseA --archive /src/c/ /dst/dst/b
COPY --from=phaseA --chown=ftp:ftp /src/c/ /dst/dst/c
COPY --chown=ftp:ftp 1/ /dst/dst/d

FROM alpine
RUN mkdir /mine
# Copy root.
COPY --from=phaseB / /
# Verify existing dir.
RUN cat /dst/a/2.txt && test $(stat -c '%U:%G' /dst/a/1.txt) = 'news:news' && test $(stat -c '%a' /dst/a/1.txt) = 770
RUN test $(stat -c '%U:%G' /dst/a) = 'mail:mail' && test $(stat -c '%a' /dst/a) = 777
RUN cat /dst/b/2.txt && test $(stat -c '%U:%G' /dst/b/1.txt) = 'news:news' && test $(stat -c '%a' /dst/b/1.txt) = 770
RUN test $(stat -c '%U:%G' /dst/b) = 'daemon:daemon' && test $(stat -c '%a' /dst/b) = 777
RUN cat /dst/c/1.txt && cat /dst/c/2.txt && test $(stat -c '%U:%G' /dst/c/1.txt) = 'ftp:ftp' && test $(stat -c '%a' /dst/c/1.txt) = 770
RUN test $(stat -c '%U:%G' /dst/c) = 'games:games' && test $(stat -c '%a' /dst/c) = 777
RUN cat /dst/d/1.txt
RUN test $(stat -c '%U:%G' /dst/d) = 'man:man' && test $(stat -c '%a' /dst/d) = 777
# Verify non-existing dir.
RUN cat /dst/dst/a/2.txt && test $(stat -c '%U:%G' /dst/dst/a/1.txt) = 'news:news' && test $(stat -c '%a' /dst/dst/a/1.txt) = 770
RUN test $(stat -c '%U:%G' /dst/dst/a) = 'root:root' && test $(stat -c '%a' /dst/dst/a) = 755
RUN cat /dst/dst/b/2.txt && test $(stat -c '%U:%G' /dst/dst/b/1.txt) = 'news:news' && test $(stat -c '%a' /dst/dst/b/1.txt) = 770
RUN test $(stat -c '%U:%G' /dst/dst/b) = 'news:news' && test $(stat -c '%a' /dst/dst/b) = 755
RUN cat /dst/dst/c/1.txt && cat /dst/dst/c/2.txt && test $(stat -c '%U:%G' /dst/dst/c/1.txt) = 'ftp:ftp' && test $(stat -c '%a' /dst/dst/c/1.txt) = 770
RUN test $(stat -c '%U:%G' /dst/dst/c) = 'ftp:ftp' && test $(stat -c '%a' /dst/dst/c) = 755
RUN cat /dst/dst/d/1.txt
RUN test $(stat -c '%U:%G' /dst/dst/d) = 'ftp:ftp' && test $(stat -c '%a' /dst/dst/d) = 755
ENTRYPOINT ["/bin/sh", "-c", "echo hello"]
