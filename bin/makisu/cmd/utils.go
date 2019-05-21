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
	ctx "context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/uber/makisu/lib/cache"
	"github.com/uber/makisu/lib/cache/keyvalue"
	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/cli"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/fileio"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/mountutils"
	"github.com/uber/makisu/lib/parser/dockerfile"
	"github.com/uber/makisu/lib/pathutils"
	"github.com/uber/makisu/lib/registry"
	"github.com/uber/makisu/lib/utils/stringset"
)

func (cmd *buildCmd) initRegistryConfig() error {
	if cmd.registryConfig == "" {
		return nil
	}
	cmd.registryConfig = os.ExpandEnv(cmd.registryConfig)
	if err := registry.UpdateGlobalConfig(cmd.registryConfig); err != nil {
		return fmt.Errorf("init registry config: %s", err)
	}
	return nil
}

// Finds a way to get the dockerfile.
// If the context passed in is not a local path, then it will try to clone the
// git repo.
func (cmd *buildCmd) getDockerfile(contextDir string) ([]*dockerfile.Stage, error) {
	fi, err := os.Lstat(contextDir)
	if err != nil {
		return nil, fmt.Errorf("failed to lstat build context %s: %s", contextDir, err)
	} else if !fi.Mode().IsDir() {
		return nil, fmt.Errorf("build context provided is not a directory: %s", contextDir)
	}

	dockerfilePath := cmd.dockerfilePath
	if !path.IsAbs(dockerfilePath) {
		dockerfilePath = path.Join(contextDir, dockerfilePath)
	}

	log.Infof("Using build context: %s", contextDir)
	contents, err := ioutil.ReadFile(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to generate/find dockerfile in context: %s", err)
	}

	buildArgMap := make(map[string]string)
	for _, pair := range cmd.buildArgs {
		parts := strings.Split(pair, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("failed to parse build-arg %s: %s", pair, err)
		}
		buildArgMap[parts[0]] = parts[1]
	}

	dockerfile, err := dockerfile.ParseFile(string(contents), buildArgMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dockerfile: %s", err)
	}
	return dockerfile, nil
}

func (cmd *buildCmd) getTargetImageName() (image.Name, error) {
	if cmd.tag == "" {
		msg := "please specify a target image name: makisu build -t=(<registry:port>/)<repo>:<tag> ./"
		return image.Name{}, fmt.Errorf(msg)
	}

	// Parse the target's image name into its components.
	targetImageName := image.MustParseName(cmd.tag)
	if len(cmd.pushRegistries) == 0 {
		return targetImageName, nil
	}

	// If the --push flag is specified we ignore the registry in the image name
	// and replace it with the first registry in the --push value. This will cause
	// all of the cache layers to go to that registry.
	return image.NewImageName(
		cmd.pushRegistries[0],
		targetImageName.GetRepository(),
		targetImageName.GetTag(),
	), nil
}

// pushImage pushes the specified image to docker registry.
// Exits with non-0 status code if it encounters an error.
func pushImage(buildContext *context.BuildContext, imageName image.Name) error {
	registryClient := registry.New(
		buildContext.ImageStore, imageName.GetRegistry(), imageName.GetRepository())
	if err := registryClient.Push(imageName.GetTag()); err != nil {
		return fmt.Errorf("failed to push image: %s", err)
	}
	log.Infof("Successfully pushed %s to %s", imageName, imageName.GetRegistry())
	return nil
}

// loadImage loads the image into the local docker daemon.
// This is only used for testing purposes.
func (cmd *buildCmd) loadImage(buildContext *context.BuildContext, imageName image.Name) error {
	log.Infof("Loading image %s", imageName.ShortName())
	tarer := cli.NewDefaultImageTarer(buildContext.ImageStore)
	if tar, err := tarer.CreateTarReader(imageName); err != nil {
		return fmt.Errorf("failed to create tar of image: %s", err)
	} else if cli, err := cli.NewDockerClient(
		buildContext.ImageStore.SandboxDir, cmd.dockerHost,
		cmd.dockerScheme, cmd.dockerVersion, http.Header{}); err != nil {

		return fmt.Errorf("failed to create new docker client: %s", err)
	} else if err := cli.ImageTarLoad(ctx.Background(), tar); err != nil {
		return fmt.Errorf("failed to load image to local docker daemon: %s", err)
	}
	log.Infof("Successfully loaded image %s", imageName)
	return nil
}

// saveImage tars the image layers and manifests into a single tar, and saves that tar
// into <destination>.
func (cmd *buildCmd) saveImage(buildContext *context.BuildContext, imageName image.Name) error {
	log.Infof("Saving image %s at location %s", imageName.ShortName(), cmd.destination)
	tarer := cli.NewDefaultImageTarer(buildContext.ImageStore)
	if tar, err := tarer.CreateTarReadCloser(imageName); err != nil {
		return fmt.Errorf("failed to create a tarball from image layers and manifests: %s", err)
	} else if err := fileio.ReaderToFile(tar, cmd.destination); err != nil {
		return fmt.Errorf("failed to write image tarball to destination %s: %s", cmd.destination, err)
	}
	return nil
}

// cleanManifest removes specified image manifest from local filesystem.
func cleanManifest(buildContext *context.BuildContext, imageName image.Name) error {
	repo, tag := imageName.GetRepository(), imageName.GetTag()
	err := buildContext.ImageStore.Manifests.DeleteStoreFile(repo, tag)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete %s from manifest store: %s", imageName, err)
	}
	return nil
}

// newCacheManager inits and returns a cache manager object.
func (cmd *buildCmd) newCacheManager(buildContext *context.BuildContext, imageName image.Name) cache.Manager {
	var kvStore keyvalue.Store
	var err error
	if cmd.redisCacheAddress != "" {
		log.Infof("Using redis at %s for cacheID storage", cmd.redisCacheAddress)

		kvStore, err = keyvalue.NewRedisStore(cmd.redisCacheAddress, cmd.redisCacheTTL)
		if err != nil {
			log.Errorf("Failed to connect to redis store: %s", err)
		}
	} else if cmd.httpCacheAddress != "" {
		log.Infof("Using http server at %s for cacheID storage", cmd.httpCacheAddress)

		kvStore, err = keyvalue.NewHTTPStore(cmd.httpCacheAddress, cmd.httpCacheHeaders...)
		if err != nil {
			log.Errorf("Failed to instantiate cache id store: %s", err)
		}
	} else if cmd.localCacheTTL != 0 {
		fullpath := path.Join(buildContext.ImageStore.RootDir, pathutils.CacheKeyValueFileName)
		log.Infof("Using local file at %s for cacheID storage", fullpath)

		kvStore, err = keyvalue.NewFSStore(
			fullpath, buildContext.ImageStore.SandboxDir, cmd.localCacheTTL)
		if err != nil {
			log.Errorf("Failed to init local cache ID store: %s", err)
		}
	} else {
		log.Infof("No cache option provided, not using cache")
		return cache.NewNoopCacheManager()
	}

	var registryClient registry.Client
	if len(cmd.pushRegistries) == 0 {
		log.Infof("No registry information provided, using cached layers")
		registryClient = nil
	} else {
		registryAddr := cmd.pushRegistries[0]
		registryClient = registry.New(
			buildContext.ImageStore, registryAddr, imageName.GetRepository())
	}
	return cache.New(buildContext.ImageStore, kvStore, registryClient)
}

func maybeBlacklistVarRun() error {
	if found, err := mountutils.ContainsMountpoint("/var/run"); err != nil {
		return err
	} else if found {
		pathutils.DefaultBlacklist = stringset.FromSlice(
			append(pathutils.DefaultBlacklist, "/var/run")).ToSlice()
		log.Warnf("Blacklisted /var/run because it contains a mountpoint inside. " +
			"No changes of that directory will be reflected in the final image.")
	}
	return nil
}
