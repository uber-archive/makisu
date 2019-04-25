FROM alpine

RUN whoami > /tmp/root_file

USER daemon

RUN whoami > /tmp/daemon_file

USER root

ENTRYPOINT [ "sh", "-c", "test $(cat /tmp/root_file) = 'root' && test $(cat /tmp/daemon_file) = 'daemon'" ]