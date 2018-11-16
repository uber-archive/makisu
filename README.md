# Makisu :sushi:

A Docker image build tool that is more flexible and faster at scale. This makes it easy to build
lots of Docker images directly from a containerized environment such as Kubernetes. Specifically, Makisu:
* Uses a distributed layer cache to improve performance across a build cluster.
* Provides control over generated layers with a new keyword `#!COMMIT`, reducing the number of layers in images.
* Requires no elevated privileges, making the build process portable.
* Docker compatible. Note, our Dockerfile parser is opinionated in some scenarios. More details can be found [here](lib/parser/dockerfile/README.md).

Makisu has been in use at Uber since early 2018, building over 1.5 thousand images every day across 4
different languages.

## Building Makisu

To build a Docker image that can perform builds:
```
make builder-image worker-image
```

## Building Makisu binary and build simple images

To get the makisu binary locally:
```
go get github.com/uber/makisu/makisu
```
If your Dockerfile doesn't have RUN, you can use makisu to build it without chroot or Docker daemon:
```
makisu build -t ${TAG} -dest ${TAR_PATH} ${CONTEXT}
```

## Makisu anywhere

To build Dockerfiles that contain RUN, Makisu still needs to run in a container.
The following snippet can be placed inside your `~/.bashrc` or `~/.zshrc`:
```shell
function makisu_build() {
    for last; do true; done
    cd $last

    makisu_version=0.1.0
    [ -z "$MAKISU_VERSION" ] || makisu_version=$MAKISU_VERSION

    docker run -i --rm --net host \
        -v /var/run/docker.sock:/docker.sock \
        -e DOCKER_HOST=unix:///docker.sock \
        -v $(pwd):/makisu-context \
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

## Makisu on Kubernetes

Makisu makes it easy to build images from a GitHub repository inside Kubernetes. A single pod (or job) is
created with a build container, which will perform the build, and a sidecar container, which will clone the repository and use the
`makisu-client` to trigger the build in the build container.

### Creating registry configuration

Makisu will need to have registry configuration mounted to push to a registry. The config format is described at the
bottom of this document. Once you have your configuration on your local filesystem, you will need to create the k8s secret:
```shell
$ kubectl create secret generic docker-registry-config --from-file=./registry.yaml
secret/docker-registry-config created
```

### Building Git repositories

Building an image from a GitHub repository is easy. We have a `makisu-client` image with a simple entrypoint this exact purpose.
If the repository is private, make sure your GitHub token is readable at `/makisu-secrets/github-token/github_token` inside
the client container. 
```shell
kubectl create secret generic github-token --from-file=./github_token
```
You will also need to mount the registry configuration after having created the secret for it. Below is a template to build a private
GitHub repository and push it to a registry.
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: imagebuilder
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: makisu-worker
        image: makisu-worker:6246b77
        imagePullPolicy: IfNotPresent
        args:
        - listen
        - -s
        - /makisu-socket/makisu.sock
        volumeMounts:
        - name: socket
          mountPath: /makisu-socket
        - name: context
          mountPath: /makisu-context
        - name: registry-config
          mountPath: /makisu-secrets/registry-config
      - name: makisu-manager
        image: makisu-client:6246b77
        imagePullPolicy: IfNotPresent
        args:
        - github.com/<your repository here>
        - --exit
        - build
        - -t=<your tag here>
        - --push=<your registry hostname here (eg: gcr.io)>
        - --registry-config=/makisu-secrets/registry-config/registry.yaml
        volumeMounts:
        - name: socket
          mountPath: /makisu-socket
        - name: context
          mountPath: /makisu-context
        - name: github-token
          mountPath: /makisu-secrets/github-token
      volumes:
      - name: socket
        emptyDir: {}
      - name: context
        emptyDir: {}
      - name: github-token
        secret:
          secretName: github-token
```
Once you have your job spec a simple `kubectl create -f job.yaml` will start your build. The job status will reflect whether or not the build failed.

### Distributed cache

A distributed layer cache maps each line of a Dockerfile to a tentative layer SHA stored in Docker registry. Using a layer
cache can significantly improve build performance.

To use the distributed caching feature of Makisu, the builder needs to be able to connect to a *cache id store*. Redis can
be used as a cache id store with the following Kubernetes job spec:

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

### Explicit caching

By default, Makisu will cache each directive in a Dockerfile. To avoid caching everything, the layer cache can be further optimized via
explicit caching with the `--commit=explicit` flag. Dockerfile directives may then be manually cached using the `#!COMMIT` annotation:

```Dockerfile
FROM node:8.1.3

ADD package.json package.json
ADD pre-build.sh

# A bunch of pre-install steps here.
...
...
...

# An expensive step we want to cache.
RUN npm install #!COMMIT
```

## Configuring Docker Registry

Makisu supports TLS and Basic Auth with Docker registry (Docker Hub, GCR, and private registries). It also contains a list of common root CA certs as a default.
Pass a custom configuration file to Makisu with `--registry-config=${PATH_TO_CONFIG}`.
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
  Security  security.Config `yaml:"security"`
}
```
To configure your own registry endpoint:
```yaml
[registry]:
  [repo]:
    security:
      tls:
        client:
          enabled: true
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
      tls:
        client:
          enabled: true
      basic:
        username: _json_key
        password: |-
            {
                <json here>
            }
```

## Comparison With Similar Tools

### Bazel

We were inspired by the Bazel project in early 2017. It is one of the first few tools that could build Docker compatible images without using Docker or 
any form of containerizer. It works very well with a subset of Docker build commands provided a Bazel build file, however, it does not support `RUN`,
making it hard to support more complex Dockerfiles.

### Kaniko

Kaniko provides good compatibility with Docker and executes build commands in userspace without the need for Docker daemon, although it must still run
inside a container. Kaniko is tightly integrated with Kubernetes, and manages secrets with Google Cloud Credential, making it a competent tool for
individual developers who are already using Kubernetes. However, Makisu's more flexible caching features make it optimal for higher build volume across
many developers.

### BuildKit

BuildKit depends on runc/containerd and supports parallel stage executions, whereas Makisu and most other tools execute Dockefile in order.
However, it still needs root privileges, which can be a security risk.