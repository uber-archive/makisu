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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewHealthcheckDirective(t *testing.T) {
	buildState := newParsingState(make(map[string]string))
	buildState.stageVars = map[string]string{"prefix": "test_", "suffix": "_test", "comma": ","}

	d15, _ := time.ParseDuration("15s")
	d5, _ := time.ParseDuration("5s")
	d0, _ := time.ParseDuration("0s")

	tests := []struct {
		desc        string
		succeed     bool
		input       string
		interval    time.Duration
		timeout     time.Duration
		startPeriod time.Duration
		retries     int
		test        []string
	}{
		{"none", true, "healthcheck none", d0, d0, d0, 0, []string{"None"}},
		{"none escaped", true, "healthcheck \\\nnoNE", d0, d0, d0, 0, []string{"None"}},
		{"empty cmd", false, "healthcheck cmd", d0, d0, d0, 0, nil},
		{"substitution", true, `healthcheck cMD ["${prefix}this", "cmd${suffix}"]`, d0, d0, d0, 0, []string{"CMD", "test_this", "cmd_test"}},
		{"substitution 2", true, `healthcheck cmd ["this"$comma "cmd"]`, d0, d0, d0, 0, []string{"CMD", "this", "cmd"}},
		{"good cmd", true, "healthcheck --interval=15s --timeout=5s --start-period=5s --retries=10\\\n \\\ncmd this cmd", d15, d5, d5, 10, []string{"CMD-SHELL", "this cmd"}},
		{"quotes", true, `healthcheck cmd "this cmd"`, d0, d0, d0, 0, []string{"CMD-SHELL", "\"this cmd\""}},
		{"quotes 2", true, `healthcheck cmd "this cmd" cmd2 "and cmd 3"`, d0, d0, d0, 0, []string{"CMD-SHELL", "\"this cmd\" cmd2 \"and cmd 3\""}},
		{"substitution", true, "healthcheck cmd ${prefix}this cmd$suffix", d0, d0, d0, 0, []string{"CMD-SHELL", "test_this cmd_test"}},
		{"good json", true, `healthcheck cmd ["this", "cmd"]`, d0, d0, d0, 0, []string{"CMD", "this", "cmd"}},
		{"bad json", false, `healthcheck cmd ["this, "cmd"]`, d0, d0, d0, 0, nil},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			directive, err := newDirective(test.input, buildState)
			if test.succeed {
				require.NoError(err)
				healthcheck, ok := directive.(*HealthcheckDirective)
				require.True(ok)
				require.Equal(test.interval, healthcheck.Interval)
				require.Equal(test.timeout, healthcheck.Timeout)
				require.Equal(test.startPeriod, healthcheck.StartPeriod)
				require.Equal(test.retries, healthcheck.Retries)
				require.Equal(test.test, healthcheck.Test)
			} else {
				require.Error(err)
			}
		})
	}
}
