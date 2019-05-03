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
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/pathutils"
	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/tario"
	"github.com/uber/makisu/lib/utils"
	"github.com/uber/makisu/lib/utils/stringset"

	"github.com/spf13/cobra"
)

type buildCmd struct {
	*cobra.Command

	dockerfilePath string
	tag            string

	pushRegistries []string
	replicas       []string
	registryConfig string
	destination    string

	buildArgs     []string
	allowModifyFS bool
	commit        string
	blacklists    []string

	localCacheTTL     time.Duration
	redisCacheAddress string
	redisCacheTTL     time.Duration
	httpCacheAddress  string
	httpCacheHeaders  []string

	dockerHost    string
	dockerVersion string
	dockerScheme  string
	doLoad        bool

	storageDir       string
	compressionLevel string

	preserveRoot bool
}

func getBuildCmd() *buildCmd {
	buildCmd := &buildCmd{
		Command: &cobra.Command{
			Use:                   "build -t=<image_tag> [flags] <context_path>",
			DisableFlagsInUseLine: true,
			Short:                 "Build docker image, optionally push to registries and/or load into docker daemon",
		},
	}
	buildCmd.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("Requires build context as argument")
		}
		return nil
	}
	buildCmd.Run = func(cmd *cobra.Command, args []string) {
		if err := buildCmd.processFlags(); err != nil {
			log.Errorf("failed to process flags: %s", err)
			os.Exit(1)
		}

		if err := buildCmd.Build(args[0]); err != nil {
			log.Error(err)
			os.Exit(1)
		}
	}

	buildCmd.PersistentFlags().StringVarP(&buildCmd.dockerfilePath, "file", "f", "Dockerfile", "The absolute path to the dockerfile")
	buildCmd.PersistentFlags().StringVarP(&buildCmd.tag, "tag", "t", "", "Image tag (required)")

	buildCmd.PersistentFlags().StringArrayVar(&buildCmd.pushRegistries, "push", nil, "Registry to push image to")
	buildCmd.PersistentFlags().StringArrayVar(&buildCmd.replicas, "replica", nil, "Push targets with alternative full image names \"<registry>/<repo>:<tag>\"")
	buildCmd.PersistentFlags().StringVar(&buildCmd.registryConfig, "registry-config", "", "Set build-time variables")
	buildCmd.PersistentFlags().StringVar(&buildCmd.destination, "dest", "", "Destination of the image tar")

	buildCmd.PersistentFlags().StringArrayVar(&buildCmd.buildArgs, "build-arg", nil, "Argument to the dockerfile as per the spec of ARG. Format is \"--build-arg <arg>=<value>\"")
	buildCmd.PersistentFlags().BoolVar(&buildCmd.allowModifyFS, "modifyfs", false, "Allow makisu to modify files outside of its internal storage dir")
	buildCmd.PersistentFlags().StringVar(&buildCmd.commit, "commit", "implicit", "Set to explicit to only commit at steps with '#!COMMIT' annotations; Set to implicit to commit at every ADD/COPY/RUN step")
	buildCmd.PersistentFlags().StringArrayVar(&buildCmd.blacklists, "blacklist", nil, "Makisu will ignore all changes to these locations in the resulting docker images")

	buildCmd.PersistentFlags().DurationVar(&buildCmd.localCacheTTL, "local-cache-ttl", time.Hour*168, "Time-To-Live for local cache")
	buildCmd.PersistentFlags().StringVar(&buildCmd.redisCacheAddress, "redis-cache-addr", "", "The address of a redis server for cacheID to layer sha mapping")
	buildCmd.PersistentFlags().DurationVar(&buildCmd.redisCacheTTL, "redis-cache-ttl", time.Hour*168, "Time-To-Live for redis cache")
	buildCmd.PersistentFlags().StringVar(&buildCmd.httpCacheAddress, "http-cache-addr", "", "The address of the http server for cacheID to layer sha mapping")
	buildCmd.PersistentFlags().StringArrayVar(&buildCmd.httpCacheHeaders, "http-cache-header", nil, "Request header for http cache server. Format is \"--http-cache-header <header>:<value>\"")

	buildCmd.PersistentFlags().StringVar(&buildCmd.dockerHost, "docker-host", utils.DefaultEnv("DOCKER_HOST", "unix:///var/run/docker.sock"), "Docker host to load images to")
	buildCmd.PersistentFlags().StringVar(&buildCmd.dockerVersion, "docker-version", utils.DefaultEnv("DOCKER_VERSION", "1.21"), "Version string for loading images to docker")
	buildCmd.PersistentFlags().StringVar(&buildCmd.dockerScheme, "docker-scheme", utils.DefaultEnv("DOCKER_SCHEME", "http"), "Scheme for api calls to docker daemon")
	buildCmd.PersistentFlags().BoolVar(&buildCmd.doLoad, "load", false, "Load image into docker daemon after build. Requires access to docker socket at location defined by ${DOCKER_HOST}")

	buildCmd.PersistentFlags().StringVar(&buildCmd.storageDir, "storage", "", "Directory that makisu uses for temp files and cached layers. Mount this path for better caching performance. If modifyfs is set, default to /makisu-storage; Otherwise default to /tmp/makisu-storage")
	buildCmd.PersistentFlags().StringVar(&buildCmd.compressionLevel, "compression", "default", "Image compression level, could be 'no', 'speed', 'size', 'default'")

	buildCmd.PersistentFlags().BoolVar(&buildCmd.preserveRoot, "preserve-root", false, "Copy / in the storage dir and copy it back after build.")

	buildCmd.MarkFlagRequired("tag")
	buildCmd.Flags().SortFlags = false
	buildCmd.PersistentFlags().SortFlags = false

	return buildCmd
}

func (cmd *buildCmd) processFlags() error {
	if err := maybeBlacklistVarRun(); err != nil {
		return fmt.Errorf("failed to extend blacklist: %s", err)
	}

	if len(cmd.blacklists) != 0 {
		newBlacklist := append(pathutils.DefaultBlacklist, cmd.blacklists...)
		pathutils.DefaultBlacklist = stringset.FromSlice(newBlacklist).ToSlice()
		log.Infof("Added %d new items to blacklist: %v", len(cmd.blacklists), cmd.blacklists)
	}

	if err := tario.SetCompressionLevel(cmd.compressionLevel); err != nil {
		return fmt.Errorf("set compression level: %s", err)
	}

	if cmd.commit != "explicit" && cmd.commit != "implicit" {
		return fmt.Errorf("invalid commit option: %s", cmd.commit)
	}

	if err := cmd.initRegistryConfig(); err != nil {
		return fmt.Errorf("failed to initialize registry configuration: %s", err)
	}

	// If modifyfs is true, verify it's not runninng on Mac.
	if cmd.allowModifyFS && runtime.GOOS == "darwin" {
		return fmt.Errorf("modifyfs option could erase fs and is not allowed on Mac")
	}

	// Configure default storage dir.
	if cmd.storageDir == "" {
		if cmd.allowModifyFS {
			cmd.storageDir = pathutils.DefaultStorageDir
		} else {
			cmd.storageDir = "/tmp/makisu-storage"
		}
	}

	// Verify storage dir is not child of internal dir.
	if pathutils.IsDescendantOfAny(cmd.storageDir, []string{pathutils.DefaultInternalDir}) {
		return fmt.Errorf("storage dir cannot be under internal dir %s",
			pathutils.DefaultInternalDir)
	}
	return nil
}

func (cmd *buildCmd) newBuildPlan(
	buildContext *context.BuildContext, imageName image.Name,
	replicas []image.Name) (*builder.BuildPlan, error) {

	// Read in and parse dockerfile.
	dockerfile, err := cmd.getDockerfile(buildContext.ContextDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get dockerfile: %s", err)
	}

	// Remove image manifest if an image with the same name already exists.
	if err := cleanManifest(buildContext, imageName); err != nil {
		return nil, fmt.Errorf("failed to clean manifest: %s", err)
	}
	for _, replica := range replicas {
		if err := cleanManifest(buildContext, replica); err != nil {
			return nil, fmt.Errorf("failed to clean manifest: %s", err)
		}
	}

	// Init cache manager.
	cacheMgr := cmd.newCacheManager(buildContext, imageName)

	// forceCommit will make every step attempt to commit a layer.
	// Commit is noop for steps other than ADD/COPY/RUN if they are not after an
	// uncommitted RUN, so this won't generate extra empty layers.
	forceCommit := cmd.commit == "implicit"

	// Create BuildPlan and validate it.
	return builder.NewBuildPlan(
		buildContext, imageName, replicas, cacheMgr, dockerfile, cmd.allowModifyFS, forceCommit)
}

// Build image from the specified dockerfile.
// If --push is specified, will also push the image to those registries.
// If --load is specified, will load the image into the local docker daemon.
func (cmd *buildCmd) Build(contextDir string) error {
	log.Infof("Starting Makisu build (version=%s)", utils.BuildHash)

	// Create BuildContext.
	contextDirAbs, err := filepath.Abs(contextDir)
	if err != nil {
		return fmt.Errorf("failed to resolve context dir: %s", err)
	}
	if contextDirAbs == "/" {
		return fmt.Errorf("the absolute path for context directory %s is /. Cannot use root as context", contextDir)
	}
	imageStore, err := storage.NewImageStore(cmd.storageDir)
	if err != nil {
		return fmt.Errorf("failed to init image store: %s", err)
	}
	buildContext, err := context.NewBuildContext("/", contextDirAbs, imageStore)
	if err != nil {
		return fmt.Errorf("failed to create initial build context: %s", err)
	}
	defer buildContext.Cleanup()

	// Make sure sandbox is cleaned after build.
	// Optionally remove everything before and after build.
	defer storage.CleanupSandbox(cmd.storageDir)
	if cmd.allowModifyFS {
		if cmd.preserveRoot {
			rootPreserver, err := storage.NewRootPreserver("/", cmd.storageDir, pathutils.DefaultBlacklist)
			if err != nil {
				return fmt.Errorf("failed to preserve root: %s", err)
			}
			defer rootPreserver.RestoreRoot()
		}
		log.Debugf("build.Cmd.Build() first call")
		buildContext.MemFS.Remove()
		defer buildContext.MemFS.Remove()
	}

	// Create and execute build plan.
	imageName, err := cmd.getTargetImageName()
	if err != nil {
		return fmt.Errorf("failed to get target image name: %s", err)
	}
	var parsedReplicas []image.Name
	for _, replica := range cmd.replicas {
		parsedReplicas = append(parsedReplicas, image.MustParseName(replica))
	}
	buildPlan, err := cmd.newBuildPlan(buildContext, imageName, parsedReplicas)
	if err != nil {
		return fmt.Errorf("failed to create build plan: %s", err)
	}
	if _, err = buildPlan.Execute(); err != nil {
		return fmt.Errorf("failed to execute build plan: %s", err)
	}
	log.Infof("Successfully built image %s", imageName.ShortName())

	// Push image to registries that were specified in the --push flag.
	for _, registry := range cmd.pushRegistries {
		target := imageName.WithRegistry(registry)
		if err := pushImage(buildContext, target); err != nil {
			return fmt.Errorf("failed to push image: %s", err)
		}
	}
	for _, replica := range cmd.replicas {
		target := image.MustParseName(replica)
		if err := pushImage(buildContext, target); err != nil {
			return fmt.Errorf("failed to push image: %s", err)
		}
	}

	// Optionally save image as a tar file.
	if cmd.destination != "" {
		if err := cmd.saveImage(buildContext, imageName); err != nil {
			return fmt.Errorf("failed to save image: %s", err)
		}
	}

	// Optionally load image to local docker daemon.
	if cmd.doLoad {
		if err := cmd.loadImage(buildContext, imageName); err != nil {
			return fmt.Errorf("failed to load image: %s", err)
		}
	}

	log.Infof("Finished building %s", imageName.ShortName())
	return nil
}
