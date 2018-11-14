# Makisu
--------

A docker image build tool that is more flexible, faster at scale. This makes it easy to build
lots of docker images directly from a containerized environment such as Kubernetes. Specifically, Makisu:
* Uses a distributed layer cache to improve performance across a build cluster.
* Provides control over generated layers with keyword #!COMMIT, reducing number of layers in images.
* Requires no elevated privileges, making the build process portable.
* Is Docker compatible. Our dockerfile parser is opinionated in some scenarios, more details can be found [here](lib/parser/dockerfile/README.md).

## Makisu anywhere

The following snippet can be placed inside your `~/.bashrc` or `~/.zshrc`:
```shell
function makisu_build() {
    makisu_version=0.1.0
    [ -z "$MAKISU_VERSION" ] || makisu_version=$MAKISU_VERSION
    for last; do true; done
    cd $last
    docker run -i --rm --net host -v $(pwd):/context \
        -v /var/run/docker.sock:/docker.sock \
        -e DOCKER_HOST=unix:///docker.sock \
        gcr.io/makisu-project/makisu-builder:$makisu_version build --modifyfs=true --load ${@:1:-1} /context
    cd -
}
```
Now you can use `makisu_build` like you would use `docker build`:
```shell
$ makisu_build -t myimage .
```

## Makisu on Kubernetes

Makisu makes it super easy to build images from a github repository inside Kubernetes. The overall design is that a single Pod (or Job) gets
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
map each line of a dockerfile to a tentative layer SHA that we will look for in your docker registry. In Kubernetes, spinning up a redis is dead simple:
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

## Building Makisu

To build a docker image that can perform the builds (makisu-builder/makisu-worker) binary:
```
make image-builder image-worker
```

To get the makisu-builder binary locally and build _some_ images with no need for a containerizer:
```
go get github.com/uber/makisu/cmd/makisu-builder
```

## Local non-docker builds

If your dockerfile doesn't have RUN, you can use makisu-builder to build it without chroot, docker daemon or other containerizer.
To build a simple docker image and save it as a tar file:
```
makisu-builder build -t ${TAG} -dest ${TAR_PATH} ${CONTEXT}
```
To build a simple docker image and load into local docker daemon:
```
makisu-builder build -t ${TAG} -load ${CONTEXT}
```
To build a simple docker image and push to a registry:
```
makisu-builder build -t ${TAG} -push ${REGISTRY} ${CONTEXT}
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
        password: //password//
```
