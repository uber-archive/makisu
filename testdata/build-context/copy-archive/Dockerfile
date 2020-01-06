FROM alpine as phaseA
RUN mkdir /hello
RUN mkdir /hello/world
RUN touch /hello/world/a.txt
RUN chmod 775 /hello/world/a.txt
RUN chown daemon:daemon /hello/world

FROM alpine
RUN mkdir /hello
COPY --from=phaseA --archive /hello /hello
ENTRYPOINT ["/bin/sh", "-c", "test $(stat -c '%U:%G' /hello) = 'root:root' && test $(stat -c '%U:%G' /hello/world) = 'daemon:daemon' && test $(stat -c '%U:%G' /hello/world/a.txt) = 'root:root' && test $(stat -c '%a' /hello/world/a.txt) = 775 "]
