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

package dockerfile

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// HeathcheckDirective represents the "LABEL" dockerfile command.
type HealthcheckDirective struct {
	*baseDirective

	Interval    time.Duration
	Timeout     time.Duration
	StartPeriod time.Duration
	Retries     int

	Test []string
}

// Variables:
//   Replaced from ARGs and ENVs from within our stage.
// Formats:
//   HEALTHCHECK NONE
//   HEALTHCHECK [--interval=<t>] [--timeout=<t>] [--start-period=<t>] [--retries=<n>] \
//     CMD ["<param>"...]
//   HEALTHCHECK [--interval=<t>] [--timeout=<t>] [--start-period=<t>] [--retries=<n>] \
//     CMD <command> <param>...
func newHealthcheckDirective(base *baseDirective, state *parsingState) (Directive, error) {
	// TODO: regexp is not the ideal solution.
	if isNone := regexp.MustCompile(`^[\s|\\]*none[\s|\\]*$`).MatchString(base.Args); isNone {
		return &HealthcheckDirective{
			baseDirective: base,
			Test:          []string{"None"},
		}, nil
	}
	cmdIndices := regexp.MustCompile(`[\s|\\]*cmd[\s|\\]*`).FindStringIndex(base.Args)
	if len(cmdIndices) < 2 {
		return nil, base.err(fmt.Errorf("CMD not defined"))
	}

	flags, err := splitArgs(base.Args[:cmdIndices[0]])
	if err != nil {
		return nil, fmt.Errorf("failed to parse interval")
	}
	var intervalStr, timeoutStr, startPeriodStr, retriesStr string
	for _, flag := range flags {
		if val, ok, err := parseFlag(flag, "interval"); err != nil {
			return nil, base.err(err)
		} else if ok {
			intervalStr = val
			continue
		}

		if val, ok, err := parseFlag(flag, "timeout"); err != nil {
			return nil, base.err(err)
		} else if ok {
			timeoutStr = val
			continue
		}

		if val, ok, err := parseFlag(flag, "start-period"); err != nil {
			return nil, base.err(err)
		} else if ok {
			startPeriodStr = val
			continue
		}

		if val, ok, err := parseFlag(flag, "retries"); err != nil {
			return nil, base.err(err)
		} else if ok {
			retriesStr = val
			continue
		}

		return nil, base.err(fmt.Errorf("Unsupported flag %s", flag))
	}

	// Assign defaults.
	var interval, timeout, startPeriod time.Duration
	var retries int

	if intervalStr == "" {
		intervalStr = "30s"
	}
	interval, err = time.ParseDuration(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse interval")
	}

	if timeoutStr == "" {
		timeoutStr = "30s"
	}
	timeout, err = time.ParseDuration(timeoutStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timeout")
	}

	if startPeriodStr == "" {
		startPeriodStr = "0s"
	}
	startPeriod, err = time.ParseDuration(startPeriodStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start-period")
	}

	if retriesStr == "" {
		retriesStr = "3"
	}
	retries, err = strconv.Atoi(retriesStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse retries")
	}

	// Replace variables.
	if state.stageVars == nil {
		return nil, base.err(errBeforeFirstFrom)
	}
	remaining := base.Args[cmdIndices[1]:]
	replaced, err := replaceVariables(remaining, state.stageVars)
	if err != nil {
		return nil, base.err(fmt.Errorf("Failed to replace variables in input: %s", err))
	}
	remaining = replaced

	// Parse CMD.
	if cmd, ok := parseJSONArray(remaining); ok {
		if len(cmd) == 0 {
			return nil, base.err(fmt.Errorf("missing CMD arguments: %s", err))
		}

		return &HealthcheckDirective{
			baseDirective: base,
			Interval:      interval,
			Timeout:       timeout,
			StartPeriod:   startPeriod,
			Retries:       retries,
			Test:          append([]string{"CMD"}, cmd...),
		}, nil
	}

	// Verify cmd arg is a valid array, but return the whole arg as one string.
	args, err := splitArgs(remaining)
	if err != nil {
		return nil, base.err(err)
	}
	if len(args) == 0 {
		return nil, base.err(fmt.Errorf("missing CMD arguments: %s", err))
	}

	return &HealthcheckDirective{
		baseDirective: base,
		Interval:      interval,
		Timeout:       timeout,
		StartPeriod:   startPeriod,
		Retries:       retries,
		Test:          append([]string{"CMD-SHELL"}, remaining),
	}, nil
}

// Add this command to the build stage.
func (d *HealthcheckDirective) update(state *parsingState) error {
	return state.addToCurrStage(d)
}
