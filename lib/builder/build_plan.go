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
	"strconv"

	"github.com/uber/makisu/lib/cache"
	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/parser/dockerfile"
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
	stages       []*buildStage
	stageAliases map[string]struct{}

	opts *buildPlanOptions
}

// NewBuildPlan takes in contextDir, a target image and an ImageStore, and
// returns a new BuildPlan.
func NewBuildPlan(
	ctx *context.BuildContext, target image.Name, replicas []image.Name, cacheMgr cache.Manager,
	parsedStages []*dockerfile.Stage, allowModifyFS, forceCommit bool) (*BuildPlan, error) {

	aliases, err := buildAliases(parsedStages)
	if err != nil {
		return nil, fmt.Errorf("build alias list: %s", err)
	}

	plan := &BuildPlan{
		baseCtx:      ctx,
		copyFromDirs: make(map[string][]string),
		target:       target,
		replicas:     replicas,
		cacheMgr:     cacheMgr,
		stages:       make([]*buildStage, len(parsedStages)),
		stageAliases: aliases,
		opts: &buildPlanOptions{
			forceCommit:   forceCommit,
			allowModifyFS: allowModifyFS,
		},
	}

	for i, parsedStage := range parsedStages {
		// Add this stage to the plan.
		stage, err := newBuildStage(
			ctx, parsedStage.From.Alias, parsedStage, plan.opts)
		if err != nil {
			return nil, fmt.Errorf("failed to convert parsed stage: %s", err)
		}

		if len(stage.copyFromDirs) > 0 && !plan.opts.allowModifyFS {
			// TODO(pourchet): Support this at some point.
			// TODO: have a centralized place for these validations.
			return nil, fmt.Errorf("must allow modifyfs for multi-stage dockerfiles with COPY --from")
		}
		plan.stages[i] = stage
	}

	if err := plan.handleCopyFromDirs(); err != nil {
		return nil, fmt.Errorf("handle cross refs: %s", err)
	}

	return plan, nil
}

// handleCopyFromDirs goes through all of the stages in the build plan and looks
// at the `COPY --from` steps to make sure they are valid.
func (plan *BuildPlan) handleCopyFromDirs() error {
	for _, stage := range plan.stages {
		for alias, dirs := range stage.copyFromDirs {
			plan.copyFromDirs[alias] = stringset.FromSlice(
				append(plan.copyFromDirs[alias], dirs...),
			).ToSlice()
		}
	}
	return nil
}

// buildAliases mutates the list of stages to assign default aliases.
// Default aliases will be integers starting from 0.
func buildAliases(stages dockerfile.Stages) (map[string]struct{}, error) {
	aliases := make(map[string]struct{})
	for i, parsedStage := range stages {
		// Check for stage alias collision if alias isn't empty.
		if parsedStage.From.Alias != "" {
			if _, ok := aliases[parsedStage.From.Alias]; ok {
				return nil, fmt.Errorf("duplicate stage alias: %s", parsedStage.From.Alias)
			} else if _, err := strconv.Atoi(parsedStage.From.Alias); err == nil {
				// Docker would return `name can't start with a number or contain symbols`.
				return nil, fmt.Errorf("stage alias cannot be a number: %s", parsedStage.From.Alias)
			}
		} else {
			parsedStage.From.Alias = strconv.Itoa(i)
		}
		aliases[parsedStage.From.Alias] = struct{}{}
	}
	return aliases, nil
}

// Execute executes all build stages in order.
func (plan *BuildPlan) Execute() (*image.DistributionManifest, error) {
	var currStage *buildStage
	for k := 0; k < len(plan.stages); k++ {
		currStage = plan.stages[k]
		log.Infof("* Stage %d/%d : %s", k+1, len(plan.stages), currStage.String())

		// Try to pull reusable layers cached from previous builds.
		currStage.pullCacheLayers(plan.cacheMgr)

		lastStage := k == len(plan.stages)-1
		_, copiedFrom := plan.copyFromDirs[currStage.alias]

		if err := plan.executeStage(currStage, lastStage, copiedFrom); err != nil {
			return nil, fmt.Errorf("execute stage: %s", err)
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
	// Handle `COPY --from=<alias>` where alias is not a stage but an image.
	// Create and execute a fake stage with only FROM. This case will not work
	// if modifyfs is set to false, but that combination was rejected in
	// NewPlan().
	// TODO: This should be done at step level.
	for alias, _ := range stage.copyFromDirs {
		if _, ok := plan.stageAliases[alias]; !ok {
			name, err := image.ParseNameForPull(alias)
			if err != nil || !name.IsValid() {
				return fmt.Errorf("copy from nonexistent stage %s", alias)
			}
			remoteImageStage, err := newRemoteImageStage(
				plan.baseCtx, alias, plan.opts)
			if err != nil {
				return fmt.Errorf("new image stage: %s", err)
			}

			if err := remoteImageStage.build(plan.cacheMgr, lastStage, copiedFrom); err != nil {
				return fmt.Errorf("build stage %s: %s", stage.alias, err)
			}

			if err := stage.checkpoint(plan.copyFromDirs[alias]); err != nil {
				return fmt.Errorf("checkpoint stage %s: %s", stage.alias, err)
			}

			if err := stage.cleanup(); err != nil {
				return fmt.Errorf("cleanup stage %s: %s", stage.alias, err)
			}
		}
	}

	if err := stage.build(plan.cacheMgr, lastStage, copiedFrom); err != nil {
		return fmt.Errorf("build stage %s: %s", stage.alias, err)
	}

	// Note: returning before checkpoint() would create problems for
	// `COPY --from`. That combination was rejected in NewPlan().
	if !plan.opts.allowModifyFS {
		return nil
	}

	if err := stage.checkpoint(plan.copyFromDirs[stage.alias]); err != nil {
		return fmt.Errorf("checkpoint stage %s: %s", stage.alias, err)
	}

	if err := stage.cleanup(); err != nil {
		return fmt.Errorf("cleanup stage %s: %s", stage.alias, err)
	}

	return nil
}
