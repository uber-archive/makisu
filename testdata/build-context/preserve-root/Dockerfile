ARG BASE_IMAGE

FROM $BASE_IMAGE

# Create a file and dir owned by daemon user
RUN touch /daemon_file && \
    chown daemon:daemon /daemon_file && \
    mkdir /daemon_dir && \
    chown daemon:daemon /daemon_dir && \
    touch /daemon_dir/root_file && \
    mkdir /data && \
    echo 'FROM alpine' > /data/Dockerfile

ENTRYPOINT [ \
  "/bin/sh", \
  "-c", \
  "cd /data && /makisu-internal/makisu --modifyfs=true build -t test-preserve-root --preserve-root . && test $(stat -c '%U:%G' /daemon_file) = 'daemon:daemon' && test $(stat -c '%U:%G' /daemon_dir) = 'daemon:daemon' && test $(stat -c '%U:%G' /daemon_dir/root_file) = 'root:root'" \
  ]