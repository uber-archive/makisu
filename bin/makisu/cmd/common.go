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

package cmd

import (
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/uber/makisu/lib/log"
	"go.uber.org/zap"
)

func (cmd *rootCmd) processGlobalFlags() error {
	// Initializes logger.
	logger, err := cmd.getLogger()
	if err != nil {
		return fmt.Errorf("configure logger: %s", err)
	}
	log.SetLogger(logger.Sugar())

	if cmd.cpuProfile {
		// Set up profiling.
		if err := setupProfiler(); err != nil {
			return fmt.Errorf("setup profiler: %s", err)
		}
		cmd.cleanup = func() { pprof.StopCPUProfile() }
		return nil
	}
	return nil
}

func (cmd *rootCmd) getLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	if cmd.logOutput != "stdout" {
		config.OutputPaths = []string{cmd.logOutput}
	}

	if err := config.Level.UnmarshalText([]byte(cmd.logLevel)); err != nil {
		return nil, fmt.Errorf("parse log level: %s", err)
	}

	config.Encoding = cmd.logFormat
	config.DisableStacktrace = true
	config.DisableCaller = true
	return config.Build()
}

func setupProfiler() error {
	f, err := os.Create("/tmp/makisu.prof")
	if err != nil {
		return fmt.Errorf("create profile file: %s", err)
	}
	pprof.StartCPUProfile(f)

	return nil
}
