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

package step

import (
	"fmt"
	"hash/crc32"
	"os"
	"strconv"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
)

// baseStep is the struct that will be embedded in all kinds of steps.
type baseStep struct {
	directive  Directive
	args       string
	workingDir string
	cacheID    string
	commit     bool
}

// newBaseStep returns a new baseStep. baseStep is not sufficient to implement
// BuildStep, but should instead be imbeded in specific steps, as it implements
// many of the functions that each BuildStep needs.
func newBaseStep(directive Directive, args string, commit bool) *baseStep {
	return &baseStep{directive: directive, args: args, commit: commit}
}

// ContextDirs returns directories that this step requires from another stage.
func (s *baseStep) ContextDirs() (string, []string) { return "", nil }

func (s *baseStep) RequireOnDisk() bool { return false }

// CacheID returns the cache ID of the step.
func (s *baseStep) CacheID() string { return s.cacheID }

// String returns the string representation of this step.
func (s *baseStep) String() string {
	commitStr := ""
	if s.commit {
		commitStr = "#!COMMIT"
	}
	return fmt.Sprintf("%s %s %s (%s)", s.directive, s.args, commitStr, s.cacheID)
}

// SetCacheID sets the cache ID of the step given a seed SHA256 value.
// Special steps like FROM, ADD, COPY have their own implementations.
func (s *baseStep) SetCacheID(ctx *context.BuildContext, seed string) error {
	commitStr := fmt.Sprintf("%v", s.commit)
	checksum := crc32.ChecksumIEEE([]byte(seed + string(s.directive) + s.args + commitStr))
	s.cacheID = fmt.Sprintf("%x", checksum)
	return nil
}

// SetWorkingDir set the working dir of the current step
// Exporting the logic to this method allows for an easier `ApplyCtxAndConfig` overwriting
func (s *baseStep) SetWorkingDir(
	ctx *context.BuildContext, imageConfig *image.Config) error {
	s.workingDir = ctx.RootDir // Default workingDir to root.

	// Set working dir from imageConfig
	if imageConfig != nil && imageConfig.Config.WorkingDir != "" {
		s.workingDir = os.ExpandEnv(imageConfig.Config.WorkingDir)
	}

	// Create working dir if it does not exist.
	if _, err := os.Lstat(s.workingDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(s.workingDir, 0755); err != nil {
				return fmt.Errorf("mkdir all working dir %s: %s", s.workingDir, err)
			}
		} else {
			return fmt.Errorf("lstat working dir %s: %s", s.workingDir, err)
		}
	}
	return nil
}

// SetEnvFromContext set environment variables from previous stages
// Exporting the logic to this method allows for an easier `ApplyCtxAndConfig` overwriting
func (s *baseStep) SetEnvFromContext(
	ctx *context.BuildContext) error {
	for key, value := range ctx.StageVars {
		unquoted, err := strconv.Unquote(value)
		if err == nil {
			value = unquoted
		}
		value = os.ExpandEnv(value)
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set env %s=%s: %s", key, value, err)
		}
	}
	return nil
}

// ApplyCtxAndConfig sets up the execution environment from build context and
// image config.
// This function will not be skipped.
func (s *baseStep) ApplyCtxAndConfig(
	ctx *context.BuildContext, imageConfig *image.Config) error {
	s.SetWorkingDir(ctx, imageConfig)
	s.SetEnvFromContext(ctx)
	return nil
}

// Execute executes the step. If modifyFS is true, the command might change the
// local file system.
// Default implementation is noop.
func (s *baseStep) Execute(ctx *context.BuildContext, modifyFS bool) error {
	return nil
}

// Commit generates an image layer.
func (s *baseStep) Commit(ctx *context.BuildContext) ([]*image.DigestPair, error) {
	return commitLayer(ctx)
}

// UpdateCtxAndConfig updates mutable states in build context, and generates a
// new image config base on config from previous step.
// Default implementation makes a copy of given image config.
func (s *baseStep) UpdateCtxAndConfig(
	ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {

	return image.NewImageConfigFromCopy(imageConfig)
}

// HasCommit returns whether or not a particular commit step has a commit
// annotation.
func (s *baseStep) HasCommit() bool { return s.commit }
