ARG BASE_IMAGE

FROM $BASE_IMAGE AS phase1

# Make sure that the variable gets expanded at some point in the command
# execution process.
RUN mkdir $HOME/test
WORKDIR $HOME
RUN ls test

RUN apt-get update

# Install runtime package
RUN apt-get install -y --allow-unauthenticated \

    # hello
    hello    #!COMMIT

# Install build-time package
RUN apt-get install -y --allow-unauthenticated \

    # tree
    tree

# Perform build
COPY . /home/udocker/simple-debian-package
WORKDIR /home/udocker/simple-debian-package
RUN make bins

FROM $BASE_IMAGE AS phase2
RUN apt-get update

# Install runtime package
RUN apt-get install -y --allow-unauthenticated hello #!COMMIT

# Copy build artifact
COPY --from=phase1 /home/udocker/simple-debian-package/binary /simple-debian-package
ENTRYPOINT ["/simple-debian-package"]
