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

package cli

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/uber/makisu/lib/cache"
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

// Finds a way to get the dockerfile.
// If the context passed in is not a local path, then it will try to clone the
// git repo.
func (cmd BuildFlags) getDockerfile(contextDir string) ([]*dockerfile.Stage, error) {
	fi, err := os.Lstat(contextDir)
	if err != nil {
		return nil, fmt.Errorf("failed to lstat build context %s: %s", contextDir, err)
	} else if !fi.Mode().IsDir() {
		return nil, fmt.Errorf("build context provided is not a directory: %s", contextDir)
	}

	dockerfilePath := cmd.DockerfilePath
	if !path.IsAbs(dockerfilePath) {
		dockerfilePath = path.Join(contextDir, dockerfilePath)
	}

	log.Infof("Using build context: %s", contextDir)
	contents, err := ioutil.ReadFile(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to generate/find dockerfile in context: %s", err)
	}

	dockerfile, err := dockerfile.ParseFile(string(contents), cmd.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dockerfile: %v", err)
	}
	return dockerfile, nil
}

// pushImage pushes the specified image to docker registry.
// Exits with non-0 status code if it encounters an error.
func (cmd BuildFlags) pushImage(imageName image.Name) error {
	registryClient := registry.New(
		cmd.imageStore, imageName.GetRegistry(), imageName.GetRepository())
	if err := registryClient.Push(imageName.GetTag()); err != nil {
		return fmt.Errorf("failed to push image: %s", err)
	}
	log.Infof("Successfully pushed %s to %s", imageName, imageName.GetRegistry())
	return nil
}

// loadImage loads the image into the local docker daemon.
// This is only used for testing purposes.
func (cmd BuildFlags) loadImage(imageName image.Name) error {
	log.Infof("Loading image %s", imageName.ShortName())
	tarer := cli.NewDefaultImageTarer(cmd.imageStore)
	if tar, err := tarer.CreateTarReader(imageName); err != nil {
		return fmt.Errorf("failed to create tar of image: %s", err)
	} else if cli, err := cli.NewDockerClient(cmd.imageStore.SandboxDir, cmd.DockerHost,
		cmd.DockerScheme, cmd.DockerVersion, http.Header{}); err != nil {
		return fmt.Errorf("failed to create new docker client: %s", err)
	} else if err := cli.ImageTarLoad(context.Background(), tar); err != nil {
		return fmt.Errorf("failed to load image to local docker daemon: %s", err)
	}
	log.Infof("Successfully loaded image %s", imageName)
	return nil
}

// saveImage tars the image layers and manifests into a single tar, and saves that tar
// into <destination>.
func (cmd BuildFlags) saveImage(imageName image.Name) error {
	log.Infof("Saving image %s at location %s", imageName.ShortName(), cmd.Destination)
	tarer := cli.NewDefaultImageTarer(cmd.imageStore)
	if tar, err := tarer.CreateTarReadCloser(imageName); err != nil {
		return fmt.Errorf("failed to create a tarball from image layers and manifests: %s", err)
	} else if err := fileio.ReaderToFile(tar, cmd.Destination); err != nil {
		return fmt.Errorf("failed to write image tarball to destination %s: %s", cmd.Destination, err)
	}
	return nil
}

// cleanManifest removes specified image manifest from local filesystem.
func (cmd BuildFlags) cleanManifest(imageName image.Name) error {
	repo, tag := imageName.GetRepository(), imageName.GetTag()
	err := cmd.imageStore.Manifests.DeleteStoreFile(repo, tag)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete %s from manifest store: %s", imageName, err)
	}
	return nil
}

// getCacheManager inits and returns a transfer.CacheManager object.
func (cmd BuildFlags) getCacheManager(imageName image.Name) cache.Manager {
	if len(cmd.GetTargetRegistries()) != 0 {
		registryClient := registry.New(
			cmd.imageStore, cmd.GetTargetRegistries()[0], imageName.GetRepository())
		if cmd.RedisCacheAddress != "" {
			// If RedisCacheAddress is provided, init redis cache.
			log.Infof("Using redis at %s for cacheID storage", cmd.RedisCacheAddress)

			cacheIDStore, err := cache.NewRedisStore(cmd.RedisCacheAddress, cmd.CacheTTL)
			if err != nil {
				log.Errorf("Failed to connect to redis store: %s", err)
				cacheIDStore = nil
			}
			return cache.New(cacheIDStore, registryClient)
		} else if cmd.CacheTTL != 0 {
			// If redis cache address is not provided, and the cache ttl is not 0,
			// use the FSStore as a key-value store.
			fullpath := path.Join(cmd.imageStore.RootDir, pathutils.CacheKeyValueFileName)
			log.Infof("Using file at %s for cacheID storage", fullpath)

			cacheIDStore, err := cache.NewFSStore(
				fullpath, cmd.imageStore.SandboxDir, int64(cmd.CacheTTL))
			if err != nil {
				log.Errorf("Failed to init local cache ID store: %s", err)
				cacheIDStore = nil
			}
			return cache.New(cacheIDStore, registryClient)
		}
	}

	log.Infof("No registry or cache option provided, not using distributed cache")
	return cache.NewNoopCacheManager()
}

func (cmd BuildFlags) maybeBlacklistVarRun() error {
	if found, err := mountutils.ContainsMountpoint("/var/run"); err != nil {
		return err
	} else if found {
		pathutils.DefaultBlacklist = stringset.FromSlice(append(pathutils.DefaultBlacklist, "/var/run")).ToSlice()
		log.Warnf("Blacklisted /var/run because it contains a mountpoint inside. No changes of that directory " +
			"will be reflected in the final image.")
	}
	return nil
}
