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
	"time"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
)

// HealthcheckStep implements BuildStep and execute HEALTHCHECK directive
type HealthcheckStep struct {
	*baseStep

	Interval    time.Duration
	Timeout     time.Duration
	StartPeriod time.Duration
	Retries     int

	Test []string
}

// NewHealthcheckStep returns a BuildStep from given arguments.
func NewHealthcheckStep(
	args string, interval, timeout, startPeriod time.Duration, retries int,
	test []string, commit bool) (BuildStep, error) {

	return &HealthcheckStep{
		baseStep:    newBaseStep(Healthcheck, args, commit),
		Interval:    interval,
		Timeout:     timeout,
		StartPeriod: startPeriod,
		Retries:     retries,
		Test:        test,
	}, nil
}

// UpdateCtxAndConfig updates mutable states in build context, and generates a
// new image config base on config from previous step.
func (s *HealthcheckStep) UpdateCtxAndConfig(
	ctx *context.BuildContext, imageConfig *image.Config) (*image.Config, error) {

	config, err := image.NewImageConfigFromCopy(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("copy image config: %s", err)
	}
	config.Config.Healthcheck = &image.HealthConfig{
		Interval:    s.Interval,
		Timeout:     s.Timeout,
		StartPeriod: s.StartPeriod,
		Retries:     s.Retries,
		Test:        s.Test,
	}
	return config, nil
}
