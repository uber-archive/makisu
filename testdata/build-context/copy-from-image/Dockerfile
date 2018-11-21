FROM busybox
COPY --from=debian:8 /bin/bash /bash
RUN ls / && ls -la /bash
ENTRYPOINT ["/bin/sh", "-c", "ls /bash"]
