package cmd

import (
	"archive/tar"
	"errors"
	"fmt"
	"os"

	"github.com/andres-erbsen/clock"
	"github.com/spf13/cobra"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/registry"
	"github.com/uber/makisu/lib/snapshot"
	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/tario"
)

type diffCmd struct {
	*cobra.Command
	ignoreModTime bool
}

func getDiffCmd() *diffCmd {
	diffCmd := &diffCmd{
		Command: &cobra.Command{
			Use:                   "diff <image name> <image name>",
			DisableFlagsInUseLine: true,
			Short:                 "Compare docker images from registry",
		},
	}

	diffCmd.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("Requires two image names as arguments")
		}
		return nil
	}

	diffCmd.Run = func(cmd *cobra.Command, args []string) {
		if err := diffCmd.Diff(args); err != nil {
			log.Error(err)
			os.Exit(1)
		}
	}

	diffCmd.PersistentFlags().BoolVar(&diffCmd.ignoreModTime, "ignoreModTime", true, "Ignore mod time of image when comparing images")
	return diffCmd
}

func (cmd *diffCmd) Diff(imagesFullName []string) error {
	log.Infof("ingore time? :%t", cmd.ignoreModTime)
	var pullImages []image.Name
	for _, imageFullName := range imagesFullName {
		pullImage, err := image.ParseNameForPull(imageFullName)
		if err != nil {
			return fmt.Errorf("parse image %s: %s", pullImage, err)
		}
		pullImages = append(pullImages, pullImage)
	}

	if err := initRegistryConfig(""); err != nil {
		return fmt.Errorf("failed to initialize registry configuration: %s", err)
	}

	store, err := storage.NewImageStore("/tmp/makisu-storage/")
	if err != nil {
		panic(err)
	}

	var memFSArr []*snapshot.MemFS
	for i, pullImage := range pullImages {
		client := registry.New(store, pullImage.GetRegistry(), pullImage.GetRepository())
		manifest, err := client.Pull(pullImage.GetTag())
		if err != nil {
			panic(err)
		}

		memfs, err := snapshot.NewMemFS(clock.New(), "/tmp/makisu-storage/", nil)
		if err != nil {
			panic(err)
		}

		for _, descriptor := range manifest.Layers {
			reader, err := store.Layers.GetStoreFileReader(descriptor.Digest.Hex())
			if err != nil {
				panic(fmt.Errorf("get reader from image %d layer: %s", i+1, err))
			}
			gzipReader, err := tario.NewGzipReader(reader)
			if err != nil {
				panic(fmt.Errorf("create gzip reader for layer: %s", err))
			}
			if err = memfs.UpdateFromTarReader(tar.NewReader(gzipReader), false); err != nil {
				panic(fmt.Errorf("untar image %d layer reader: %s", i+1, err))
			}
		}
		memFSArr = append(memFSArr, memfs)
	}

	log.Infof("* Diff two images")

	snapshot.CompareFS(memFSArr[0], memFSArr[1], pullImages[0], pullImages[1], cmd.ignoreModTime)
	return nil
}
