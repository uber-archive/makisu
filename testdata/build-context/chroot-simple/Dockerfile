FROM debian:9
RUN echo "ASD" > /asd
RUN cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1
