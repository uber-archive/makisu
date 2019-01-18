package main

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/andres-erbsen/clock"
	"github.com/spf13/cobra"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/registry"
	"github.com/uber/makisu/lib/snapshot"
	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/tario"
)

var (
	registryURL string
	tag         string
	destination string
	cacerts     string
)

func main() {
	cmd := &cobra.Command{
		Use:                   "--dest <destination of rootfs> <image repository>",
		DisableFlagsInUseLine: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("Requires an image repository as argument")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			pullAndExtract(registryURL, args[0], tag, destination)
		},
	}

	cmd.PersistentFlags().StringVar(&registryURL, "registry", "index.docker.io", "The registry to pull the image from.")
	cmd.PersistentFlags().StringVar(&tag, "tag", "latest", "The tag of the image to pull.")
	cmd.PersistentFlags().StringVar(&destination, "dest", "rootfs", "The destination of the rootfs that we will untar the image to.")
	cmd.PersistentFlags().StringVar(&cacerts, "cacerts", "/registry-ca-certs.pem", "The location of the CA certs to use for TLS authentication with the registry.")

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

func pullAndExtract(registryURL, repository, tag, destination string) {
	store, err := storage.NewImageStore("/tmp/makisu-storage")
	if err != nil {
		panic(err)
	}

	registry.DefaultDockerHubConfiguration.Security.TLS.CA.Cert.Path = cacerts
	registry.ConfigurationMap[image.DockerHubRegistry] = make(registry.RepositoryMap)
	registry.ConfigurationMap[image.DockerHubRegistry]["library/*"] = registry.DefaultDockerHubConfiguration
	registry.ConfigurationMap[image.DockerHubRegistry][".*"] = registry.DefaultDockerHubConfiguration

	client := registry.New(store, registryURL, repository)
	manifest, err := client.Pull(tag)
	if err != nil {
		panic(err)
	}

	config := &image.Config{}
	if reader, err := store.Layers.GetStoreFileReader(manifest.Config.Digest.Hex()); err != nil {
		panic(err)
	} else if content, err := ioutil.ReadAll(reader); err != nil {
		panic(err)
	} else if err := json.Unmarshal(content, config); err != nil {
		panic(err)
	}

	memfs, err := snapshot.NewMemFS(clock.New(), destination, nil)
	if err != nil {
		panic(err)
	}

	for _, descriptor := range manifest.Layers {
		reader, err := store.Layers.GetStoreFileReader(descriptor.Digest.Hex())
		if err != nil {
			panic(fmt.Errorf("get reader from layer: %s", err))
		}
		gzipReader, err := tario.NewGzipReader(reader)
		if err != nil {
			panic(fmt.Errorf("create gzip reader for layer: %s", err))
		}
		log.Infof("* Processing FROM layer %s", descriptor.Digest.Hex())
		if err = memfs.UpdateFromTarReader(tar.NewReader(gzipReader), true); err != nil {
			panic(fmt.Errorf("untar reader: %s", err))
		}
	}
}
