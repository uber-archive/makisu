FROM busybox:latest
RUN echo "OK" > /test1.txt

FROM busybox:latest
# Copy from numbered alias.
COPY --from=0 /test.txt1 /test2.txt
RUN cat /test2.txt

FROM busybox
# Copy from another image directly.
COPY --from=debian:8 /bin/bash /bash
RUN ls / && ls -la /bash
ENTRYPOINT ["/bin/sh", "-c", "ls /bash"]
