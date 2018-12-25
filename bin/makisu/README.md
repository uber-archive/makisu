# Makisu Usage Information
--------------------------

```
$ makisu build --help
Build docker image, optionally push to registries and/or load into docker daemon

Usage:
  makisu build -t=<image_tag> [flags] <context_path>

Flags:
  -f, --file string                     The absolute path to the dockerfile (default "Dockerfile")
  -t, --tag string                      Image tag (required)
      --push stringArray                Registry to push image to
      --registry-config string          Set build-time variables
      --dest string                     Destination of the image tar
      --build-arg stringArray           Argument to the dockerfile as per the spec of ARG. Format is "--build-arg <arg>=<value>"
      --modifyfs                        Allow makisu to modify files outside of its internal storage dir
      --commit string                   Set to explicit to only commit at steps with '#!COMMIT' annotations; Set to implicit to commit at every ADD/COPY/RUN step (default "implicit")
      --blacklist stringArray           Makisu will ignore all changes to these locations in the resulting docker images
      --local-cache-ttl duration        Time-To-Live for local cache (default 168h0m0s)
      --redis-cache-addr string         The address of a redis server for cacheID to layer sha mapping
      --redis-cache-ttl duration        Time-To-Live for redis cache (default 168h0m0s)
      --http-cache-addr string          The address of the http server for cacheID to layer sha mapping
      --http-cache-header stringArray   Request header for http cache server. Format is "--http-cache-header <header>:<value>"
      --docker-host string              Docker host to load images to (default "unix:///var/run/docker.sock")
      --docker-version string           Version string for loading images to docker (default "1.21")
      --docker-scheme string            Scheme for api calls to docker daemon (default "http")
      --load                            Load image into docker daemon after build. Requires access to docker socket at location defined by ${DOCKER_HOST}
      --storage string                  Directory that makisu uses for temp files and cached layers. Mount this path for better caching performance. If modifyfs is set, default to /makisu-storage; Otherwise default to /tmp/makisu-storage
      --compression string              Image compression level, could be 'no', 'speed', 'size', 'default' (default "default")
  -h, --help                            help for build

Global Flags:
      --cpu-profile         Profile the application
      --log-fmt string      The format of the logs. Valid values are "json" and "console" (default "json")
      --log-level string    Verbose level of logs. Valid values are "trace", "debug", "info", "warn", "error", "fatal" (default "info")
      --log-output string   The output file path for the logs. Set to "stdout" to output to stdout (default "stdout")

$ makisu version
v0.1.2
```
