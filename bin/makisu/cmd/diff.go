package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/registry"
	"github.com/uber/makisu/lib/storage"
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
	// Pull the image here.
	pullImage1, err := image.ParseNameForPull(image1)
	if err != nil {
		return fmt.Errorf("parse the first image %s: %s", pullImage1, err)
	}

	pullImage2, err := image.ParseNameForPull(image2)
	if err != nil {
		return fmt.Errorf("parse the second image %s: %s", pullImage2, err)
	}

	store, err := storage.NewImageStore("/tmp/makisu-diff-storage")
	if err != nil {
		panic(err)
	}

	registry.DefaultDockerHubConfiguration.Security.TLS.CA.Cert.Path = "/etc/ssl/a"
	registry.ConfigurationMap[image.DockerHubRegistry] = make(registry.RepositoryMap)
	registry.ConfigurationMap[image.DockerHubRegistry]["library/*"] = registry.DefaultDockerHubConfiguration

	client1 := registry.New(store, pullImage1.GetRegistry(), pullImage1.GetRepository())
	client2 := registry.New(store, pullImage2.GetRegistry(), pullImage2.GetRepository())
	_, err = client1.Pull(pullImage1.GetTag())
	if err != nil {
		panic(err)
	}

	_, err = client2.Pull(pullImage2.GetTag())
	if err != nil {
		panic(err)
	}

	//TODO(xiaoweic): diff the image here.
	return nil
}
