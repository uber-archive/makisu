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
	"fmt"
	"hash/crc32"
	"os"
	"strconv"

	"github.com/uber/makisu/lib/cache"
	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/parser/dockerfile"
	"github.com/uber/makisu/lib/utils"
	"github.com/uber/makisu/lib/utils/stringset"
)

type buildPlanOptions struct {
	forceCommit   bool
	allowModifyFS bool
}

// BuildPlan describes a list of named buildStages, that can copy files between
// one another.
type BuildPlan struct {
	baseCtx      *context.BuildContext
	copyFromDirs map[string][]string
	target       image.Name
	replicas     []image.Name
	cacheMgr     cache.Manager

	// stages contains the build stages defined in dockerfile.
	stages []*buildStage
	// Which stage is the target for this plan
	stageTarget string

	// TODO: this is not used for now.
	// Aliases of stages.
	stageAliases map[string]struct{}

	// TODO: this is not used for now.
	// Index aliases of stages.
	// This extra index is needed because shadow stages could be inserted into
	// stages list to support `COPY --from=<image>`.
	stageIndexAliases map[string]*buildStage

	opts *buildPlanOptions
}

// NewBuildPlan takes in contextDir, a target image and an ImageStore, and
// returns a new BuildPlan.
func NewBuildPlan(
	ctx *context.BuildContext, target image.Name, replicas []image.Name, cacheMgr cache.Manager,
	parsedStages []*dockerfile.Stage, allowModifyFS, forceCommit bool, stageTarget string) (*BuildPlan, error) {

	plan := &BuildPlan{
		baseCtx:           ctx,
		copyFromDirs:      make(map[string][]string),
		target:            target,
		replicas:          replicas,
		cacheMgr:          cacheMgr,
		stages:            make([]*buildStage, 0),
		stageTarget:       stageTarget,
		stageAliases:      make(map[string]struct{}),
		stageIndexAliases: make(map[string]*buildStage),
		opts: &buildPlanOptions{
			forceCommit:   forceCommit,
			allowModifyFS: allowModifyFS,
		},
	}

	if err := plan.processStagesAndAliases(ctx, parsedStages); err != nil {
		return nil, fmt.Errorf("process stages and aliases: %s", err)
	}

	return plan, nil
}

func (plan *BuildPlan) processStagesAndAliases(
	ctx *context.BuildContext, parsedStages dockerfile.Stages) error {

	checksum := crc32.ChecksumIEEE([]byte(utils.BuildHash + fmt.Sprintf("%v", plan.opts)))
	seedCacheID := fmt.Sprintf("%x", checksum)

	existingAliases := make(map[string]struct{})
	for i, parsedStage := range parsedStages {
		// Record alias.
		if parsedStage.From.Alias != "" {
			if _, ok := existingAliases[parsedStage.From.Alias]; ok {
				return fmt.Errorf("duplicate stage alias: %s", parsedStage.From.Alias)
			} else if _, err := strconv.Atoi(parsedStage.From.Alias); err == nil {
				// Note: Docker would return `name can't start with a number or
				// contain symbols`.
				return fmt.Errorf("stage alias cannot be a number: %s", parsedStage.From.Alias)
			}
		} else {
			parsedStage.From.Alias = strconv.Itoa(i)
		}
		existingAliases[parsedStage.From.Alias] = struct{}{}

		// Add this stage to the plan.
		stage, err := newBuildStage(
			ctx, parsedStage.From.Alias, seedCacheID, parsedStage, plan.opts)
		if err != nil {
			return fmt.Errorf("failed to convert parsed stage: %s", err)
		}

		if len(stage.copyFromDirs) > 0 && !plan.opts.allowModifyFS {
			// TODO(pourchet): Support this at some point.
			// TODO: have a centralized place for these validations.
			return fmt.Errorf("must allow modifyfs for multi-stage dockerfiles with COPY --from")
		}

		// Goes through all of the stages in the build plan and looks
		// at the `COPY --from` steps to make sure they are valid.
		for alias, dirs := range stage.copyFromDirs {
			// Populate copyFromDirs.
			plan.copyFromDirs[alias] = stringset.FromSlice(
				append(plan.copyFromDirs[alias], dirs...),
			).ToSlice()

			if _, ok := existingAliases[alias]; !ok {
				// If the alias was an image name and not already handled,
				// prepend a fake stage with the alias to download that image.
				if name, err := image.ParseNameForPull(alias); err != nil || !name.IsValid() {
					return fmt.Errorf("copy from nonexistent stage %s", alias)
				}
				remoteImageStage, err := newRemoteImageStage(
					plan.baseCtx, alias, seedCacheID, plan.opts)
				if err != nil {
					return fmt.Errorf("new image stage: %s", err)
				}

				// Append to stage list and update cache id.
				plan.stages = append(plan.stages, remoteImageStage)
				// TODO: instead of chaining cache ID, it's better to calculate
				// from scratch before executing a stage.
				seedCacheID = remoteImageStage.nodes[len(remoteImageStage.nodes)-1].CacheID()
			}
		}

		// Append to stage list and update cache id.
		plan.stages = append(plan.stages, stage)
		// TODO: instead of chaining cache ID, it's better to calculate from
		// scratch before executing a stage.
		seedCacheID = stage.nodes[len(stage.nodes)-1].CacheID()
	}
	plan.stageAliases = existingAliases

	if plan.stageTarget != "" {
		if _, ok := plan.stageAliases[plan.stageTarget]; !ok {
			return fmt.Errorf("target stage not found in dockerfile %s", plan.stageTarget)
		}
	}

	return nil
}

// Execute executes all build stages in order.
func (plan *BuildPlan) Execute() (*image.DistributionManifest, error) {
	// We need to backup the original env to restore it between stages
	orignalEnv := utils.ConvertStringSliceToMap(os.Environ())

	var currStage *buildStage
	for k := 0; k < len(plan.stages); k++ {
		currStage = plan.stages[k]

		// TODO: Implicit stages from "COPY --from=<image>" might introduce
		// confusion here. Print stageIndexAliases instead.
		log.Infof("* Stage %d/%d : %s", k+1, len(plan.stages), currStage.String())

		// Try to pull reusable layers cached from previous builds.
		currStage.pullCacheLayers(plan.cacheMgr)

		lastStage := k == len(plan.stages)-1
		_, copiedFrom := plan.copyFromDirs[currStage.alias]

		if err := plan.executeStage(currStage, lastStage, copiedFrom); err != nil {
			return nil, fmt.Errorf("execute stage: %s", err)
		}

		// Restore env
		os.Clearenv()
		for k, v := range orignalEnv {
			os.Setenv(k, v)
		}

		if plan.stageTarget != "" && currStage.alias == plan.stageTarget {
			log.Info("Finished building target stage")
			break
		}
	}

	// Wait for cache layers to be pushed. This will make them available to
	// other builds ongoing on different machines.
	if err := plan.cacheMgr.WaitForPush(); err != nil {
		log.Errorf("Failed to push cache: %s", err)
	}

	// Save image manifest.
	manifest, err := currStage.saveManifest(plan.baseCtx.ImageStore, plan.target)
	if err != nil {
		return nil, fmt.Errorf("save image manifest %s: %s", plan.target, err)
	}
	for _, replica := range plan.replicas {
		_, err := currStage.saveManifest(plan.baseCtx.ImageStore, replica)
		if err != nil {
			return nil, fmt.Errorf("save alias manifest %s: %s", replica, err)
		}
	}

	// Print out the image size.
	size := int64(0)
	for _, layer := range manifest.Layers {
		size += layer.Size
	}
	log.Infow(fmt.Sprintf("Computed total image size %d", size), "total_image_size", size)

	return manifest, nil
}

func (plan *BuildPlan) executeStage(stage *buildStage, lastStage, copiedFrom bool) error {
	if err := stage.build(plan.cacheMgr, lastStage, copiedFrom); err != nil {
		return fmt.Errorf("build stage %s: %s", stage.alias, err)
	}

	if plan.opts.allowModifyFS {
		// Note: The rest of this function mostly deal with `COPY --from`
		// related logic, and currently `COPY --from` cannot be supported with
		// modifyfs=false. That combination was rejected in NewPlan().
		if err := stage.checkpoint(plan.copyFromDirs[stage.alias]); err != nil {
			return fmt.Errorf("checkpoint stage %s: %s", stage.alias, err)
		}

		if err := stage.cleanup(); err != nil {
			return fmt.Errorf("cleanup stage %s: %s", stage.alias, err)
		}
	}

	return nil
}
