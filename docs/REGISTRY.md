# Registry configuration

## General info
Makisu supports TLS and Basic Auth with Docker registry (Docker Hub, GCR, and private registries).
By default, TLS is enabled and makisu uses a list of common root CA certs to authenticate registry.
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

Configs can be passed in through the `--registry-config` flag, either as filepath, or as a raw json blob :
```
--registry-config='{"gcr.io": {"makisu-project/*": {"push_chunk": -1, "security": {"basic": {"username": "_json_key", "password": "<escaped key here>"}}}}}'
```
Consider using the great tool [yq](https://github.com/kislyuk/yq) to convert your yaml configuration into the blob that can be passed in.


## Examples
For the convenience to work with all public Docker Hub repositories including library/.*, a default config is provided:
```yaml
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

Example config for GCR:
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
Note: For the cert path, you can point to a directory containing your certificates. Makisu will then use all of the certs in that
directory for TLS verification.

## Cred helper

Makisu images (>= 0.1.8) contains [ECR](https://github.com/awslabs/amazon-ecr-credential-helper) and [GCR](https://github.com/GoogleCloudPlatform/docker-credential-gcr) cred helper binaries.
For ECR, you can export the following [variables](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html) and you might need to export `AWS_SDK_LOAD_CONFIG=true`.

If you encounter a certificate validation errors (ex: `x509: certificate signed by unknown authority`) you might want to export the following variable `SSL_CERT_DIR=/makisu-internal/certs/`.

Example AWS ECR config:
```yaml
"someawsregistry":
  "my-project/*":
    push_chunk: -1
    security:
      credsStore: ecr-login
```

Example GCR config:
```yaml
"gcr.io":
  "my-project/*":
    push_chunk: -1
    security:
      credsStore: gcr
```

NB: You need to put your config files (ex: aws config/credentials file) inside the /makisu-internal/ dir (and use env variable to specify their locations) in order for the helpers to find and use them when building your images.

### Using another cred helper

For now makisu handles ECR and GCR as lib instead of calling their binaries.
If you want to use another docker credentials helper, add its binary in the directory `/makisu-internal`, with a name matching `docker-credential-<cred-helper-name>`, then in your configuration:
```yaml
"example.com":
  "my-project/*":
    security:
      credsStore: <cred-helper-name>
```

## Handling `BLOB_UPLOAD_INVALID` and `BLOB_UPLOAD_UNKNOWN` errors

If you encounter these errors when pushing your image to a registry, try to use the `push_chunk: -1` option (some registries, despite implementing registry v2 do not support chunked upload, ECR and GCR being one example).