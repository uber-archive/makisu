FROM golang:1.11.1 AS phase1

# Perform build
COPY . /home/udocker/test-scratch
WORKDIR /home/udocker/test-scratch
RUN make bins

FROM scratch AS phase2

# Copy build artifact
COPY --from=phase1 /home/udocker/test-scratch/binary /test-scratch
ENTRYPOINT ["/test-scratch"]
