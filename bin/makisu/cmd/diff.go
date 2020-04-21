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

func (cmd *diffCmd) Diff(image1, image2 string) error {
	pullImage1, err := image.ParseNameForPull(image1)
	if err != nil {
		return fmt.Errorf("parse the first image %s: %s", pullImage1, err)
	}

	pullImage2, err := image.ParseNameForPull(image2)
	if err != nil {
		return fmt.Errorf("parse the second image %s: %s", pullImage2, err)
	}

	if err := initRegistryConfig(""); err != nil {
		return fmt.Errorf("failed to initialize registry configuration: %s", err)
	}

	store1, err := storage.NewImageStore("/tmp/makisu-storage/image1")
	if err != nil {
		panic(err)
	}

	store2, err := storage.NewImageStore("/tmp/makisu-storage/image2")
	if err != nil {
		panic(err)
	}

	client1 := registry.New(store1, pullImage1.GetRegistry(), pullImage1.GetRepository())
	client2 := registry.New(store2, pullImage2.GetRegistry(), pullImage2.GetRepository())

	manifest1, err := client1.Pull(pullImage1.GetTag())
	if err != nil {
		panic(err)
	}

	manifest2, err := client2.Pull(pullImage2.GetTag())
	if err != nil {
		panic(err)
	}

	memfs1, err := snapshot.NewMemFS(clock.New(), "/tmp/makisu-storage/diff1", nil)
	if err != nil {
		panic(err)
	}

	for _, descriptor := range manifest1.Layers {
		reader, err := store1.Layers.GetStoreFileReader(descriptor.Digest.Hex())
		if err != nil {
			panic(fmt.Errorf("get reader from first image layer: %s", err))
		}
		gzipReader, err := tario.NewGzipReader(reader)
		if err != nil {
			panic(fmt.Errorf("create gzip reader for layer: %s", err))
		}
		if err = memfs1.UpdateFromTarReader(tar.NewReader(gzipReader), false); err != nil {
			panic(fmt.Errorf("untar first image layer reader: %s", err))
		}
	}

	memfs2, err := snapshot.NewMemFS(clock.New(), "/tmp/makisu-storage/diff2", nil)
	if err != nil {
		panic(err)
	}

	for _, descriptor := range manifest2.Layers {
		reader, err := store2.Layers.GetStoreFileReader(descriptor.Digest.Hex())
		if err != nil {
			panic(fmt.Errorf("get reader from second image layer: %s", err))
		}
		gzipReader, err := tario.NewGzipReader(reader)
		if err != nil {
			panic(fmt.Errorf("create gzip reader for layer: %s", err))
		}
		if err = memfs2.UpdateFromTarReader(tar.NewReader(gzipReader), false); err != nil {
			panic(fmt.Errorf("untar second image layer reader: %s", err))
		}
	}

	log.Infof("* Diff two images")
	snapshot.CompareFS(memfs1, memfs2, pullImage1, pullImage2)
	return nil
}
