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

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.PersistentFlags().StringVarP(&DockerfilePath, "file", "f", "Dockerfile", "The absolute path to the dockerfile")
	buildCmd.PersistentFlags().StringVarP(&Tag, "tag", "t", "", "Image tag (required)")

	buildCmd.PersistentFlags().StringArrayVar(&PushRegistries, "push", nil, "Registry to push image to")
	buildCmd.PersistentFlags().StringVar(&RegistryConfig, "registry-config", "", "Set build-time variables")
	buildCmd.PersistentFlags().StringVar(&Destination, "dest", "", "Destination of the image tar")

	buildCmd.PersistentFlags().StringArrayVar(&BuildArgs, "build-arg", nil, "Argument to the dockerfile as per the spec of ARG. Format is \"--build-arg <arg>=<value>\"")
	buildCmd.PersistentFlags().BoolVar(&AllowModifyFS, "modifyfs", false, "Allow makisu to modify files outside of its internal storage dir")
	buildCmd.PersistentFlags().StringVar(&Commit, "commit", "implicit", "Set to explicit to only commit at steps with '#!COMMIT' annotations; Set to implicit to commit at every ADD/COPY/RUN step")
	buildCmd.PersistentFlags().StringArrayVar(&Blacklists, "blacklist", nil, "Makisu will ignore all changes to these locations in the resulting docker images")

	buildCmd.PersistentFlags().DurationVar(&LocalCacheTTL, "local-cache-ttl", time.Hour*168, "Time-To-Live for local cache")
	buildCmd.PersistentFlags().StringVar(&RedisCacheAddress, "redis-cache-addr", "", "The address of a redis server for cacheID to layer sha mapping")
	buildCmd.PersistentFlags().DurationVar(&RedisCacheTTL, "redis-cache-ttl", time.Hour*168, "Time-To-Live for redis cache")
	buildCmd.PersistentFlags().StringVar(&HTTPCacheAddress, "http-cache-addr", "", "The address of the http server for cacheID to layer sha mapping")
	buildCmd.PersistentFlags().StringArrayVar(&HTTPCacheHeaders, "http-cache-header", nil, "Request header for http cache server. Format is \"--http-cache-header <header>:<value>\"")

	buildCmd.PersistentFlags().StringVar(&DockerHost, "docker-host", utils.DefaultEnv("DOCKER_HOST", "unix:///var/run/docker.sock"), "Docker host to load images to")
	buildCmd.PersistentFlags().StringVar(&DockerVersion, "docker-version", utils.DefaultEnv("DOCKER_VERSION", "1.21"), "Version string for loading images to docker")
	buildCmd.PersistentFlags().StringVar(&DockerScheme, "docker-scheme", utils.DefaultEnv("DOCKER_SCHEME", "http"), "Scheme for api calls to docker daemon")
	buildCmd.PersistentFlags().BoolVar(&DoLoad, "load", false, "Load image into docker daemon after build. Requires access to docker socket at location defined by ${DOCKER_HOST}")

	buildCmd.PersistentFlags().StringVar(&StorageDir, "storage", "", "Directory that makisu uses for temp files and cached layers. Mount this path for better caching performance. If modifyfs is set, default to /makisu-storage; Otherwise default to /tmp/makisu-storage")
	buildCmd.PersistentFlags().StringVar(&CompressionLevel, "compression", "default", "Image compression level, could be 'no', 'speed', 'size', 'default'")

	buildCmd.MarkFlagRequired("tag")
	buildCmd.Flags().SortFlags = false
	buildCmd.PersistentFlags().SortFlags = false
}

var (
	DockerfilePath string
	Tag            string

	PushRegistries []string
	RegistryConfig string
	Destination    string

	BuildArgs     []string
	AllowModifyFS bool
	Commit        string
	Blacklists    []string

	LocalCacheTTL     time.Duration
	RedisCacheAddress string
	RedisCacheTTL     time.Duration
	HTTPCacheAddress  string
	HTTPCacheHeaders  []string

	DockerHost    string
	DockerVersion string
	DockerScheme  string
	DoLoad        bool

	StorageDir       string
	CompressionLevel string

	buildCmd = &cobra.Command{
		Use: "build -t=<image_tag> [flags] <context_path>",
		DisableFlagsInUseLine: true,
		Short: "Build docker image, optionally push to registries and/or load into docker daemon",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("Requires build context as argument")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := Build(args[0]); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
)

func processFlags() error {
	if err := maybeBlacklistVarRun(); err != nil {
		return fmt.Errorf("failed to extend blacklist: %s", err)
	}

	if len(Blacklists) != 0 {
		newBlacklist := append(pathutils.DefaultBlacklist, Blacklists...)
		pathutils.DefaultBlacklist = stringset.FromSlice(newBlacklist).ToSlice()
		log.Infof("Added %d new items to blacklist: %v", len(Blacklists), Blacklists)
	}

	if err := tario.SetCompressionLevel(CompressionLevel); err != nil {
		return fmt.Errorf("set compression level: %s", err)
	}

	if Commit != "explicit" && Commit != "implicit" {
		return fmt.Errorf("invalid commit option: %s", Commit)
	}

	if err := initRegistryConfig(); err != nil {
		return fmt.Errorf("failed to initialize registry configuration: %s", err)
	}

	// If modifyfs is true, verify it's not runninng on Mac.
	if AllowModifyFS && runtime.GOOS == "darwin" {
		return fmt.Errorf("modifyfs option could erase fs and is not allowed on Mac")
	}

	// Configure default storage dir.
	if StorageDir == "" {
		if AllowModifyFS {
			StorageDir = pathutils.DefaultStorageDir
		} else {
			StorageDir = "/tmp/makisu-storage"
		}
	}

	// Verify storage dir is not child of internal dir.
	if pathutils.IsDescendantOfAny(StorageDir, []string{pathutils.DefaultInternalDir}) {
		return fmt.Errorf("storage dir cannot be under internal dir %s",
			pathutils.DefaultInternalDir)
	}
	return nil
}

func newBuildPlan(
	buildContext *context.BuildContext, imageName image.Name) (*builder.BuildPlan, error) {

	// Read in and parse dockerfile.
	dockerfile, err := getDockerfile(buildContext.ContextDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get dockerfile: %s", err)
	}

	// Remove image manifest if an image with the same name already exists.
	if err := cleanManifest(buildContext, imageName); err != nil {
		return nil, fmt.Errorf("failed to clean manifest: %s", err)
	}

	// Init cache manager.
	cacheMgr := newCacheManager(buildContext, imageName)

	// forceCommit will make every step attempt to commit a layer.
	// Commit is noop for steps other than ADD/COPY/RUN if they are not after an
	// uncommitted RUN, so this won't generate extra empty layers.
	forceCommit := Commit == "implicit"

	// Create BuildPlan and validate it.
	return builder.NewBuildPlan(buildContext, imageName, cacheMgr,
		dockerfile, AllowModifyFS, forceCommit)
}

// Build image from the specified dockerfile.
// If --push is specified, will also push the image to those registries.
// If --load is specified, will load the image into the local docker daemon.
func Build(contextDir string) error {
	log.Infof("Starting Makisu build (version=%s)", utils.BuildHash)

	if err := processFlags(); err != nil {
		return fmt.Errorf("failed to process flags: %s", err)
	}

	// Create BuildContext.
	contextDirAbs, err := filepath.Abs(contextDir)
	if err != nil {
		return fmt.Errorf("failed to resolve context dir: %s", err)
	}
	if contextDirAbs == "/" {
		return fmt.Errorf("the absolute path for context directory %s is /. Cannot use root as context", contextDir)
	}
	imageStore, err := storage.NewImageStore(StorageDir)
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
	defer storage.CleanupSandbox(StorageDir)
	if AllowModifyFS {
		buildContext.MemFS.Remove()
		defer buildContext.MemFS.Remove()
	}

	// Create and execute build plan.
	imageName, err := getTargetImageName()
	if err != nil {
		return fmt.Errorf("failed to get target image name: %s", err)
	}
	buildPlan, err := newBuildPlan(buildContext, imageName)
	if err != nil {
		return fmt.Errorf("failed to create build plan: %s", err)
	}
	if _, err = buildPlan.Execute(); err != nil {
		return fmt.Errorf("failed to execute build plan: %s", err)
	}
	log.Infof("Successfully built image %s", imageName.ShortName())

	// Push image to registries that were specified in the --push flag.
	for _, registry := range PushRegistries {
		target := imageName.WithRegistry(registry)
		if err := pushImage(buildContext, target); err != nil {
			return fmt.Errorf("failed to push image: %s", err)
		}
	}

	// Optionally save image as a tar file.
	if Destination != "" {
		if err := saveImage(buildContext, imageName); err != nil {
			return fmt.Errorf("failed to save image: %s", err)
		}
	}

	// Optionally load image to local docker daemon.
	if DoLoad {
		if err := loadImage(buildContext, imageName); err != nil {
			return fmt.Errorf("failed to load image: %s", err)
		}
	}

	log.Infof("Finished building %s", imageName.ShortName())
	return nil
}
