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
	"strings"
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
	// Args are simply splited by whitespace, and those that are not flags are
	// joined by space later.
	args := whitespaceRegexp.Split(strings.TrimSpace(base.Args), -1)

	var intervalStr, timeoutStr, startPeriodStr, retriesStr string
	var cmdIdx int

	for i, arg := range args {
		if val, ok, err := parseFlag(arg, "interval"); err != nil {
			return nil, base.err(err)
		} else if ok {
			intervalStr = val
			continue
		}

		if val, ok, err := parseFlag(arg, "timeout"); err != nil {
			return nil, base.err(err)
		} else if ok {
			timeoutStr = val
			continue
		}

		if val, ok, err := parseFlag(arg, "start-period"); err != nil {
			return nil, base.err(err)
		} else if ok {
			startPeriodStr = val
			continue
		}

		if val, ok, err := parseFlag(arg, "retries"); err != nil {
			return nil, base.err(err)
		} else if ok {
			retriesStr = val
			continue
		}

		// TODO: Find a better way to handle escaped whitespaces.
		if regexp.MustCompile("^\\\\$").MatchString(arg) {
			continue
		}

		if strings.EqualFold(arg, "none") {
			cmdIdx = i
			if cmdIdx != len(args)-1 {
				return nil, base.err(fmt.Errorf("NONE cannot have arguments"))
			}
			return &HealthcheckDirective{
				baseDirective: base,
				Test:          []string{"None"},
			}, nil
		}
		if strings.EqualFold(arg, "cmd") {
			cmdIdx = i
			if cmdIdx == len(args)-1 {
				return nil, base.err(fmt.Errorf("CMD cannot be empty"))
			}
			break
		}

		return nil, base.err(fmt.Errorf("Unsupported flag %s", arg))
	}

	// Assign defaults.
	var err error
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
	remaining := strings.Join(args[cmdIdx+1:], " ")
	replaced, err := replaceVariables(remaining, state.stageVars)
	if err != nil {
		return nil, base.err(fmt.Errorf("Failed to replace variables in input: %s", err))
	}
	remaining = replaced

	// Parse CMD.
	if cmd, ok := parseJSONArray(remaining); ok {
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
	if _, err := splitArgs(remaining); err != nil {
		return nil, base.err(err)
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
