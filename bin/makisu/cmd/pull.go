package cmd

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/andres-erbsen/clock"
	"github.com/spf13/cobra"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/registry"
	"github.com/uber/makisu/lib/snapshot"
	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/tario"
)

type pullCmd struct {
	*cobra.Command

	cacerts        string
	extract        string
}

func getPullCmd() *pullCmd {
	pullCmd := &pullCmd{
		Command: &cobra.Command{
			Use:                   "pull --dest <destination of rootfs> <image>",
			DisableFlagsInUseLine: true,
			Short:                 "Pull docker image from registry into the storage directory of makisu.",
		},
	}
	pullCmd.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("Requires an image as argument")
		}
		return nil
	}
	pullCmd.Run = func(cmd *cobra.Command, args []string) {
		pullCmd.Pull(args[0])
	}

	pullCmd.PersistentFlags().StringVar(&pullCmd.cacerts, "cacerts", "/etc/ssl/certs", "The location of the CA certs to use for TLS authentication with the registry.")

	pullCmd.PersistentFlags().StringVar(&pullCmd.extract, "extract", "", "The destination of the rootfs that we will untar the image to.")
	return pullCmd
}

func (cmd *pullCmd) Pull(imageName string) {
	store, err := storage.NewImageStore("/tmp/makisu-storage")
	if err != nil {
		panic(err)
	}

	pullImage, err := image.ParseNameForPull(imageName)
	if err != nil {
		panic(fmt.Errorf("parse pull image %s: %s", pullImage, err))
	}
	client := registry.New(store, pullImage.GetRegistry(), pullImage.GetRepository())
	manifest, err := client.Pull(pullImage.GetTag())
	if err != nil {
		panic(fmt.Errorf("pull image %s: %s", imageName, err))
	}

	// If extract is not specified, exit here.
	if cmd.extract == "" {
		return
	}

	cmd.Extract(store, manifest)
}

func (cmd *pullCmd) Extract(store *storage.ImageStore, manifest *image.DistributionManifest) {
	config := &image.Config{}
	if reader, err := store.Layers.GetStoreFileReader(manifest.Config.Digest.Hex()); err != nil {
		panic(err)
	} else if content, err := ioutil.ReadAll(reader); err != nil {
		panic(err)
	} else if err := json.Unmarshal(content, config); err != nil {
		panic(err)
	}

	if _, err := os.Lstat(cmd.extract); err == nil || !os.IsNotExist(err) {
		panic(fmt.Errorf("destination rootfs directory should not exist"))
	} else if err := os.MkdirAll(cmd.extract, os.ModePerm); err != nil {
		panic(fmt.Errorf("failed to create destination rootfs directory: %s", err))
	}

	memfs, err := snapshot.NewMemFS(clock.New(), cmd.extract, nil)
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
		if err = memfs.UpdateFromTarReader(tar.NewReader(gzipReader), true); err != nil {
			panic(fmt.Errorf("untar reader: %s", err))
		}
	}
}
