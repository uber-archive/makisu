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

	"github.com/uber/makisu/lib/cache"
	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/parser/dockerfile"
)

// BuildPlan describes a list of named buildStages, that can copy files between
// one another.
type BuildPlan struct {
	baseCtx       *context.BuildContext
	contextDirs   map[string][]string
	target        image.Name
	cacheMgr      cache.Manager
	stages        []*buildStage
	allowModifyFS bool
	forceCommit   bool
}

// NewBuildPlan takes in contextDir, a target image and an ImageStore, and
// returns a new BuildPlan.
func NewBuildPlan(
	ctx *context.BuildContext, target image.Name, cacheMgr cache.Manager,
	parsedStages []*dockerfile.Stage, allowModifyFS, forceCommit bool) (*BuildPlan, error) {

	plan := &BuildPlan{
		baseCtx:       ctx,
		contextDirs:   make(map[string][]string),
		target:        target,
		cacheMgr:      cacheMgr,
		stages:        make([]*buildStage, len(parsedStages)),
		allowModifyFS: allowModifyFS,
		forceCommit:   forceCommit,
	}

	aliases := make(map[string]bool)
	digestPairs := make(map[string][]*image.DigestPair)
	for i, parsedStage := range parsedStages {
		// Check for stage alias collision if alias isn't empty.
		if parsedStage.From.Alias != "" {
			if _, ok := aliases[parsedStage.From.Alias]; ok {
				return nil, fmt.Errorf("duplicate stage alias: %s", parsedStage.From.Alias)
			}
			aliases[parsedStage.From.Alias] = true
		}

		// Add this stage to the plan.
		stage, err := newBuildStage(
			plan.baseCtx, parsedStage, digestPairs, plan.allowModifyFS, plan.forceCommit)
		if err != nil {
			return nil, fmt.Errorf("failed to convert parsed stage: %s", err)
		}
		plan.stages[i] = stage

		// Merge context dirs.
		for alias, dirs := range stage.contextDirs {
			if _, ok := aliases[alias]; !ok {
				return nil, fmt.Errorf("copy from nonexistent stage %s", alias)
			}
			if _, ok := plan.contextDirs[alias]; ok {
				plan.contextDirs[alias] = append(plan.contextDirs[alias], dirs...)
			} else {
				plan.contextDirs[alias] = dirs
			}
		}
	}

	return plan, nil
}

// Execute executes all build stages in order.
func (plan *BuildPlan) Execute() (*image.DistributionManifest, error) {
	// Execute pre-build procedures. Try to pull some reusable layers from the registry.
	// TODO: in parallel
	for _, stage := range plan.stages {
		stage.pullCacheLayers(plan.cacheMgr)
	}

	var currStage *buildStage
	for k := 0; k < len(plan.stages); k++ {
		currStage = plan.stages[k]
		log.Infof("* Stage %d/%d : %s", k+1, len(plan.stages), currStage.String())

		lastStage := k == len(plan.stages)-1
		_, copiedFrom := plan.contextDirs[currStage.alias]
		if err := currStage.build(
			plan.cacheMgr, lastStage, copiedFrom); err != nil {
			return nil, fmt.Errorf("build stage: %s", err)
		}

		if plan.allowModifyFS {
			if k < len(plan.stages)-1 {
				// Save context directories needed for cross-stage copy operations.
				newRoot := currStage.ctx.StageDir(currStage.alias)
				contextDirs := plan.contextDirs[currStage.alias]

				if err := currStage.ctx.MemFS.Checkpoint(newRoot, contextDirs); err != nil {
					return nil, fmt.Errorf("checkpoint memfs: %s", err)
				}
			}

			if err := currStage.ctx.MemFS.Remove(); err != nil {
				return nil, fmt.Errorf("remove memfs: %s", err)
			}
		}
	}

	// Wait for cache layers to be pushed.
	// This will make them available to other builds ongoing on different machines.
	if err := plan.cacheMgr.WaitForPush(); err != nil {
		log.Errorf("Failed to push cache: %s", err)
	}

	// Save image.
	repo, tag := plan.target.GetRepository(), plan.target.GetTag()
	manifest, err := currStage.saveImage(plan.baseCtx.ImageStore, repo, tag)
	if err != nil {
		return nil, fmt.Errorf("save context image: %s", err)
	}

	// Print out the image size.
	size := int64(0)
	for _, layer := range manifest.Layers {
		size += layer.Size
	}
	log.Infow(fmt.Sprintf("Computed total image size %d", size), "total_image_size", size)

	return manifest, nil
}
