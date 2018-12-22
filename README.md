# Makisu :sushi:

[![Build Status](https://travis-ci.org/uber/makisu.svg?branch=master)](https://travis-ci.org/uber/makisu)
[![GoReportCard](https://goreportcard.com/badge/github.com/uber/makisu)](https://goreportcard.com/report/github.com/uber/makisu)
[![Github Release](https://img.shields.io/github/release/uber/makisu.svg)](https://github.com/uber/makisu/releases)

Makisu is a fast and flexible Docker image build tool designed for unprivileged containerized environments such as Mesos or Kubernetes.

Some highlights of Makisu:
* Requires no elevated privileges or containerd/Docker daemon, making the build process portable.
* Uses a distributed layer cache to improve performance across a build cluster.
* Provides control over generated layers with a new optional keyword [`#!COMMIT`](#explicit-commit-and-cache), reducing the number of layers in images.
* Is Docker compatible. Note, the Dockerfile parser in Makisu is opinionated in some scenarios. More details can be found [here](lib/parser/dockerfile/README.md).

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


# Building Makisu

## Building Makisu image

To build a Docker image that can perform builds inside a container:
```
make image
```

## Building Makisu binary and build simple images

To get the makisu binary locally:
```
go get github.com/uber/makisu/bin/makisu
```
For a Dockerfile that doesn't have RUN, makisu can build it without Docker daemon, containerd or runc:
```
makisu build -t ${TAG} -dest ${TAR_PATH} ${CONTEXT}
```

# Running Makisu

For a full list of flags, run `makisu build --help` or refer to the README [here](bin/makisu/README.md).

## Makisu anywhere

To build Dockerfiles that contain RUN, Makisu still needs to run in a container.
To try it locally, the following snippet can be placed inside your `~/.bashrc` or `~/.zshrc`:
```shell
function makisu_build() {
    makisu_version=${MAKISU_VERSION:-v0.1.4}
    cd ${@: -1}
    docker run -i --rm --net host \
        -v /var/run/docker.sock:/docker.sock \
        -e DOCKER_HOST=unix:///docker.sock \
        -v $(pwd):/makisu-context \
        -v /tmp/makisu-storage:/makisu-storage \
        gcr.io/makisu-project/makisu:$makisu_version build \
            --modifyfs=true --load \
            ${@:1:-1} /makisu-context
    cd -
}
```
Now you can use `makisu_build` like you would use `docker build`:
```shell
$ makisu_build -t myimage .
```
Note: Docker socket mount is optional. It's used together with `--load` for loading images back into Docker daemon for convenience of local development. Neither does the mount to /makisu-storage, which is used for local cache.

## Makisu on Kubernetes

Makisu makes it easy to build images from a GitHub repository inside Kubernetes. A single pod (or job) is
created with an init container, which will fetch the build context through git or other means, and place
that context in a designated volume. Once it completes, the Makisu container will be created and executes
the build, using that volume as its build context.

### Creating registry configuration

Makisu needs registry configuration mounted to push to a secure registry. The config format is described [here](#configuring-docker-registry). After creating configuration file on local filesystem, run the following
command to create the k8s secret:
```shell
$ kubectl create secret generic docker-registry-config --from-file=./registry.yaml
secret/docker-registry-config created
```

Below is a template to build a GitHub repository and push to a secure registry:
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: imagebuilder-github
spec:
  template:
    spec:
      restartPolicy: Never
      initContainers:
      - name: provisioner
        image: alpine/git
        args:
        - clone
        - https://github.com/<your repo>
        - /makisu-context
        volumeMounts:
        - name: context
          mountPath: /makisu-context
      containers:
      - name: makisu
        image: gcr.io/makisu-project/makisu:v0.1.4
        imagePullPolicy: IfNotPresent
        args:
        - build
        - --push=gcr.io
        - --modifyfs=true
        - -t=<your image tag>
        - --registry-config=/registry-config/registry.yaml
        - /makisu-context
        volumeMounts:
        - name: context
          mountPath: /makisu-context
        - name: registry-config
          mountPath: /registry-config
      volumes:
      - name: context
        emptyDir: {}
      - name: registry-config
        secret:
          secretName: docker-registry-config
```
With this job spec, a simple `kubectl create -f job.yaml` will start the build. The job status will reflect whether the build succeeded or failed (see out of the box example [here](examples/k8s/github-job.yaml).

# Using cache
## Configuring distributed cache

Makisu supports distributed layer cache, which can significantly reduce build time, by up to 90% for some of Uber's code repos.
It uses target registry for layer storage, and needs to connect to a separate key-value store to map lines of a Dockerfile to a tentative layer SHA stored in Docker registry. For example, Redis can be used as a cache key-value store with the following Kubernetes job spec:

```yaml
# redis.yaml
---
apiVersion: v1
kind: Pod
metadata:
  name: redis
  labels:
    redis: "true"
spec:
  containers:
  - name: main
    image: kubernetes/redis:v1
    env:
    - name: MASTER
      value: "true"
    ports:
    - containerPort: 6379
---
kind: Service
apiVersion: v1
metadata:
  name: redis
spec:
  selector:
    redis: "true"
  ports:
  - protocol: TCP
    port: 6379
    targetPort: 6379
---
```

Finally, connect Redis as the Makisu layer cache by passing `--redis-cache-addr=redis:6379` argument.
Cache has a 7 day TTL by default, which can be configured with `--local-cache-ttl=7d` argument.

## Explicit commit and cache

By default, Makisu will cache each directive in a Dockerfile. To avoid committing and caching everything, the layer cache can be further optimized via explicit caching with the `--commit=explicit` flag. Dockerfile directives may then be manually cached using the `#!COMMIT` annotation:

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

# Configuring Docker Registry

Makisu supports TLS and Basic Auth with Docker registry (Docker Hub, GCR, and private registries). By default, TLS is enabled and makisu uses a list of common root CA certs to authenticate registry.
```go
// Config contains Docker registry client configuration.
type Config struct {
  Concurrency int           `yaml:"concurrency"`
  Timeout     time.Duration `yaml:"timeout"`
  Retries     int           `yaml:"retries"`
  PushRate    float64       `yaml:"push_rate"`
  // If not specify, a default chunk size will be used.
  // Set it to -1 to turn off chunk upload.
  // NOTE: gcr does not support chunked upload.
  PushChunk int64           `yaml:"push_chunk"`
  Security  security.Config{
    TLS       *httputil.TLSConfig `yaml:"tls"`
    BasicAuth *types.AuthConfig   `yaml:"basic"`
  }`yaml:"security"`
}
```
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
To configure your own registry endpoint, pass a custom configuration file to Makisu with `--registry-config=${PATH_TO_CONFIG}`.:
```yaml
[registry]:
  [repo]:
    security:
      tls:
        client:
          disabled: false
          cert:
            path: <path to cert>
          key:
            path: <path to key>
          passphrase
            path: <path to passphrase>
        ca:
          cert:
            path: <path to ca certs, appends to system certs. A list of common ca certs are used if empty>
      basic:
        username: <username>
        password: <password>
```
Example:
```yaml
"gcr.io":
  "makisu-project/*":
    push_chunk: -1
    security:
      basic:
        username: _json_key
        password: |-
            {
                <json here>
            }
```
This config can also be passed in as a raw json blob through the `--registry-config` flag:
```
--registry-config='{"gcr.io": {"makisu-project/*": {"push_chunk": -1, "security": {"basic": {"username": "_json_key", "password": "<escaped key here>"}}}}}'
```
Consider using the great tool [yq](https://github.com/kislyuk/yq) to convert your yaml configuration into the blob that can be passed in.

# Comparison With Similar Tools

### Bazel

We were inspired by the Bazel project in early 2017. It is one of the first few tools that could build Docker compatible images without using Docker or any form of containerizer.
It works very well with a subset of Docker build scenarios given a Bazel build file. However, it does not support `RUN`, making it hard to replace most docker build workflows.

### Kaniko

Kaniko provides good compatibility with Docker and executes build commands in userspace without the need for Docker daemon, although it must still run inside a container. Kaniko offers smooth integration with Kubernetes, making it a competent tool for Kubernetes users.
On the other hand, Makisu has some performance tweaks for large images (especially those with node_modules), allows cache to expire, and offers more control over cache generation through #!COMMIT, make it optimal for complex workflows.

### BuildKit

BuildKit depends on runc/containerd and supports parallel stage executions, whereas Makisu and most other tools execute Dockefile in order.
However, BuildKit still needs access to /proc to launch nested containers, which is not ideal and may not be doable in some production environments.
