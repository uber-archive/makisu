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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/registry"
	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/tario"
	"github.com/uber/makisu/lib/utils"

	"github.com/spf13/cobra"
)

type pushCmd struct {
	*cobra.Command

	tag string

	pushRegistries []string
	replicas       []string
	registryConfig string
}

func getPushCmd() *pushCmd {
	pushCmd := &pushCmd{
		Command: &cobra.Command{
			Use:                   "push -t=<image_tag> [flags] <image_tar_path>",
			DisableFlagsInUseLine: true,
			Short:                 "Push docker image to registries",
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
	if err := initRegistryConfig(cmd.registryConfig); err != nil {
		return fmt.Errorf("failed to initialize registry configuration: %s", err)
	}

	return nil
}

// Push image tar to docker registries.
func (cmd *pushCmd) Push(imageTarPath string) error {
	log.Infof("Starting Makisu push (version=%s)", utils.BuildHash)

	imageName, err := cmd.getTargetImageName()
	if err != nil {
		return err
	}

	// TODO: make configurable?
	store, err := storage.NewImageStore("/tmp/makisu-storage")
	if err != nil {
		return fmt.Errorf("unable to create internal store: %s", err)
	}

	if err := cmd.loadImageTarIntoStore(store, imageName, cmd.replicas, imageTarPath); err != nil {
		return fmt.Errorf("unable to import image: %s", err)
	}

	// Push image to registries that were specified in the --push flag.
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
		msg := "please specify a target image name: push -t=<image_tag> [flags] <image_tar_path>"
		return image.Name{}, errors.New(msg)
	}

	return image.MustParseName(cmd.tag), nil
}

func (cmd *pushCmd) loadImageTarIntoStore(
	store *storage.ImageStore, imageName image.Name, replicas []string, imageTarPath string) error {

	if err := cmd.importTar(store, imageName, replicas, imageTarPath); err != nil {
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

// importTar imports an image, as a tar, to the image store.
func (cmd *pushCmd) importTar(
	store *storage.ImageStore, imageName image.Name, replicas []string, tarPath string) error {

	repo, tag := imageName.GetRepository(), imageName.GetTag()

	// Extract tar into temporary directory.
	dir := filepath.Join(store.SandboxDir, repo, tag)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create unpack directory: %s", err)
	}
	defer os.RemoveAll(dir)

	reader, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("open tar file: %s", err)
	}
	defer reader.Close()

	if err := tario.Untar(reader, dir); err != nil {
		return fmt.Errorf("unpack tar: %s", err)
	}

	// Read manifest.
	exportManifestPath := filepath.Join(dir, "manifest.json")
	exportManifestData, err := ioutil.ReadFile(exportManifestPath)
	if err != nil {
		return fmt.Errorf("read export manifest: %s", err)
	}

	var exportManifests []image.ExportManifest
	if err := json.Unmarshal(exportManifestData, &exportManifests); err != nil {
		return fmt.Errorf("unmarshal export manifest: %s", err)
	}

	for _, exportManifest := range exportManifests {
		// Import extracted dir content into image store -- {sha}.json.
		configPath := filepath.Join(dir, exportManifest.Config.String())

		configInfo, err := os.Stat(configPath)
		if err != nil {
			return fmt.Errorf("lookup config file info: %s", err)
		}

		configReader, err := os.Open(configPath)
		if err != nil {
			return fmt.Errorf("open config json: %s", err)
		}
		defer configReader.Close()
		configDigest, err := image.NewDigester().FromReader(configReader)

		if err := store.Layers.LinkStoreFileFrom(
			configDigest.Hex(), configPath); err != nil && !os.IsExist(err) {

			return fmt.Errorf("commit config to store: %s", err)
		}

		// Import extracted dir content into image store -- {sha}/layer.tar.
		var layers []image.Descriptor
		for _, layer := range exportManifest.Layers {
			layerPath := path.Join(dir, layer.String())

			layerInfo, err := os.Stat(layerPath)
			if err != nil {
				return fmt.Errorf("lookup layer file info: %s", err)
			}

			layerReader, err := os.Open(layerPath)
			if err != nil {
				return fmt.Errorf("open layer tar: %s", err)
			}
			defer layerReader.Close()
			layerDigest, err := image.NewDigester().FromReader(layerReader)

			if err := store.Layers.LinkStoreFileFrom(
				layerDigest.Hex(), layerPath); err != nil && !os.IsExist(err) {

				return fmt.Errorf("commit layer to store: %s", err)
			}

			layers = append(layers, image.Descriptor{
				MediaType: image.MediaTypeLayer,
				Size:      layerInfo.Size(),
				Digest:    layerDigest,
			})
		}

		// Import extracted dir content into image store -- manifest.json.
		distManifest := image.DistributionManifest{
			SchemaVersion: 2,
			MediaType:     image.MediaTypeManifest,
			Config: image.Descriptor{
				MediaType: image.MediaTypeConfig,
				Size:      configInfo.Size(),
				Digest:    configDigest,
			},
			Layers: layers,
		}
		store.SaveManifest(distManifest, imageName)

		for _, replica := range replicas {
			parsed := image.MustParseName(replica)
			store.SaveManifest(distManifest, parsed)
		}
	}

	return nil
}
