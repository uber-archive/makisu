
![Makisu](assets/logo/Lockup.svg "Makisu Logo")

[![Build Status](https://travis-ci.com/uber/makisu.svg?branch=master)](https://travis-ci.com/uber/makisu)
[![GoReportCard](https://goreportcard.com/badge/github.com/uber/makisu)](https://goreportcard.com/report/github.com/uber/makisu)
[![Github Release](https://img.shields.io/github/release/uber/makisu.svg)](https://github.com/uber/makisu/releases)

**This project will be deprecated and be archived by 4th of May 2021**

The makisu project is no longer actively maintained and will soon be archived. Please read the details [in this issue](https://github.com/uber/makisu/issues/365).

Makisu is a fast and flexible Docker image build tool designed for unprivileged containerized environments such as Mesos or Kubernetes.

Some highlights of Makisu:

* Requires no elevated privileges or containerd/Docker daemon, making the build process portable.
* Uses a distributed layer cache to improve performance across a build cluster.
* Provides control over generated layers with a new optional keyword [`#!COMMIT`](#explicit-commit-and-cache), reducing the number of layers in images.
* Is Docker compatible. Note, the Dockerfile parser in Makisu is opinionated in some scenarios. More details can be found [here](docs/PARSER.md).

Makisu has been in use at Uber since early 2018, building thousands of images every day across 4
different languages. The motivation and mechanism behind it are explained in https://eng.uber.com/makisu/.


- [Building Makisu](#building-makisu)
- [Running Makisu](#running-makisu)
  - [Makisu anywhere](#makisu-anywhere)
  - [Makisu on Kubernetes](#makisu-on-kubernetes)
- [Using Cache](#using-cache)
  - [Configuring distributed cache](#configuring-distributed-cache)
  - [Explicit Commit and Cache](#explicit-commit-and-cache)
- [Configuring Docker Registry](#configuring-docker-registry)
- [Comparison With Similar Tools](#comparison-with-similar-tools)
- [Contributing](#contributing)
- [Contact](#contact)


# Building Makisu

## Building Makisu image

To build a Docker image that can perform builds inside a container:
```
make images
```

## Building Makisu binary and build simple images

To get the makisu binary locally:
```
go get github.com/uber/makisu/bin/makisu
```
For a Dockerfile that doesn't have RUN, makisu can build it without Docker daemon, containerd or runc:
```
makisu build -t ${TAG} --dest ${TAR_PATH} ${CONTEXT}
```

# Running Makisu

For a full list of flags, run `makisu build --help` or refer to the README [here](docs/COMMAND.md).

## Makisu anywhere

To build Dockerfiles that contain RUN, Makisu needs to run in a container.
To try it locally, the following snippet can be placed inside your `~/.bashrc` or `~/.zshrc`:
```shell
function makisu_build() {
    makisu_version=${MAKISU_VERSION:-latest}
    cd ${@: -1}
    docker run -i --rm --net host \
        -v /var/run/docker.sock:/docker.sock \
        -e DOCKER_HOST=unix:///docker.sock \
        -v $(pwd):/makisu-context \
        -v /tmp/makisu-storage:/makisu-storage \
        gcr.io/uber-container-tools/makisu:$makisu_version build \
            --commit=explicit \
            --modifyfs=true \
            --load \
            ${@:1:${#@}-1} /makisu-context
    cd -
}
```
Now you can use `makisu_build` like you would use `docker build`:
```shell
$ makisu_build -t myimage .
```
Note:
* Docker socket mount is optional. It's used together with `--load` for loading images back into Docker daemon for convenience of local development. So does the mount to /makisu-storage, which is used for local cache. If the image would be pushed to registry directly, please remove `--load` for better performance.
* The `--modifyfs=true` option let Makisu assume ownership of the filesystem inside the container. Files in the container that don't belong to the base image will be overwritten at the beginning of build.
* The `--commit=explicit` option let Makisu only commit layer when it sees `#COMMIT` and at the end of the Dockerfile. See ["Explicit Commit and Cache"](#explicit-commit-and-cache) for more details.

## Makisu on Kubernetes

Makisu makes it easy to build images from a GitHub repository inside Kubernetes. A single pod (or job) is
created with an init container, which will fetch the build context through git or other means, and place
that context in a designated volume. Once it completes, the Makisu container will be created and executes
the build, using that volume as its build context.

### Creating registry configuration

Makisu needs registry configuration mounted in to push to a secure registry.
The config format is described in [documentation](docs/REGISTRY.md).
After creating configuration file on local filesystem, run the following command to create the k8s secret:
```shell
$ kubectl create secret generic docker-registry-config --from-file=./registry.yaml
secret/docker-registry-config created
```

### Creating Kubernetes job spec

To setup a Kubernetes job to build a GitHub repository and push to a secure registry, you can refer to our Kubernetes job spec [template](examples/k8s/github-job-template.yaml) (and out of the box [example](examples/k8s/github-job.yaml)) .

With such a job spec, a simple `kubectl create -f job.yaml` will start the build.
The job status will reflect whether the build succeeded or failed

# Using cache

## Configuring distributed cache

Makisu supports distributed cache, which can significantly reduce build time, by up to 90% for some of Uber's code repos.
Makisu caches docker image layers both locally and in docker registry (if --push parameter is provided), and uses a separate key-value store to map lines of a Dockerfile to names of the layers.

For example, Redis can be setup as a distributed cache key-value store with this [Kubernetes job spec](examples/k8s/redis.yaml).
Then connect Makisu to redis cache by passing `--redis-cache-addr=redis:6379` argument.
If the Redis server is password-protected, use `--redis-cache-password=password` argument.
Cache has a 14 day TTL by default, which can be configured with `--local-cache-ttl=14d` argument.

For more options on cache, please see [Cache](docs/CACHE.md).

## Explicit commit and cache

By default, Makisu will cache each directive in a Dockerfile. To avoid committing and caching everything, the layer cache can be further optimized via explicit caching with the `--commit=explicit` flag.
Dockerfile directives may then be manually cached using the `#!COMMIT` annotation:

```Dockerfile
FROM node:8.1.3

ADD package.json package.json
ADD pre-build.sh

# A bunch of pre-install steps here.
...
...
...

# A step to be cached. A single layer will be committed and cached here on top of base image.
RUN npm install #!COMMIT

...
...
...

# The last step of the last stage always commit by default, generating and caching another layer.
ENTRYPOINT ["/bin/bash"]
```
In this example, only 2 additional layers on top of base image will be generated and cached.

# Configuring Docker Registry

For the convenience to work with any public Docker Hub repositories including library/.*, a default config is provided:
```
index.docker.io:
  .*:
    security:
      tls:
        client:
          disabled: false
      // Docker Hub requires basic auth with empty username and password for all public repositories.
      basic:
        username: ""
        password: ""
```
Registry configs can be passed in through the `--registry-config` flag, either as a file path of as a raw json blob (converted to json using [yq](https://github.com/kislyuk/yq)):
```
--registry-config='{"gcr.io": {"uber-container-tools/*": {"push_chunk": -1, "security": {"basic": {"username": "_json_key", "password": "<escaped key here>"}}}}}'
```
For more details on configuring Makisu to work with your registry client, see the [documentation](docs/REGISTRY.md).

# Comparison With Similar Tools

### Bazel

We were inspired by the Bazel project in early 2017. It is one of the first few tools that could build Docker compatible images without using Docker or any form of containerizer.
It works very well with a subset of Docker build scenarios given a Bazel build file. However, it does not support `RUN`, making it hard to replace most docker build workflows.

### Kaniko

Kaniko provides good compatibility with Docker and executes build commands in userspace without the need for Docker daemon, although it must still run inside a container. Kaniko offers smooth integration with Kubernetes, making it a competent tool for Kubernetes users.
On the other hand, Makisu has some performance tweaks for large images with multi-phase builds by avoiding unnecessary disk scans, and offers more control over cache generation and layer size through #!COMMIT, make it optimal for complex workflows.

### BuildKit / img

BuildKit and img depend on runc/containerd and supports parallel stage executions, whereas Makisu and most other tools execute Dockefile in order.
However, BuildKit and img still need seccomp and AppArmor to be disabled to launch nested containers, which is not ideal and may not be doable in some production environments.

# Contributing

Please check out our [guide](docs/CONTRIBUTING.md).

# Contact

To contact us, please join our [Slack channel](https://join.slack.com/t/uber-container-tools/shared_invite/enQtNTIxODAwMDEzNjM1LWIwYzIxNmUwOGY3MmVmM2MxYTczOTQ4ZDU0YjAxMTA0NDgyNzdlZTA4ZWVkZGNlMDUzZDA1ZTJiZTQ4ZDY0YTM).
