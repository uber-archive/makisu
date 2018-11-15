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

// FromDirectiveFixture returns a FromDirective for testing purposes.
func FromDirectiveFixture(args, image, alias string) *FromDirective {
	return &FromDirective{&baseDirective{"from", args, false}, image, alias}
}

// RunDirectiveFixture returns a RunDirective for testing purposes.
func RunDirectiveFixture(args string, cmd string) *RunDirective {
	return &RunDirective{&baseDirective{"run", args, false}, cmd}
}

// RunCommitDirectiveFixture returns a RunDirective with a commit annotation
// for testing purposes.
func RunCommitDirectiveFixture(args string, cmd string) *RunDirective {
	return &RunDirective{&baseDirective{"run", args, true}, cmd}
}

// CmdDirectiveFixture returns a CmdDirective for testing purposes.
func CmdDirectiveFixture(args string, cmd []string) *CmdDirective {
	return &CmdDirective{&baseDirective{"cmd", args, false}, cmd}
}

// LabelDirectiveFixture returns a LabelDirective for testing purposes.
func LabelDirectiveFixture(args string, labels map[string]string) *LabelDirective {
	return &LabelDirective{&baseDirective{"label", args, false}, labels}
}

// ExposeDirectiveFixture returns a ExposeDirective for testing purposes.
func ExposeDirectiveFixture(args string, ports []string) *ExposeDirective {
	return &ExposeDirective{&baseDirective{"expose", args, false}, ports}
}

// CopyDirectiveFixture returns a CopyDirective for testing purposes.
func CopyDirectiveFixture(args, chown, fromStage string, srcs []string, dst string) *CopyDirective {
	return &CopyDirective{
		&addCopyDirective{
			&baseDirective{"copy", args, false},
			chown,
			srcs,
			dst,
		},
		fromStage,
	}
}

// EntrypointDirectiveFixture returns a EntrypointDirective for testing purposes.
func EntrypointDirectiveFixture(args string, entrypoint []string) *EntrypointDirective {
	return &EntrypointDirective{&baseDirective{"entrypoint", args, false}, entrypoint}
}

// EnvDirectiveFixture returns a EnvDirective for testing purposes.
func EnvDirectiveFixture(args string, envs map[string]string) *EnvDirective {
	return &EnvDirective{&baseDirective{"env", args, false}, envs}
}

// UserDirectiveFixture returns a UserDirective for testing purposes.
func UserDirectiveFixture(args, user string) *UserDirective {
	return &UserDirective{&baseDirective{"user", args, false}, user}
}

// VolumeDirectiveFixture returns a VolumeDirective for testing purposes.
func VolumeDirectiveFixture(args string, volumes []string) *VolumeDirective {
	return &VolumeDirective{&baseDirective{"volume", args, false}, volumes}
}

// WorkdirDirectiveFixture returns a WorkdirDirective for testing purposes.
func WorkdirDirectiveFixture(args string, workdir string) *WorkdirDirective {
	return &WorkdirDirective{&baseDirective{"workdir", args, false}, workdir}
}

// AddDirectiveFixture returns an AddDirective for testing purposes.
func AddDirectiveFixture(args, chown string, srcs []string, dst string) *AddDirective {
	return &AddDirective{
		&addCopyDirective{
			&baseDirective{"add", args, false},
			chown,
			srcs,
			dst,
		},
	}
}
