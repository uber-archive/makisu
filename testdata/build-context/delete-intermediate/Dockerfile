FROM alpine:latest
RUN mkdir -p /test/dir
RUN touch /test/dir/file
RUN rm -rf /test/dir #!COMMIT

FROM alpine:latest
RUN mkdir -p /test/dir
RUN touch /test/dir/file
RUN rm -rf /test/dir
