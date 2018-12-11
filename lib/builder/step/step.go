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

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/parser/dockerfile"
)

// Directive represents a valid directive type.
type Directive string

// Set of all valid directives.
const (
	Add         = Directive("ADD")
	Arg         = Directive("ARG")
	Cmd         = Directive("CMD")
	Copy        = Directive("COPY")
	Entrypoint  = Directive("ENTRYPOINT")
	Env         = Directive("ENV")
	Expose      = Directive("EXPOSE")
	From        = Directive("FROM")
	Healthcheck = Directive("HEALTHCHECK")
	Label       = Directive("LABEL")
	Maintainer  = Directive("MAINTAINER")
	Run         = Directive("RUN")
	Stopsignal  = Directive("STOPSIGNAL")
	User        = Directive("USER")
	Volume      = Directive("VOLUME")
	Workdir     = Directive("WORKDIR")
)

// BuildStep performs build for one build step.
type BuildStep interface {
	String() string

	// RequireOnDisk returns whether executing this step requires on-disk state.
	RequireOnDisk() bool

	// ContextDirs returns directories that this step requires from another stage.
	ContextDirs() (string, []string)

	// CacheID returns the step's cache id after it is set using SetCacheID().
	CacheID() string

	// SetCacheID sets the cache ID of the step given a seed value.
	SetCacheID(ctx *context.BuildContext, seed string) error

	// ApplyCtxAndConfig sets up the execution environment using image config
	// from previous step.
	// This function will not be skipped.
	ApplyCtxAndConfig(ctx *context.BuildContext, imageConfig *image.Config) error

	// Execute executes the step. If modifyFS is true, the command might change
	// the local file system.
	Execute(ctx *context.BuildContext, modifyFS bool) error

	// Commit generates an image layer.
	Commit(ctx *context.BuildContext) ([]*image.DigestPair, error)

	// UpdateCtxAndConfig generates a new image config base on config from
	// previous step.
	// This function will not be skipped.
	UpdateCtxAndConfig(ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error)

	// HasCommit returns whether or not a particular commit step has a commit
	// annotation.
	HasCommit() bool
}

// NewDockerfileStep initializes a build step from a dockerfile directive.
func NewDockerfileStep(
	ctx *context.BuildContext, d dockerfile.Directive, seed string) (BuildStep, error) {

	var err error
	var step BuildStep
	switch t := d.(type) {
	case *dockerfile.AddDirective:
		s, _ := d.(*dockerfile.AddDirective)
		step, err = NewAddStep(s.Args, s.Chown, s.Srcs, s.Dst, s.Commit)
	case *dockerfile.ArgDirective:
		s, _ := d.(*dockerfile.ArgDirective)
		step = NewArgStep(s.Args, s.Name, s.ResolvedVal, s.Commit)
	case *dockerfile.CmdDirective:
		s, _ := d.(*dockerfile.CmdDirective)
		step = NewCmdStep(s.Args, s.Cmd, s.Commit)
	case *dockerfile.CopyDirective:
		s, _ := d.(*dockerfile.CopyDirective)
		step, err = NewCopyStep(s.Args, s.Chown, s.FromStage, s.Srcs, s.Dst, s.Commit)
	case *dockerfile.EntrypointDirective:
		s, _ := d.(*dockerfile.EntrypointDirective)
		step = NewEntrypointStep(s.Args, s.Entrypoint, s.Commit)
	case *dockerfile.EnvDirective:
		s, _ := d.(*dockerfile.EnvDirective)
		step = NewEnvStep(s.Args, s.Envs, s.Commit)
	case *dockerfile.ExposeDirective:
		s, _ := d.(*dockerfile.ExposeDirective)
		step = NewExposeStep(s.Args, s.Ports, s.Commit)
	case *dockerfile.FromDirective:
		s, _ := d.(*dockerfile.FromDirective)
		step, err = NewFromStep(s.Args, s.Image, s.Alias)
	case *dockerfile.HealthcheckDirective:
		s, _ := d.(*dockerfile.HealthcheckDirective)
		step, err = NewHealthcheckStep(s.Args, s.Interval, s.Timeout, s.StartPeriod, s.Retries, s.Test, s.Commit)
	case *dockerfile.LabelDirective:
		s, _ := d.(*dockerfile.LabelDirective)
		step = NewLabelStep(s.Args, s.Labels, s.Commit)
	case *dockerfile.MaintainerDirective:
		s, _ := d.(*dockerfile.MaintainerDirective)
		step = NewMaintainerStep(s.Args, s.Author, s.Commit)
	case *dockerfile.RunDirective:
		s, _ := d.(*dockerfile.RunDirective)
		step = NewRunStep(s.Args, s.Cmd, s.Commit)
	case *dockerfile.StopsignalDirective:
		s, _ := d.(*dockerfile.StopsignalDirective)
		step = NewStopsignalStep(s.Args, s.Signal, s.Commit)
	case *dockerfile.UserDirective:
		s, _ := d.(*dockerfile.UserDirective)
		step = NewUserStep(s.Args, s.User, s.Commit)
	case *dockerfile.VolumeDirective:
		s, _ := d.(*dockerfile.VolumeDirective)
		step = NewVolumeStep(s.Args, s.Volumes, s.Commit)
	case *dockerfile.WorkdirDirective:
		s, _ := d.(*dockerfile.WorkdirDirective)
		step = NewWorkdirStep(s.Args, s.WorkingDir, s.Commit)
	default:
		err = fmt.Errorf("unsupported directive type: %#v", t)
	}
	if err != nil {
		return nil, fmt.Errorf("convert directive: %s", err)
	}
	if err := step.SetCacheID(ctx, seed); err != nil {
		return nil, fmt.Errorf("set cache id: %s", err)
	}
	return step, nil
}
