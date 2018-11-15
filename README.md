# Makisu
--------

A docker image build tool that is more flexible, faster at scale. This makes it easy to build
lots of docker images directly from a containerized environment such as Kubernetes. Specifically, Makisu:
* Uses a distributed layer cache to improve performance across a build cluster.
* Provides control over generated layers with keyword #!COMMIT, reducing number of layers in images.
* Requires no elevated privileges, making the build process portable.
* Is Docker compatible. Our dockerfile parser is opinionated in some scenarios, more details can be found [here](lib/parser/dockerfile/README.md).

Makisu has been in use at Uber for about a year, building over 1.5 thousand images every day, ranging 4 
different languages.

## Building Makisu

To build a docker image that can perform builds (makisu-builder/makisu-worker):
```
make builder-image worker-image
```

## Building Makisu binary and build simple images

To get the makisu-builder binary locally:
```
go get github.com/uber/makisu/bin/makisu-builder
```
If your dockerfile doesn't have RUN, you can use makisu-builder to build it without chroot or docker daemon:
```
makisu-builder build -t ${TAG} -dest ${TAR_PATH} ${CONTEXT}
```

## Makisu anywhere

To build dockerfiles that contain RUN, makisu still need to run in a container.
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
        gcr.io/makisu-project/makisu-builder:$makisu_version build \
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

Makisu makes it easy to build images from a github repository inside Kubernetes. The overall design is that a single Pod (or Job) gets
created with a builder/worker container that will perform the build, and a sidecar container that will clone the repo and use the 
`makisu-client` to trigger the build in the sidecar container.

### Creating your registry configuration

Makisu will need to have your registry configuration mounted if you want it to push to your registry. The config format is described at the 
bottom of this Readme. Once you have your configuration on your local filesystem, you will need to create the k8s secret:
```shell
$ kubectl create secret generic docker-registry-config --from-file=./registry.yaml
secret/docker-registry-config created
```

### Building Git repositories

Building your image from a GitHub repo is super easy, we have a `makisu-client` image that has a simple entrypoint which lets you do 
that easily. If your repo is private, make sure to have your github token readable at `/makisu-secrets/github-token/github_token` inside
the client container. 
```shell
kubectl create secret generic github-token --from-file=./github_token
```
You will also need to mount the registry configuration after having created the secret for it. Below is a template to build your private 
github repository and push it to your registry.
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
        - github.com/<your github repo>
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

If you want to use the distributed caching feature of Makisu, the builder needs to be able to connect to a "cache id store". In essence this lets us
map each line of a dockerfile to a tentative layer SHA that we will look for in your docker registry. In Kubernetes, spinning up a redis is simple:
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
Now all you need to do is pass in the `--redis-cache-addr=redis:6379` flag, and you should see significant improvements in your build speeds just like
you would if you repeated the same builds locally.

### Explicit caching

Makisu lets you decide which layers you want to cache during your build process. This helps shave some precious minutes off your build process because we
no longer need to capture the filesystem state, or push layers as often. You will need to make sure to pass in `--commit=explicit` to the build command. 
In this example we will tell Makisu to only produce a cache layer after an expensive build step:
```Dockerfile
FROM node:8.1.3

ADD package.json package.json
ADD pre-build.sh

# A bunch of pre install steps here
...
...
...

# This is the expensive step that we want to cache 
RUN npm install #!COMMIT
```

## Configuring Docker Registry

Makisu supports TLS and Basic Auth with Docker Registry (Docker Hub, GCR, and private registries). It also contains a list of common root CA certs as default.
Pass your configuration file to Makisu with flag `-registry-config=${PATH_TO_CONFIG}`.
```go
// Config contains docker registry client configuration.
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
            path: //path to cert//
          key:
            path: //path to key//
          passphrase
            path: //path to passphrase//
        ca:
          cert:
            path: //path to ca certs, appends to system certs. A list of common ca certs are used if empty//
      basic:
        username: //username//
        password: //password//
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
                // json here
            }
```

## Comparison with similar tools
Bazel:
We were inspired by the Bazel project in early 2017. It is one of the first few tools that could build Docker compatible images without using Docker or 
any form of containerizer. It works very well with a subset of Docker build commands provided a Bazel build file. It does not have the support for `RUN` 
which makes it hard to support the complexity and variety of Dockerfiles from our internal use cases.

Kaniko:
Kaniko provides good compatibility with Docker and executes build commands in userspace without the need for Docker daemon. It requires a containerized 
environment to perform the build. It has a tight integration with Kubernetes and uses Google Cloud Credential to manage secrets. Kaniko is a competent 
tool for individual developers who are using Kubernetes already. Makisu has minimum dependency: it requires a container at build time only when the provided 
Dockerfile contains `RUN`. To support Uber’s build volume and speed requirement, Makisu has more flexible caching features and is optimized for performance.

BuildKit:
BuildKit is one of the build tools that depends on runc/containerd. While Makisu and most other tools execute Dockefile in order, BuildKit supports 
parallel stage executions. However, it still needs privileges.

We are happy that more similar tools are created and open sourced to solve the issues our team has been seeing and we welcome collaborations from the community!
