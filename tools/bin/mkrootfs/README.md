# mkrootfs
----------

This tool was written to allow the creation of root filesystems from docker 
image names. The binary will download the docker image and untar them in the 
destination directory provided.
```
Usage:
  --dest <destination of rootfs> <image repository>

Flags:
      --cacerts string    The location of the CA certs to use for TLS authentication with the registry. (default "/registry-ca-certs.pem")
      --dest string       The destination of the rootfs that we will untar the image to. (default "rootfs")
  -h, --help              help for --dest
      --registry string   The registry to pull the image from. (default "index.docker.io")
      --tag string        The tag of the image to pull. (default "latest")

panic: Requires an image repository as argument
```
Example usage:
`mkrootfs library/alpine --dest alpine-rootfs --tag=3.6`

Note: The destination rootfs directory needs to be created and empty.
