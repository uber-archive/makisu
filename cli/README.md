# Makisu Usage Information
--------------------------

```
$ makisu --help
Usage of makisu:
  -cpu-profile
    	Profile the application. (type: bool, default: false)
  -help
    	Display usage information for Makisu. (type: bool, default: false)
  -log-fmt
    	The format of the logs. (type: string, default: "json")
  -log-level
    	The level at which to log. (type: string, default: "info")
  -log-output
    	The output file path for the logs. (type: string, default: "stdout")

Sub-Commands:
  build  |  Builds a docker image from a build context and a dockerfile.

$ makisu build --help
Usage of makisu build:
  -blacklist
    	Comma separated list of files/directories. Makisu will omit all changes to these locations in the resulting docker images. (type: string, default: "")
  -build-args
    	Arguments to the dockerfile as per the spec of ARG. Format is a json object. (type: map, default: {})
  -commit
    	Set to explicit to only commit at steps with '#!COMMIT' annotations; Set to implicit to commit at every ADD/COPY/RUN step. (type: string, default: "implicit")
  -compression
    	Image compression level, could be 'no', 'speed', 'size', 'default'. (type: string, default: "default")
  -dest
    	Destination of the image tar. (type: string, default: "")
  -docker-host
    	Docker host to load images to. (type: string, default: "unix:///var/run/docker.sock")
  -docker-scheme
    	Scheme for api calls to docker daemon. (type: string, default: "http")
  -docker-version
    	Version string for loading images to docker. (type: string, default: "1.21")
  -f	The absolute path to the dockerfile. (type: string, default: "Dockerfile")
  -load
    	Load image into docker daemon after build. Requires access to docker socket at location defined by ${DOCKER_HOST}. (type: bool, default: false)
  -local-cache-ttl
    	Time-To-Live for local cache. (type: string, default: 168h)
  -modifyfs
    	Allow makisu to touch files outside of its own storage dir. (type: bool, default: false)
  -push
    	Push image after build to the comma-separated list of registries. (type: string, default: "")
  -redis-cache-addr
    	The address of a redis cache server for cacheID to layer sha mapping. (type: string, default: "")
  -redis-cache-ttl
    	Time-To-Live for redis cache. (type: string, default: 168h)
  -registry-config
    	Registry configuration file for pulling and pushing images. Default configuration for DockerHub is used if not specified. (type: string, default: "")
  -storage
    	Directory that makisu uses for temp files and cached layers. Mount this path for better caching performance. If modifyfs is set, default to /makisu-storage; Otherwise default to /tmp/makisu-storage. (type: string, default: "")
  -t	image tag (required). (type: string, default: "")

$ makisu version
v0.1.2
```
