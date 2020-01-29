//  Copyright (c) 2018 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/uber/makisu/lib/builder"
	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/cli"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/pathutils"
	"github.com/uber/makisu/lib/registry"
	"github.com/uber/makisu/lib/snapshot"
	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/tario"
	"github.com/uber/makisu/lib/utils"
	"github.com/uber/makisu/lib/utils/stringset"

	"github.com/spf13/cobra"
)

type pushCmd struct {
	*cobra.Command

	tag            string

	pushRegistries []string
	replicas       []string
	registryConfig string
}

func getPushCmd() *pushCmd {
	pushCmd := &pushCmd{
		Command: &cobra.Command{
			Use: "push -t=<image_tag> [flags] <image_tar_path>",
			DisableFlagsInUseLine: true,
			Short: "Push docker image to registries",
		},
	}
	pushCmd.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("Requires image tar path as argument")
		}
		return nil
	}
	pushCmd.Run = func(cmd *cobra.Command, args []string) {
		if err := pushCmd.processFlags(); err != nil {
			log.Errorf("failed to process flags: %s", err)
			os.Exit(1)
		}

		if err := pushCmd.Push(args[0]); err != nil {
			log.Error(err)
			os.Exit(1)
		}
	}

	pushCmd.PersistentFlags().StringVarP(&pushCmd.tag, "tag", "t", "", "Image tag (required)")

	pushCmd.PersistentFlags().StringArrayVar(&pushCmd.pushRegistries, "push", nil, "Registry to push image to")
	pushCmd.PersistentFlags().StringArrayVar(&pushCmd.replicas, "replica", nil, "Push targets with alternative full image names \"<registry>/<repo>:<tag>\"")
	pushCmd.PersistentFlags().StringVar(&pushCmd.registryConfig, "registry-config", "", "Set build-time variables")

	pushCmd.MarkFlagRequired("tag")
	pushCmd.Flags().SortFlags = false
	pushCmd.PersistentFlags().SortFlags = false

	return pushCmd
}

func (cmd *pushCmd) processFlags() error {
	if err := cmd.initRegistryConfig(); err != nil {
		return fmt.Errorf("failed to initialize registry configuration: %s", err)
	}

	return nil
}

// Push image tar to docker registries
func (cmd *pushCmd) Push(imageTarPath string) error {
	log.Infof("Starting Makisu push (version=%s)", utils.BuildHash)

	imageName, err := cmd.getTargetImageName()
	if err != nil {
		return err
	}

	store, err := storage.NewImageStore("/tmp/makisu-storage")  // TODO make configurable?
	if err != nil {
		return fmt.Errorf("unable to create internal store: %s", err)
	}

	if err := cmd.loadImageTarIntoStore(store, imageTarPath); err != nil {
		return fmt.Errorf("unable to import image: %s", err)
	}

	// Push image to registries that were specified in the --push flag.
	// TODO figure out where in the build process we map imageName X with the below names for push
	for _, registry := range cmd.pushRegistries {
		target := imageName.WithRegistry(registry)
		if err := cmd.pushImage(store, target); err != nil {
			return fmt.Errorf("failed to push image: %s", err)
		}
	}
	for _, replica := range cmd.replicas {
		target := image.MustParseName(replica)
		if err := cmd.pushImage(store, target); err != nil {
			return fmt.Errorf("failed to push image: %s", err)
		}
	}

	log.Infof("Finished pushing %s", imageName.ShortName())
	return nil
}

func (cmd *pushCmd) getTargetImageName() (image.Name, error) {
	if cmd.tag == "" {
		// TODO message
		return image.Name{}, fmt.Errorf("please specify a target image name: makisu build -t=(<registry:port>/)<repo>:<tag> ./")
	}

	return image.MustParseName(cmd.tag), nil
}

func (cmd *pushCmd) loadImageTarIntoStore(store *storage.ImageStore, imageTarPath string, repository string) error {
	tarer := cli.NewDefaultImageTarer(store)
	if err := tarer.WriteTar(imageTarPath, ...); err != nil {
		return fmt.Errorf("import image tar: %s", err)
	}

	return nil
}

func (cmd *pushCmd) pushImage(store *storage.ImageStore, imageName image.Name) error {
	registryClient := registry.New(store, imageName.GetRegistry(), imageName.GetRepository())
	if err := registryClient.Push(imageName.GetTag()); err != nil {
		return fmt.Errorf("failed to push image: %s", err)
	}
	log.Infof("Successfully pushed %s to %s", imageName, imageName.GetRegistry())
	return nil
}
