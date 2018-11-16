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

	AllowModifyFS bool   `commander:"flag=modifyfs,Allow makisu to touch files outside of its own storage and sandbox dir."`
	StorageDir    string `commander:"flag=storage,Directory that makisu uses for cached layer files. Mount this path for better caching performance."`

	DockerHost    string `commander:"flag=docker-host,Docker host to load images to."`
	DockerVersion string `commander:"flag=docker-version,Version string for loading images to docker."`
	DockerScheme  string `commander:"flag=docker-scheme,Scheme for api calls to docker daemon."`
	DoLoad        bool   `commander:"flag=load,Load image after build."`

	RedisCacheAddress   string `commander:"flag=redis-cache-addr,The address of a redis cache server for cacheID to layer sha mapping."`
	RedisCacheTTL       int    `commander:"flag=redis-cache-ttl,The TTL of each cacheID-sha mapping entry in seconds."`
	FileCachePath       string `commander:"flag=file-cache-path,The path of a local file for cacheID to layer sha mapping. Used for testing only."`
	CompressionLevelStr string `commander:"flag=compression,Image compression level, could be 'no', 'speed', 'size', 'default'."`
	Commit              string `commander:"flag=commit,Set to explicit to only commit at steps with '#!COMMIT' annotations; Set to implicit to commit at every ADD/COPY/RUN step."`

	forceCommit bool
}

func newBuildFlags() BuildFlags {
	storageDir := pathutils.DefaultStorageDir
	if runtime.GOOS == "Darwin" {
		storageDir = "/tmp/makisu-storage"
	}
	return BuildFlags{
		DockerfilePath: "Dockerfile",
		Arguments:      map[string]string{},

		AllowModifyFS: false,
		StorageDir:    storageDir,

		DockerHost:    utils.DefaultEnv("DOCKER_HOST", "unix:///var/run/docker.sock"),
		DockerVersion: utils.DefaultEnv("DOCKER_VERSION", "1.21"),
		DockerScheme:  utils.DefaultEnv("DOCKER_SCHEME", "http"),

		RedisCacheAddress:   "",
		RedisCacheTTL:       7 * 24 * 3600,
		CompressionLevelStr: "default",

		Commit: "implicit",
	}
}

func (cmd *BuildFlags) postInit() error {
	if err := cmd.maybeBlacklistVarRun(); err != nil {
		return fmt.Errorf("failed to extend blacklist: %v", err)
	}

	if err := tario.SetCompressionLevel(cmd.CompressionLevelStr); err != nil {
		return fmt.Errorf("set compression level: %s", err)
	}

	// Configure commit option.
	switch cmd.Commit {
	case "explicit":
		cmd.forceCommit = false
	case "implicit":
		// forceCommit will make every step attampt to commit a layer.
		// Commit() is noop for steps other than ADD/COPY/RUN if they are not
		// after an uncommited RUN, so this won't generate extra empty layers.
		cmd.forceCommit = true
	default:
		return fmt.Errorf("invalid commit option: %s", cmd.Commit)
	}

	// Configure registries.
	if cmd.RegistryConfig != "" {
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
	}

	// Verify storage dir is not child of internal dir.
	if pathutils.IsDescendantOfAny(cmd.StorageDir, []string{pathutils.DefaultInternalDir}) {
		return fmt.Errorf("storage dir cannot be under internal dir %s",
			pathutils.DefaultInternalDir)
	}

	// Verify it's not runninng on Mac if modifyfs is true.
	if cmd.AllowModifyFS && runtime.GOOS == "darwin" {
		return fmt.Errorf("modifyfs option could erase fs and is not allowed on Mac")
	}

	return nil
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

// Build image from the specified dockerfile.
// If -push is specified, will also push the image to those registries.
// If -load is specified, will load the image into the local docker daemon.
func (cmd BuildFlags) Build(contextDir string) error {
	if cmd.Tag == "" {
		return fmt.Errorf("please specify a target image name: makisu build -t=(<registry:port>/)<repo>:<tag> ./")
	}

	// Parse the target's image name into its components.
	targetImageName := image.MustParseName(cmd.Tag)

	// If the --push flag is specified we ignore the registry in the image name
	// and replace it with the first registry in the --push value. This will cause
	// all of the cache layers to go to that registry.
	if len(cmd.GetTargetRegistries()) != 0 {
		targetImageName = image.NewImageName(
			cmd.GetTargetRegistries()[0],
			targetImageName.GetRepository(),
			targetImageName.GetTag(),
		)
	}

	// Init storage.
	imageStore, err := storage.NewImageStore(cmd.StorageDir)
	if err != nil {
		return fmt.Errorf("failed to init image store: %s", err)
	}

	// Remove image manifest if it already exists.
	if err := cmd.cleanManifest(targetImageName, imageStore); err != nil {
		return fmt.Errorf("failed to clean manifest: %v", err)
	}

	// Read in and parse dockerfile.
	contextDir, err = filepath.Abs(contextDir)
	if err != nil {
		return fmt.Errorf("failed to resolve context dir: %s", err)
	}
	dockerfile, err := cmd.getDockerfile(contextDir, imageStore)
	if err != nil {
		return fmt.Errorf("failed to get dockerfile: %v", err)
	}

	// Create BuildContext.
	buildContext, err := context.NewBuildContext("/", contextDir, imageStore)
	if err != nil {
		return fmt.Errorf("failed to create initial build context: %s", err)
	}
	defer buildContext.Cleanup()

	// Init cache manager.
	cacheMgr := cmd.getCacheManager(imageStore, targetImageName)

	// Create BuildPlan and validate it.
	buildPlan, err := builder.NewBuildPlan(
		buildContext, targetImageName, cacheMgr, dockerfile, cmd.AllowModifyFS, cmd.forceCommit)
	if err != nil {
		return fmt.Errorf("failed to create build plan: %s", err)
	}

	// Execute build plan.
	if _, err = buildPlan.Execute(); err != nil {
		return fmt.Errorf("failed to execute build plan: %s", err)
	}
	log.Infof("Successfully built image %s", targetImageName.ShortName())

	// Push image to registries that were specified in the --push flag.
	for _, registry := range cmd.GetTargetRegistries() {
		target := image.NewImageName(
			registry, targetImageName.GetRepository(), targetImageName.GetTag())
		if err := cmd.pushImage(target, imageStore); err != nil {
			return fmt.Errorf("failed to push image: %v", err)
		}
	}

	if cmd.Destination != "" {
		if err := cmd.saveImage(targetImageName, imageStore); err != nil {
			return fmt.Errorf("failed to save image: %v", err)
		}
	}

	// Load image to local docker daemon.
	if cmd.DoLoad {
		if err := cmd.loadImage(targetImageName, imageStore); err != nil {
			return fmt.Errorf("failed to load image: %v", err)
		}
	}

	log.Infof("Finished building %s", targetImageName.ShortName())
	return nil
}
