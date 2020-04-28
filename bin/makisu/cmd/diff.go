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
		if err := diffCmd.Diff(args[0], args[1]); err != nil {
			log.Error(err)
			os.Exit(1)
		}
	}

	return diffCmd
}

func (cmd *diffCmd) Diff(image1FullName, image2FullName string) error {
	pullImage1, err := image.ParseNameForPull(image1FullName)
	if err != nil {
		return fmt.Errorf("parse the first image %s: %s", pullImage1, err)
	}

	pullImage2, err := image.ParseNameForPull(image2FullName)
	if err != nil {
		return fmt.Errorf("parse the second image %s: %s", pullImage2, err)
	}

	var pullImages []image.Name
	pullImages = append(pullImages, pullImage1)
	pullImages = append(pullImages, pullImage2)

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
	snapshot.CompareFS(memFSArr[0], memFSArr[1], pullImage1, pullImage2)
	return nil
}
