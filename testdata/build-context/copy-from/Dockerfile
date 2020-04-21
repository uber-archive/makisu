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
RUN mkdir /mine
RUN touch /mine/a.txt
RUN touch /mine/b.txt #!COMMIT
RUN mkdir /mine/c
RUN touch /mine/c/1.txt
RUN touch /mine/c/2.txt

FROM alpine
RUN mkdir /mine
RUN touch /mine/a.txt
RUN touch /mine/b.txt #!COMMIT
# Copy files that were not cached from previous stage.
COPY --from=phaseA /mine/c/ /mine/c/
RUN cat /mine/c/1.txt && cat /mine/c/2.txt
RUN rm -rf /mine/c/ #!COMMIT

FROM alpine AS phaseB
RUN mkdir /mine
# Copy files again that were deleted from previous stage.
COPY --from=phaseA /mine/c/ /mine/c/
RUN cat /mine/c/1.txt && cat /mine/c/2.txt
# Copy to existing dir with changed permission.
RUN mkdir /mine/d/ && chown daemon:daemon /mine/d
COPY --from=phaseA /mine/c/ /mine/d/

FROM alpine
RUN mkdir /mine
# Copy root.
COPY --from=phaseB / /
ENTRYPOINT ["/bin/sh", "-c", "cat /mine/c/1.txt && cat /mine/c/2.txt && cat /mine/d/1.txt && cat /mine/d/2.txt && test $(stat -c '%U:%G' /mine/d) = 'daemon:daemon'"]
