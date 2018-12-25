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

package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/uber/makisu/lib/builder/step"
	"github.com/uber/makisu/lib/cache"
	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/parser/dockerfile"
	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/utils"
)

type buildStageOptions struct {
	// forceCommit will make every step attampt to commit a layer.
	// Commit() is noop for steps other than ADD/COPY/RUN if they are not after
	// an uncommitted RUN, so this won't generate extra empty layers.
	forceCommit   bool
	allowModifyFS bool
	requireOnDisk bool
}

// buildStage represents a sequence of steps to build intermediate layers or a final image.
type buildStage struct {
	ctx               *context.BuildContext
	alias             string
	copyFromDirs      map[string][]string
	nodes             []*buildNode
	lastImageConfig   *image.Config
	sharedDigestPairs image.DigestPairMap

	opts *buildStageOptions
}

// newBuildStage initializes a buildStage.
func newBuildStage(
	baseCtx *context.BuildContext, alias string, parsedStage *dockerfile.Stage,
	digestPairs image.DigestPairMap, planOpts *buildPlanOptions) (*buildStage, error) {

	// Create a new build context for the stage.
	ctx, err := context.NewBuildContext(
		baseCtx.RootDir, baseCtx.ContextDir, baseCtx.ImageStore)
	if err != nil {
		return nil, fmt.Errorf("create stage build context: %s", err)
	}

	// Create steps from parsed stage.
	steps, err := createDockerfileSteps(ctx, parsedStage, planOpts)
	if err != nil {
		return nil, fmt.Errorf("new dockerfile steps: %s", err)
	}

	return newBuildStageHelper(ctx, alias, steps, digestPairs, planOpts)
}

// newRemoteImageStage initializes a buildStage.
func newRemoteImageStage(
	baseCtx *context.BuildContext, alias string, digestPairs image.DigestPairMap,
	planOpts *buildPlanOptions) (*buildStage, error) {

	// Create a new build context for the stage.
	ctx, err := context.NewBuildContext(
		baseCtx.RootDir, baseCtx.ContextDir, baseCtx.ImageStore)
	if err != nil {
		return nil, fmt.Errorf("create stage build context: %s", err)
	}

	// Create from step.
	from, err := step.NewFromStep(alias, alias, alias)
	if err != nil {
		return nil, fmt.Errorf("new from step: %s", err)
	}
	steps := []step.BuildStep{from}

	// Set forceCommit to false.
	// TODO: currently, allowModifyFS has to be true so FROM can be executed and
	// checkpointed. Value of allowModifyFS is verified later, but maybe the
	// verification should happen here?
	opts := &buildPlanOptions{
		forceCommit:   false,
		allowModifyFS: planOpts.allowModifyFS,
	}

	return newBuildStageHelper(ctx, alias, steps, digestPairs, opts)
}

func newBuildStageHelper(
	ctx *context.BuildContext, alias string, steps []step.BuildStep,
	digestPairs image.DigestPairMap, planOpts *buildPlanOptions) (*buildStage, error) {

	// Convert each step to a build node.
	var requireOnDisk bool
	nodes := make([]*buildNode, 0)
	copyFromDirs := make(map[string][]string)
	for _, step := range steps {
		newNode := newBuildNode(ctx, step)
		nodes = append(nodes, newNode)

		// Add context dirs for cross-stage copy, if any.
		alias, dirs := step.ContextDirs()
		if len(dirs) > 0 {
			if _, ok := copyFromDirs[alias]; !ok {
				copyFromDirs[alias] = make([]string, 0)
			}
			copyFromDirs[alias] = append(copyFromDirs[alias], dirs...)
		}
		if step.RequireOnDisk() {
			requireOnDisk = true
		}
	}

	stage := &buildStage{
		ctx:               ctx,
		copyFromDirs:      copyFromDirs,
		alias:             alias,
		nodes:             nodes,
		sharedDigestPairs: digestPairs,
		opts: &buildStageOptions{
			allowModifyFS: planOpts.allowModifyFS,
			forceCommit:   planOpts.forceCommit,
			requireOnDisk: requireOnDisk,
		},
	}

	return stage, nil
}

// createDockerfileSteps returns a list of build steps given a parsed stage.
func createDockerfileSteps(
	ctx *context.BuildContext, stage *dockerfile.Stage,
	planOpts *buildPlanOptions) ([]step.BuildStep, error) {

	checksum := crc32.ChecksumIEEE([]byte(utils.BuildHash + fmt.Sprintf("%v", *planOpts)))
	seed := fmt.Sprintf("%x", checksum)
	directives := append([]dockerfile.Directive{stage.From}, stage.Directives...)
	var steps []step.BuildStep
	for _, directive := range directives {
		step, err := step.NewDockerfileStep(ctx, directive, seed)
		if err != nil {
			return nil, fmt.Errorf("directive to build step: %s", err)
		}
		steps = append(steps, step)
		seed = step.CacheID()
	}
	return steps, nil
}

// build performs the build for that stage. There are side effects that should
// be expected on each node within the stage.
func (stage *buildStage) build(cacheMgr cache.Manager, lastStage, copiedFrom bool) error {
	// Reuse the digestpairs that other stages have populated.
	for _, node := range stage.nodes {
		if pairs, ok := stage.sharedDigestPairs[node.CacheID()]; ok {
			log.Infof("* Reusing digest pairs computed from earlier step %s", node.CacheID())
			node.digestPairs = pairs
		}
	}

	var err error
	diffIDs := make([]image.Digest, 0)
	histories := make([]image.History, 0)
	for i, node := range stage.nodes {
		// Build current step from the previous image config (possibly cached).
		modifyFS := stage.opts.requireOnDisk || copiedFrom
		if modifyFS && !stage.opts.allowModifyFS {
			return fmt.Errorf("fs not allowed to be modified")
		}
		skipBuild := i < stage.latestFetched() && i > 0
		lastStep := i == len(stage.nodes)-1
		forceCommit := i == 0 || (lastStage && lastStep) || stage.opts.forceCommit

		nodeOpts := &buildNodeOptions{
			skipBuild:   skipBuild,
			forceCommit: forceCommit,
			modifyFS:    modifyFS,
		}

		log.Infof("* Step %d/%d (%s) : %s", i+1, len(stage.nodes), nodeOpts.String(), node.String())
		stage.lastImageConfig, err = node.Build(cacheMgr, stage.lastImageConfig, nodeOpts)
		if err != nil {
			return fmt.Errorf("build node: %s", err)
		}

		// Update diff IDs and history information.
		for _, digestPair := range node.digestPairs {
			diffIDs = append(diffIDs, digestPair.TarDigest)
			histories = append(histories, image.History{
				Created:   time.Now(),
				CreatedBy: fmt.Sprintf("makisu: %s", node.String()),
				Author:    "makisu",
			})
		}

		// Update the shared map of cacheID to digest pair.
		if len(node.digestPairs) != 0 {
			stage.sharedDigestPairs[node.CacheID()] = node.digestPairs
		}
	}
	stage.lastImageConfig.Created = time.Now()
	stage.lastImageConfig.History = histories
	stage.lastImageConfig.RootFS.DiffIDs = diffIDs
	stage.lastImageConfig.ContainerConfiguration = nil
	return nil
}

// GetDistributionManifest returns the distribution manifest produced at the end of the stage.
func (stage *buildStage) GetDistributionManifest(store storage.ImageStore) (*image.DistributionManifest, error) {
	imageConfigJSON, err := json.Marshal(stage.lastImageConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal image config: %s", err)
	}
	imageConfigDigester := sha256.New()
	imageConfigDigester.Write(imageConfigJSON)
	imageConfigSHA256 := hex.EncodeToString(imageConfigDigester.Sum(nil))

	imageConfigPath := path.Join(stage.ctx.ImageStore.SandboxDir, imageConfigSHA256)
	if err := ioutil.WriteFile(imageConfigPath, imageConfigJSON, 0755); err != nil {
		return nil, fmt.Errorf("write image config: %s", err)
	}
	if err := store.Layers.LinkStoreFileFrom(imageConfigSHA256, imageConfigPath); err != nil {
		return nil, fmt.Errorf("commit image config to store: %s", err)
	}
	imageConfigStat, err := store.Layers.GetStoreFileStat(imageConfigSHA256)
	if err != nil {
		return nil, fmt.Errorf("get image config file stat: %s", err)
	}

	// Save the manifest at the last node to a temp file, then move into store.
	distributionManfest := image.DistributionManifest{
		SchemaVersion: 2,
		MediaType:     image.MediaTypeManifest,
	}

	distributionManfest.Config = image.Descriptor{
		MediaType: image.MediaTypeConfig,
		Size:      imageConfigStat.Size(),
		Digest:    image.Digest("sha256:" + imageConfigSHA256),
	}

	descriptors := []image.Descriptor{}
	for _, node := range stage.nodes {
		for _, digestPair := range node.digestPairs {
			descriptors = append(descriptors, digestPair.GzipDescriptor)
		}
	}

	distributionManfest.Layers = descriptors
	return &distributionManfest, nil
}

// saveImage saves the image produced at the end of this stage.
func (stage *buildStage) saveImage(store storage.ImageStore, repo, tag string) (*image.DistributionManifest, error) {
	manifest, err := stage.GetDistributionManifest(store)
	if err != nil {
		return nil, fmt.Errorf("get distribution manifest: %s", err)
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %s", err)
	}
	manifestFile, err := ioutil.TempFile(stage.ctx.ImageStore.SandboxDir, "")
	if err != nil {
		return nil, fmt.Errorf("tmp manifest file: %s", err)
	}

	manifestPath := manifestFile.Name()
	// Remove temp file after hard-linked to manifest store
	defer os.Remove(manifestPath)

	if err := ioutil.WriteFile(manifestPath, manifestJSON, 0755); err != nil {
		return nil, fmt.Errorf("write manifest file: %s", err)
	}
	if err := store.Manifests.LinkStoreFileFrom(repo, tag, manifestPath); err != nil {
		return nil, fmt.Errorf("commit manifest to store: %s", err)
	}
	return manifest, nil
}

// pullCacheLayers attempts to pull reusable layers from the distributed cache. Terminates once
// a node that can be cached fails to pull its layer.
func (stage *buildStage) pullCacheLayers(cacheMgr cache.Manager) {
	// Skip the first node since it's a FROM step. We do not want to try
	// to pull from cache because the step itself will pull the right layers when
	// it gets executed.
	for _, node := range stage.nodes[1:] {
		// Stop once the cache chain is broken.
		if node.HasCommit() || stage.opts.forceCommit {
			if !node.pullCacheLayer(cacheMgr) {
				return
			}
		}
	}
}

func (stage *buildStage) latestFetched() int {
	latest := -1
	for i, node := range stage.nodes[1:] {
		// Stop once the cache chain is broken.
		if node.HasCommit() {
			if len(node.digestPairs) != 0 {
				latest = i + 1
			} else {
				return latest
			}
		}
	}
	return latest
}

// String returns the string representation of this stage. This may be useful in debugging issues.
func (stage *buildStage) String() string {
	return fmt.Sprintf("(alias=%v,latestfetched=%v)", stage.alias, stage.latestFetched())
}

// checkpoint copies over the cross stage referenced files and directories to the cross ref root
// location inside a blacklisted directory. Those files will be copied back onto the real root
// of the fs once the step that references them gets executed.
func (stage *buildStage) checkpoint(copyFromDirs []string) error {
	newRoot := stage.ctx.CopyFromRoot(stage.alias)
	return stage.ctx.MemFS.Checkpoint(newRoot, copyFromDirs)
}

func (stage *buildStage) cleanup() error { return stage.ctx.MemFS.Remove() }
