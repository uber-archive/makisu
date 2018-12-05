FROM debian:8

MAINTAINER foo@bar
ENV TEST=testenv
LABEL test.label.key=test_label_value
HEALTHCHECK --interval=10s\
  --timeout=30s \
  CMD echo hello || exit 1
RUN touch /home/testfile
