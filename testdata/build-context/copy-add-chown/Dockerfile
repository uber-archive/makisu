FROM alpine:latest

RUN mkdir -p /tmp/test-data

# Copy without user specified (so should be root)
COPY 1.txt /tmp/test-data/root_file

COPY --chown=daemon:daemon 1.txt /tmp/test-data/daemon_copy_file

ADD --chown=2:2 1.txt /tmp/test-data/daemon_add_file

ENTRYPOINT ["/bin/sh", "-c", "ls -lah /tmp/test-data && test $(stat -c '%U:%G' /tmp/test-data/daemon_copy_file) = 'daemon:daemon' && test $(stat -c '%U:%G' /tmp/test-data/daemon_add_file) = 'daemon:daemon' && test $(stat -c '%U:%G' /tmp/test-data/root_file) = 'root:root'"]