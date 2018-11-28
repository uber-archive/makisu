ARG BASE_IMAGE="busybox:latest"
ARG RUNTIME_BASE_IMAGE="alpine"

# construct build image
FROM ${BASE_IMAGE} as builder

ARG BUILD_HOME="/home/builder"
RUN mkdir -p ${BUILD_HOME}
WORKDIR ${BUILD_HOME}

ARG TEST_ARG_1="true"
ENV TEST_ENV_1="true"

ARG SCRIPT_FOLDER="scripts"
COPY ./${SCRIPT_FOLDER}/test-a.sh ./scripts/test-a.sh
RUN ./scripts/test-a.sh

COPY ${SCRIPT_FOLDER}/test-b.sh /usr/local/bin/run-foo.sh

# construct application image
FROM ${RUNTIME_BASE_IMAGE}
LABEL base_image_name="alpine" app_id="docker-integration-test"

ENV CONFIG_DIR="/etc/foo/conf.d"
COPY --from=builder /usr/local/bin/run-foo.sh /usr/local/bin/run-foo

CMD ["/usr/local/bin/run-foo"]
