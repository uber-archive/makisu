# Cache Configuration

Makisu supports distributed cache, which can significantly reduce build time, by up to 90% for some of Uber's code repos.
Makisu caches docker image layers both locally and in docker registry (if --push parameter is provided).
It uses a separate key-value store to map lines of a Dockerfile to names of the layers.

For cache key-value store, Makisu supports 3 choices:
local file cache, redis based distributed cache, and generic HTTP based distributed cache.

## Local file cache

If no cache options are provided, local file cache is used by default.
To configure local file cache TTL:
```
--local-cache-ttl duration        Time-To-Live for local cache (default 168h0m0s)
```
To disable it, set ttl to 0s.

## Redis cache

To configure redis cache, use the following options:
```
--redis-cache-addr string         The address of a redis server for cacheID to layer sha mapping
--redis-cache-ttl duration        Time-To-Live for redis cache (default 168h0m0s)
```

## HTTP cache

To configure HTTP cache, use the following options:
```
--http-cache-addr string          The address of the http server for cacheID to layer sha mapping
--http-cache-header stringArray   Request header for http cache server. Format is "--http-cache-header <header>:<value>"
```

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
