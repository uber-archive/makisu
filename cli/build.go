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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/uber/makisu/lib/builder"
	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/pathutils"
	"github.com/uber/makisu/lib/registry"
	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/tario"
	"github.com/uber/makisu/lib/utils"
	"github.com/uber/makisu/lib/utils/stringset"
	yaml "gopkg.in/yaml.v2"
)

// BuildFlags contains all of the flags for `makisu build ...`
type BuildFlags struct {
	DockerfilePath string `commander:"flag=f,The absolute path to the dockerfile"`
	Tag            string `commander:"flag=t,image tag (required)"`

	Arguments      map[string]string `commander:"flag=build-args,Arguments to the dockerfile as per the spec of ARG. Format is a json object."`
	Destination    string            `commander:"flag=dest,Destination of the image tar."`
	PushRegistries string            `commander:"flag=push,Push image after build to the comma-separated list of registries."`
	RegistryConfig string            `commander:"flag=registry-config,Registry configuration file for pulling and pushing images. Default configuration for DockerHub is used if not specified."`

	AllowModifyFS bool   `commander:"flag=modifyfs,Allow makisu to touch files outside of its own storage dir."`
	StorageDir    string `commander:"flag=storage,Directory that makisu uses for temp files and cached layers. Mount this path for better caching performance. If modifyfs is set, default to /makisu-storage; Otherwise default to /tmp/makisu-storage."`
	Blacklist     string `commander:"flag=blacklist,Comma separated list of files/directories. Makisu will omit all changes to these locations in the resulting docker images."`

	DockerHost    string `commander:"flag=docker-host,Docker host to load images to."`
	DockerVersion string `commander:"flag=docker-version,Version string for loading images to docker."`
	DockerScheme  string `commander:"flag=docker-scheme,Scheme for api calls to docker daemon."`
	DoLoad        bool   `commander:"flag=load,Load image after build."`

	RedisCacheAddress   string `commander:"flag=redis-cache-addr,The address of a redis cache server for cacheID to layer sha mapping."`
	CacheTTL            int    `commander:"flag=cache-ttl,The TTL of cacheID-sha mapping entries in seconds"`
	CompressionLevelStr string `commander:"flag=compression,Image compression level, could be 'no', 'speed', 'size', 'default'."`
	Commit              string `commander:"flag=commit,Set to explicit to only commit at steps with '#!COMMIT' annotations; Set to implicit to commit at every ADD/COPY/RUN step."`

	imageStore storage.ImageStore
}

func newBuildFlags() BuildFlags {
	return BuildFlags{
		DockerfilePath: "Dockerfile",
		Arguments:      map[string]string{},

		AllowModifyFS: false,
		StorageDir:    "",

		DockerHost:    utils.DefaultEnv("DOCKER_HOST", "unix:///var/run/docker.sock"),
		DockerVersion: utils.DefaultEnv("DOCKER_VERSION", "1.21"),
		DockerScheme:  utils.DefaultEnv("DOCKER_SCHEME", "http"),

		RedisCacheAddress:   "",
		CacheTTL:            7 * 24 * 3600,
		CompressionLevelStr: "default",

		Commit: "implicit",
	}
}

func (cmd *BuildFlags) postInit() error {
	if err := cmd.maybeBlacklistVarRun(); err != nil {
		return fmt.Errorf("failed to extend blacklist: %v", err)
	}

	if cmd.Blacklist != "" {
		newItems := strings.Split(cmd.Blacklist, ",")
		newBlacklist := append(pathutils.DefaultBlacklist, newItems...)
		pathutils.DefaultBlacklist = stringset.FromSlice(newBlacklist).ToSlice()
		log.Infof("Added %d new items to blacklist: %v", len(newItems), newItems)
	}

	if err := tario.SetCompressionLevel(cmd.CompressionLevelStr); err != nil {
		return fmt.Errorf("set compression level: %s", err)
	}

	if cmd.Commit != "explicit" && cmd.Commit != "implicit" {
		return fmt.Errorf("invalid commit option: %s", cmd.Commit)
	}

	if err := cmd.initRegistryGlobals(); err != nil {
		return fmt.Errorf("failed to initialize registry configuration: %v", err)
	}

	// Verify it's not runninng on Mac if modifyfs is true.
	if cmd.AllowModifyFS && runtime.GOOS == "darwin" {
		return fmt.Errorf("modifyfs option could erase fs and is not allowed on Mac")
	}

	// Configure default storage dir.
	if cmd.StorageDir == "" {
		if cmd.AllowModifyFS {
			cmd.StorageDir = pathutils.DefaultStorageDir
		} else {
			cmd.StorageDir = "/tmp/makisu-storage"
		}
	}

	// Init the image store.
	imageStore, err := storage.NewImageStore(cmd.StorageDir)
	if err != nil {
		return fmt.Errorf("failed to init image store: %s", err)
	}
	cmd.imageStore = imageStore

	// Verify storage dir is not child of internal dir.
	if pathutils.IsDescendantOfAny(cmd.StorageDir, []string{pathutils.DefaultInternalDir}) {
		return fmt.Errorf("storage dir cannot be under internal dir %s",
			pathutils.DefaultInternalDir)
	}
	return nil
}

func (cmd BuildFlags) initRegistryGlobals() error {
	if cmd.RegistryConfig == "" {
		// TODO(pourchet): Shouldn't we do this regardless of if a registry config was passed?
		registry.ConfigurationMap[image.DockerHubRegistry] = make(registry.RepositoryMap)
		registry.ConfigurationMap[image.DockerHubRegistry][".*"] = registry.DefaultDockerHubConfiguration
		return nil
	}
	data, err := ioutil.ReadFile(cmd.RegistryConfig)
	if err != nil {
		return fmt.Errorf("read registry config: %s", err)
	}
	config := make(registry.Map)
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("unmarshal registry config: %s", err)
	}
	for reg, repoConfig := range config {
		if _, ok := registry.ConfigurationMap[reg]; !ok {
			registry.ConfigurationMap[reg] = make(registry.RepositoryMap)
		}
		for repo, config := range repoConfig {
			registry.ConfigurationMap[reg][repo] = config
		}
	}
	return nil
}

func (cmd BuildFlags) forceCommit() bool {
	return cmd.Commit == "implicit"
}

// GetTargetRegistries returns the target registries that the image should
// be pushed to.
func (cmd BuildFlags) GetTargetRegistries() []string {
	registries := strings.Trim(cmd.PushRegistries, " \t\r\n")
	if registries == "" {
		return nil
	}
	return strings.Split(registries, ",")
}

func (cmd BuildFlags) getTargetImageName() (image.Name, error) {
	if cmd.Tag == "" {
		msg := "please specify a target image name: makisu build -t=(<registry:port>/)<repo>:<tag> ./"
		return image.Name{}, fmt.Errorf(msg)
	}

	// Parse the target's image name into its components.
	targetImageName := image.MustParseName(cmd.Tag)
	if len(cmd.GetTargetRegistries()) == 0 {
		return targetImageName, nil
	}

	// If the --push flag is specified we ignore the registry in the image name
	// and replace it with the first registry in the --push value. This will cause
	// all of the cache layers to go to that registry.
	return image.NewImageName(
		cmd.GetTargetRegistries()[0],
		targetImageName.GetRepository(),
		targetImageName.GetTag(),
	), nil
}

func (cmd BuildFlags) getBuildPlan(contextDir string, imageName image.Name) (*builder.BuildPlan, error) {
	// Remove image manifest if it already exists.
	if err := cmd.cleanManifest(imageName); err != nil {
		return nil, fmt.Errorf("failed to clean manifest: %v", err)
	}

	// Read in and parse dockerfile.
	contextDir, err := filepath.Abs(contextDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve context dir: %s", err)
	}
	dockerfile, err := cmd.getDockerfile(contextDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get dockerfile: %v", err)
	}

	// Create BuildContext.
	buildContext, err := context.NewBuildContext("/", contextDir, cmd.imageStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create initial build context: %s", err)
	}
	defer buildContext.Cleanup()

	// Init cache manager.
	cacheMgr := cmd.getCacheManager(imageName)

	// Create BuildPlan and validate it.
	return builder.NewBuildPlan(buildContext, imageName, cacheMgr,
		dockerfile, cmd.AllowModifyFS, cmd.forceCommit())
}

// Build image from the specified dockerfile.
// If -push is specified, will also push the image to those registries.
// If -load is specified, will load the image into the local docker daemon.
func (cmd BuildFlags) Build(contextDir string) error {
	log.Infof("Starting Makisu build (version=%s)", utils.BuildHash)
	imageName, err := cmd.getTargetImageName()
	if err != nil {
		return fmt.Errorf("failed to get target image name: %v", err)
	}

	buildPlan, err := cmd.getBuildPlan(contextDir, imageName)
	if err != nil {
		return fmt.Errorf("failed to create build plan: %s", err)
	}

	// Execute build plan.
	if _, err = buildPlan.Execute(); err != nil {
		return fmt.Errorf("failed to execute build plan: %s", err)
	}
	log.Infof("Successfully built image %s", imageName.ShortName())

	// Push image to registries that were specified in the --push flag.
	for _, registry := range cmd.GetTargetRegistries() {
		target := imageName.WithRegistry(registry)
		if err := cmd.pushImage(target); err != nil {
			return fmt.Errorf("failed to push image: %v", err)
		}
	}

	if cmd.Destination != "" {
		if err := cmd.saveImage(imageName); err != nil {
			return fmt.Errorf("failed to save image: %v", err)
		}
	}

	// Load image to local docker daemon.
	if cmd.DoLoad {
		if err := cmd.loadImage(imageName); err != nil {
			return fmt.Errorf("failed to load image: %v", err)
		}
	}

	log.Infof("Finished building %s", imageName.ShortName())
	return nil
}
