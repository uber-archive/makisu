FROM alpine
RUN mkdir /testdata && chmod a+rw /testdata
RUN whoami > /testdata/root_file

USER daemon
RUN whoami > /testdata/daemon_file
USER root

ENTRYPOINT [ "sh", "-c", "test $(cat /testdata/root_file) = 'root' && test $(cat /testdata/daemon_file) = 'daemon'" ]
