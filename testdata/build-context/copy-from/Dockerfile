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
RUN touch /mine/c/3.txt

FROM alpine
RUN mkdir /mine
RUN touch /mine/a.txt
RUN touch /mine/b.txt #!COMMIT
# Copy files that were not cached from previous stage.
COPY --from=phaseA /mine/c/ /mine/c/
RUN cat /mine/c/1.txt && cat /mine/c/2.txt && cat /mine/c/3.txt
RUN rm -rf /mine/c/ #!COMMIT

FROM alpine
RUN mkdir /mine
# Copy files again that were deleted from previous stage.
COPY --from=phaseA /mine/c/ /mine/c/
RUN cat /mine/c/1.txt && cat /mine/c/2.txt && cat /mine/c/3.txt

FROM alpine
RUN mkdir /mine
# Copy root.
COPY --from=phaseA / /
ENTRYPOINT ["/bin/sh", "-c", "cat /mine/a.txt && cat /mine/b.txt && cat /mine/c/1.txt && cat /mine/c/2.txt && cat /mine/c/3.txt"]
