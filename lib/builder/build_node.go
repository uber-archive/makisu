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
	"archive/tar"
	"fmt"
	"strings"
	"time"

	"github.com/uber/makisu/lib/builder/step"
	"github.com/uber/makisu/lib/cache"
	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/tario"
)

// buildOptions wraps options that are specified when a node is built.
type buildOptions struct {
	modifyFS    bool // If true, the node will modify the file system.
	skipBuild   bool // If true, the node will not call build on its build step.
	forceCommit bool // If true, the node will always commit a layer if it can.
}

// buildNode corresponds to a single BuildStep and its metadata.
type buildNode struct {
	step.BuildStep

	// ctx is the build context that this node should operate on. It may be
	// shared with other nodes, requiring that the nodes be executed in order.
	ctx *context.BuildContext

	// digestPair are the layer(s) committed or fetched by this node.
	digestPairs []*image.DigestPair
}

// newBuildNode initalizes a buildNode.
func newBuildNode(ctx *context.BuildContext, step step.BuildStep) *buildNode {
	return &buildNode{
		BuildStep: step,
		ctx:       ctx,
	}
}

// Build applies the image config, builds the step unless it should be skipped or was cached, and
// generates a resulting config for the next step. Also pushes cache layers if this step commits
// a layer.
// TODO: Build and push intermediate cache layers concurrently.
func (n *buildNode) Build(
	cacheMgr cache.Manager, prevConfig *image.Config, opts *buildOptions) (*image.Config, error) {

	// Always apply config.
	if err := n.ApplyConfig(n.ctx, prevConfig); err != nil {
		return nil, fmt.Errorf("apply config: %s", err)
	}

	cached := n.digestPairs != nil
	if cached {
		// The step was cached.
		// Update MemFS, and only untar layers if modifyFS is strue.
		for _, pair := range n.digestPairs {
			if err := n.applyLayer(pair, opts.modifyFS); err != nil {
				return nil, fmt.Errorf("apply cache: %s", err)
			}
		}
	}

	if opts.skipBuild {
		log.Infof("* Skipping execution; a later step was cached *")
	} else if cached {
		log.Infof("* Skipping execution; cache was applied *")
	} else if err := n.doExecute(cacheMgr, opts); err != nil {
		return nil, fmt.Errorf("do execute: %s", err)
	} else if !n.HasCommit() && !opts.forceCommit {
		log.Infof("* Not committing step %s", n.String())
	} else if err := n.doCommit(cacheMgr, opts); err != nil {
		return nil, fmt.Errorf("do commit: %s", err)
	}

	// Always generate a new config.
	config, err := n.GenerateConfig(n.ctx, prevConfig)
	if err != nil {
		return nil, fmt.Errorf("generate config: %s", err)
	}
	return config, nil
}

func (n *buildNode) doCommit(cacheMgr cache.Manager, opts *buildOptions) error {
	var err error
	n.digestPairs, err = n.Commit(n.ctx)
	if err != nil {
		return fmt.Errorf("commit: %s", err)
	}

	// If the number of digestPairs is 0 or greater than 1 then we cannot push
	// the resulting layer mappings to the distributed cache.
	if len(n.digestPairs) != 1 {
		return nil
	}

	if err := n.pushCacheLayers(cacheMgr); err != nil {
		return fmt.Errorf("push cache: %s", err)
	}
	return nil
}

func (n *buildNode) doExecute(cacheMgr cache.Manager, opts *buildOptions) error {
	start := time.Now()
	err := n.Execute(n.ctx, opts.modifyFS)
	if err != nil {
		return fmt.Errorf("execute step: %s", err)
	}
	log.Infof("* Execute %s took %v", n.String(), time.Since(start))
	return nil
}

// applyLayer applies the layer to the current memFS.
// If modifyfs is true, writes it to the local file system.
func (n *buildNode) applyLayer(digestPair *image.DigestPair, modifyfs bool) error {
	reader, err := n.ctx.ImageStore.Layers.GetStoreFileReader(digestPair.GzipDescriptor.Digest.Hex())
	if err != nil {
		return fmt.Errorf("get reader from layer: %s", err)
	}
	gzipReader, err := tario.NewGzipReader(reader)
	if err != nil {
		return fmt.Errorf("create gzip reader for layer: %s", err)
	}
	log.Infof("* Applying cache layer %s (unpack=%v)",
		digestPair.GzipDescriptor.Digest.Hex(), modifyfs)
	if err := n.ctx.MemFS.UpdateFromTarReader(tar.NewReader(gzipReader), modifyfs); err != nil {
		return fmt.Errorf("untar reader: %s", err)
	}
	return nil
}

// pushCacheLayers pushs cached layers for this node's digest pair(s).
func (n *buildNode) pushCacheLayers(cacheMgr cache.Manager) error {
	digestPair := n.digestPairs[0]
	log.Infof("* Committed gzipped layer %s (%d bytes), pushing with cache ID %s *",
		digestPair.GzipDescriptor.Digest, digestPair.GzipDescriptor.Size, n.CacheID())
	return cacheMgr.PushCache(n.CacheID(), digestPair)
}

// pullCacheLayers pulls cached layers for this node's digest pair(s).
func (n *buildNode) pullCacheLayers(cacheMgr cache.Manager) bool {
	digestPair, err := cacheMgr.PullCache(n.CacheID())
	if err != nil {
		log.Errorf("Failed to fetch intermediate layer with cache ID %s: %s", n.CacheID(), err)
		return false
	}
	n.digestPairs = []*image.DigestPair{digestPair}
	return true
}

func (opts *buildOptions) String() string {
	s := []string{}
	if opts.skipBuild {
		s = append(s, "skip")
	}
	if opts.forceCommit {
		s = append(s, "commit")
	}
	if opts.modifyFS {
		s = append(s, "modifyfs")
	}
	if len(s) == 0 {
		return ""
	}
	return strings.Join(s, ",")
}
