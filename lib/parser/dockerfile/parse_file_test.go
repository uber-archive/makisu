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
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const _testDir = "test-files"

type test struct {
	desc       string
	dockerfile string
	args       map[string]string
	succeed    bool
	stages     []*Stage
}

var tests []*test

func init() {
	tests = concatSlices(
		fromOnly(),
		invalidDirective(),
		invalidDirectivesBeforeFirstFrom(),
		globalArgs(),
		stageArgs(),
		envs(),
		integration(),
	)
}

func concatSlices(slices ...[]*test) []*test {
	tests := make([]*test, 0)
	for _, slice := range slices {
		for _, test := range slice {
			tests = append(tests, test)
		}
	}
	return tests
}

func TestParseStages(t *testing.T) {
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			require := require.New(t)
			stages, err := ParseFile(test.dockerfile, test.args)
			if test.succeed {
				require.NoError(err)
				require.Equal(test.stages, stages)
			} else {
				require.Error(err)
				require.NotEqual("", err.Error())
			}
		})
	}
}

func TestParseSucceeds(t *testing.T) {
	testFiles, err := ioutil.ReadDir(_testDir)
	if err != nil {
		panic(err)
	}
	for _, f := range testFiles {
		t.Run(f.Name(), func(t *testing.T) {
			require := require.New(t)
			contents, err := ioutil.ReadFile(filepath.Join(_testDir, f.Name()))
			require.NoError(err)
			_, err = ParseFile(string(contents), nil)
			require.NoError(err)
		})
	}
}

func TestRemoveComments(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		contents := `RUN echo asd #!COMMIT
	RUN apt-get install -y qwasd \

		# asdwqe
		zxczxd #!COMMIT
`
		cleaned := `RUN echo asd #!COMMIT
	RUN apt-get install -y qwasd \
		zxczxd #!COMMIT
`
		output := removeCommentLines(contents)
		require.Equal(t, output, cleaned)
	})
}

func invalidDirective() []*test {
	return []*test{{
		desc:       "invalid directive",
		dockerfile: "DIRECTIVE arg1 arg2",
		args:       nil,
		succeed:    false,
		stages:     nil,
	}}
}

func fromOnly() []*test {
	tests := make([]*test, 0)

	dockerfile := `
	FROM alpine:latest AS alias       
	`

	stage := newStage(&FromDirective{
		&baseDirective{"from", "alpine:latest AS alias", false},
		"alpine:latest",
		"alias",
	})

	tests = append(tests, &test{
		desc:       "one FROM only",
		dockerfile: dockerfile,
		args:       nil,
		succeed:    true,
		stages:     []*Stage{stage},
	})

	dockerfile = `
	FROM alpine:latest AS alias1
	FROM   ubuntu:trusty AS alias2
	FROM   ubuntu:trusty AS alias3
	`

	stage1 := newStage(&FromDirective{
		&baseDirective{"from", "alpine:latest AS alias1", false},
		"alpine:latest",
		"alias1",
	})
	stage2 := newStage(&FromDirective{
		&baseDirective{"from", "ubuntu:trusty AS alias2", false},
		"ubuntu:trusty",
		"alias2",
	})
	stage3 := newStage(&FromDirective{
		&baseDirective{"from", "ubuntu:trusty AS alias3", false},
		"ubuntu:trusty",
		"alias3",
	})

	tests = append(tests, &test{
		desc:       "three FROMs only",
		dockerfile: dockerfile,
		args:       nil,
		succeed:    true,
		stages:     []*Stage{stage1, stage2, stage3},
	})

	return tests
}

func invalidDirectivesBeforeFirstFrom() []*test {
	tests := make([]*test, 0)

	directives := []string{
		"RUN", "CMD", "LABEL", "EXPOSE", "COPY", "ENTRYPOINT",
		"ENV", "ADD", "USER", "VOLUME", "WORKDIR",
	}

	for _, d := range directives {
		tests = append(tests, &test{
			desc:       fmt.Sprintf("%s before first FROM", d),
			dockerfile: d,
			args:       nil,
			succeed:    false,
			stages:     nil,
		})
	}
	return tests
}

func globalArgs() []*test {
	tests := make([]*test, 0)

	dockerfile := `
	FROM ${image}:latest AS alias1
	`
	stage := newStage(&FromDirective{
		&baseDirective{"from", "${image}:latest AS alias1", false},
		"${image}:latest",
		"alias1",
	})
	tests = append(tests, &test{
		desc:       "global arg missing",
		dockerfile: dockerfile,
		args:       nil,
		succeed:    true,
		stages:     []*Stage{stage},
	})

	dockerfile = `
	ARG image
	FROM ${image}:latest AS alias1
	`
	stage = newStage(&FromDirective{
		&baseDirective{"from", "${image}:latest AS alias1", false},
		"${image}:latest",
		"alias1",
	})
	tests = append(tests, &test{
		desc:       "global arg not set",
		dockerfile: dockerfile,
		args:       nil,
		succeed:    true,
		stages:     []*Stage{stage},
	})

	dockerfile = `
	ARG image
	FROM ${image}:latest AS alias1
	`
	stage = newStage(&FromDirective{
		&baseDirective{"from", "alpine:latest AS alias1", false},
		"alpine:latest",
		"alias1",
	})

	tests = append(tests, &test{
		desc:       "global arg no default",
		dockerfile: dockerfile,
		args:       map[string]string{"image": "alpine"},
		succeed:    true,
		stages:     []*Stage{stage},
	})

	dockerfile = `
	ARG image=alpine
	FROM ${image}:latest AS alias1
	`
	stage = newStage(&FromDirective{
		&baseDirective{"from", "alpine:latest AS alias1", false},
		"alpine:latest",
		"alias1",
	})

	tests = append(tests, &test{
		desc:       "global arg default not set",
		dockerfile: dockerfile,
		args:       nil,
		succeed:    true,
		stages:     []*Stage{stage},
	})

	stage = newStage(&FromDirective{
		&baseDirective{"from", "ubuntu:latest AS alias1", false},
		"ubuntu:latest",
		"alias1",
	})

	tests = append(tests, &test{
		desc:       "global arg default set",
		dockerfile: dockerfile,
		args:       map[string]string{"image": "ubuntu"},
		succeed:    true,
		stages:     []*Stage{stage},
	})

	return tests
}

func stageArgs() []*test {
	tests := make([]*test, 0)

	dockerfile := `
	ARG cmd
	FROM alpine:latest AS alias1
	CMD ${cmd}
	`
	stage := newStage(&FromDirective{
		&baseDirective{"from", "alpine:latest AS alias1", false},
		"alpine:latest",
		"alias1",
	})
	stage.addDirective(&CmdDirective{
		&baseDirective{"cmd", "${cmd}", false},
		[]string{"${cmd}"},
	})

	tests = append(tests, &test{
		desc:       "stage arg missing",
		dockerfile: dockerfile,
		args:       map[string]string{"cmd": "ls"},
		succeed:    true,
		stages:     []*Stage{stage},
	})

	dockerfile = `
	FROM alpine:latest AS alias1
	ARG cmd
	CMD ${cmd}
	`
	stage = newStage(&FromDirective{
		&baseDirective{"from", "alpine:latest AS alias1", false},
		"alpine:latest",
		"alias1",
	})
	stage.addDirective(&ArgDirective{
		&baseDirective{"arg", "cmd", false},
		"cmd",
		"",
		nil,
	})
	stage.addDirective(&CmdDirective{
		&baseDirective{"cmd", "${cmd}", false},
		[]string{"${cmd}"},
	})

	tests = append(tests, &test{
		desc:       "stage arg not set",
		dockerfile: dockerfile,
		args:       nil,
		succeed:    true,
		stages:     []*Stage{stage},
	})

	dockerfile = `
	FROM alpine:latest AS alias1
	ARG cmd
	CMD ${cmd}
	FROM alpine:latest AS alias2
	CMD ${cmd}
	`
	stage1 := newStage(&FromDirective{
		&baseDirective{"from", "alpine:latest AS alias1", false},
		"alpine:latest",
		"alias1",
	})
	paramVal := "ls"
	stage1.addDirective(&ArgDirective{
		&baseDirective{"arg", "cmd", false},
		"cmd",
		"",
		&paramVal,
	})
	stage1.addDirective(&CmdDirective{
		&baseDirective{"cmd", "ls", false},
		[]string{"ls"},
	})
	stage2 := newStage(&FromDirective{
		&baseDirective{"from", "alpine:latest AS alias2", false},
		"alpine:latest",
		"alias2",
	})
	stage2.addDirective(&CmdDirective{
		&baseDirective{"cmd", "${cmd}", false},
		[]string{"${cmd}"},
	})

	tests = append(tests, &test{
		desc:       "stage arg set in previous stage",
		dockerfile: dockerfile,
		args:       map[string]string{"cmd": "ls"},
		succeed:    true,
		stages:     []*Stage{stage1, stage2},
	})

	dockerfile = `
	FROM alpine:latest AS alias1
	ARG cmd
	CMD ${cmd}
	`
	stage = newStage(&FromDirective{
		&baseDirective{"from", "alpine:latest AS alias1", false},
		"alpine:latest",
		"alias1",
	})
	paramVal = "ls"
	stage.addDirective(&ArgDirective{
		&baseDirective{"arg", "cmd", false},
		"cmd",
		"",
		&paramVal,
	})
	stage.addDirective(&CmdDirective{
		&baseDirective{"cmd", "ls", false},
		[]string{"ls"},
	})

	tests = append(tests, &test{
		desc:       "stage arg no default",
		dockerfile: dockerfile,
		args:       map[string]string{"cmd": "ls"},
		succeed:    true,
		stages:     []*Stage{stage},
	})

	dockerfile = `
	ARG cmd
	CMD ${cmd}
	`

	tests = append(tests, &test{
		desc:       "replace local before first FROM",
		dockerfile: dockerfile,
		args:       map[string]string{"cmd": "ls"},
		succeed:    false,
		stages:     nil,
	})

	return tests
}

func envs() []*test {
	tests := make([]*test, 0)

	dockerfile := `
	FROM alpine:latest AS alias1
	ENV cmd ls
	CMD ${cmd}

	FROM alpine:latest AS alias2
	CMD ${cmd}
	`
	stage1 := newStage(&FromDirective{
		&baseDirective{"from", "alpine:latest AS alias1", false},
		"alpine:latest",
		"alias1",
	})
	stage1.addDirective(&EnvDirective{
		&baseDirective{"env", "cmd ls", false},
		map[string]string{"cmd": "ls"},
	})
	stage1.addDirective(&CmdDirective{
		&baseDirective{"cmd", "ls", false},
		[]string{"ls"},
	})
	stage2 := newStage(&FromDirective{
		&baseDirective{"from", "alpine:latest AS alias2", false},
		"alpine:latest",
		"alias2",
	})
	stage2.addDirective(&CmdDirective{
		&baseDirective{"cmd", "${cmd}", false},
		[]string{"${cmd}"},
	})

	tests = append(tests, &test{
		desc:       "env defined in previous stage",
		dockerfile: dockerfile,
		args:       nil,
		succeed:    true,
		stages:     []*Stage{stage1, stage2},
	})

	dockerfile = `
	FROM alpine:latest AS alias1
	ENV cmd ls
	ENV cmd ls -la
	ENV cmd="ls -la" cmd2=echo
	ENV empty="" nonEmpty="true"
	CMD ${cmd}
	CMD ${cmd2}
	`
	stage := newStage(&FromDirective{
		&baseDirective{"from", "alpine:latest AS alias1", false},
		"alpine:latest",
		"alias1",
	})
	stage.addDirective(&EnvDirective{
		&baseDirective{"env", "cmd ls", false},
		map[string]string{"cmd": "ls"},
	})
	stage.addDirective(&EnvDirective{
		&baseDirective{"env", "cmd ls -la", false},
		map[string]string{"cmd": "ls -la"},
	})
	stage.addDirective(&EnvDirective{
		&baseDirective{"env", "cmd=\"ls -la\" cmd2=echo", false},
		map[string]string{"cmd": "ls -la", "cmd2": "echo"},
	})
	stage.addDirective(&EnvDirective{
		&baseDirective{"env", "empty=\"\" nonEmpty=\"true\"", false},
		map[string]string{"empty": "", "nonEmpty": "true"},
	})
	stage.addDirective(&CmdDirective{
		&baseDirective{"cmd", "ls -la", false},
		[]string{"ls", "-la"},
	})
	stage.addDirective(&CmdDirective{
		&baseDirective{"cmd", "echo", false},
		[]string{"echo"},
	})

	tests = append(tests, &test{
		desc:       "env overwrite",
		dockerfile: dockerfile,
		args:       nil,
		succeed:    true,
		stages:     []*Stage{stage},
	})

	return tests
}

func integration() []*test {
	tests := make([]*test, 0)

	dockerfile := `
	ARG image=alpine
	ARG alias

	FROM ${image}:latest AS ${alias}1
	ARG cmd=ls
	ENV image=ubuntu cmd="${cmd} ${cmd}"
	RUN $cmd $image
	CMD $cmd $image
	CMD ["${cmd}", "${image}"]

	FROM ${image}:latest AS ${alias}2
	ARG key
	ENV dir1 home
	ARG dir2=dir
	label k1=v1 k2=${key}
	cOpY --from=digest --chown=user:group src1 src2 src3 dst/ #!commit
	WORKDIR /path/to/${dir1}/$dir2

	FROM ${image}:latest AS ${alias}3

	MAINTAINER  ${alias}-maintainer <${alias}@example.com>

	add --chown=user:group ["src1", "src2", "src3", "dst/"]   #! commit
	arg cmd
	ENTRYPOINT ["bash", "$cmd"]
	VOLUME v1 v2
	EXPOSE 80/tcp 81 82/udp
	ENV PATH=/tmp:$PATH
	ENV PATH=/tmp2:$PATH
	USER udocker
	`
	args := map[string]string{"alias": "test_alias", "cmd": "echo", "key": "v2"}

	stage1 := newStage(&FromDirective{
		&baseDirective{"from", "alpine:latest AS test_alias1", false},
		"alpine:latest",
		"test_alias1",
	})
	paramVal1 := "echo"
	stage1.addDirective(&ArgDirective{
		&baseDirective{"arg", "cmd=ls", false},
		"cmd",
		"ls",
		&paramVal1,
	})
	stage1.addDirective(&EnvDirective{
		&baseDirective{"env", "image=ubuntu cmd=\"echo echo\"", false},
		map[string]string{"image": "ubuntu", "cmd": "echo echo"},
	})
	stage1.addDirective(&RunDirective{
		&baseDirective{"run", "echo echo ubuntu", false},
		"echo echo ubuntu",
	})
	stage1.addDirective(&CmdDirective{
		&baseDirective{"cmd", "echo echo ubuntu", false},
		[]string{"echo", "echo", "ubuntu"},
	})
	stage1.addDirective(&CmdDirective{
		&baseDirective{"cmd", `["echo echo", "ubuntu"]`, false},
		[]string{"echo echo", "ubuntu"},
	})

	stage2 := newStage(&FromDirective{
		&baseDirective{"from", "alpine:latest AS test_alias2", false},
		"alpine:latest",
		"test_alias2",
	})
	paramVal2 := "v2"
	stage2.addDirective(&ArgDirective{
		&baseDirective{"arg", "key", false},
		"key",
		"",
		&paramVal2,
	})
	stage2.addDirective(&EnvDirective{
		&baseDirective{"env", "dir1 home", false},
		map[string]string{"dir1": "home"},
	})
	defaultVal1 := "dir"
	stage2.addDirective(&ArgDirective{
		&baseDirective{"arg", "dir2=dir", false},
		"dir2",
		"dir",
		&defaultVal1,
	})
	stage2.addDirective(&LabelDirective{
		&baseDirective{"label", "k1=v1 k2=v2", false},
		map[string]string{"k1": "v1", "k2": "v2"},
	})
	stage2.addDirective(&CopyDirective{
		&addCopyDirective{
			&baseDirective{"copy", "--from=digest --chown=user:group src1 src2 src3 dst/", true},
			"user:group",
			[]string{"src1", "src2", "src3"},
			"dst/",
		},
		"digest",
	})
	stage2.addDirective(&WorkdirDirective{
		&baseDirective{"workdir", "/path/to/home/dir", false},
		"/path/to/home/dir",
	})

	stage3 := newStage(&FromDirective{
		&baseDirective{"from", "alpine:latest AS test_alias3", false},
		"alpine:latest",
		"test_alias3",
	})
	stage3.addDirective(&MaintainerDirective{
		&baseDirective{"maintainer", `${alias}-maintainer <${alias}@example.com>`, false},
		"${alias}-maintainer <${alias}@example.com>",
	})
	stage3.addDirective(&AddDirective{
		&addCopyDirective{
			&baseDirective{"add", `--chown=user:group ["src1", "src2", "src3", "dst/"]`, true},
			"user:group",
			[]string{"src1", "src2", "src3"},
			"dst/",
		},
	})
	stage3.addDirective(&ArgDirective{
		&baseDirective{"arg", "cmd", false},
		"cmd",
		"",
		&paramVal1,
	})
	stage3.addDirective(&EntrypointDirective{
		&baseDirective{"entrypoint", `["bash", "echo"]`, false},
		[]string{"bash", "echo"},
	})
	stage3.addDirective(&VolumeDirective{
		&baseDirective{"volume", "v1 v2", false},
		[]string{"v1", "v2"},
	})
	stage3.addDirective(&ExposeDirective{
		&baseDirective{"expose", "80/tcp 81 82/udp", false},
		[]string{"80/tcp", "81", "82/udp"},
	})
	stage3.addDirective(&EnvDirective{
		&baseDirective{"env", "PATH=/tmp:$PATH", false},
		map[string]string{"PATH": "/tmp:$PATH"},
	})
	stage3.addDirective(&EnvDirective{
		&baseDirective{"env", "PATH=/tmp2:/tmp:$PATH", false},
		map[string]string{"PATH": "/tmp2:/tmp:$PATH"},
	})
	stage3.addDirective(&UserDirective{
		&baseDirective{"user", "udocker", false},
		"udocker",
	})

	tests = append(tests, &test{
		desc:       "integration",
		dockerfile: dockerfile,
		args:       args,
		succeed:    true,
		stages:     []*Stage{stage1, stage2, stage3},
	})

	return tests
}
