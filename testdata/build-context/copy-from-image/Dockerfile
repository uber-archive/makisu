FROM busybox
COPY --from=debian:8 /bin/bash /bash
RUN /bash -c "echo OK"
ENTRYPOINT ["/bin/sh", "-c", "ls /bin/bash"]
