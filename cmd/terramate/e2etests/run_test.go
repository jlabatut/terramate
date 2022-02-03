// Copyright 2021 Mineiros GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2etest

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/madlambda/spells/assert"
	"github.com/mineiros-io/terramate"

	"github.com/mineiros-io/terramate/cmd/terramate/cli"
	"github.com/mineiros-io/terramate/hcl"
	"github.com/mineiros-io/terramate/project"
	"github.com/mineiros-io/terramate/run/dag"
	"github.com/mineiros-io/terramate/test"
	"github.com/mineiros-io/terramate/test/sandbox"
)

func TestCLIRunOrder(t *testing.T) {
	type testcase struct {
		name   string
		layout []string
		want   runExpected
	}

	for _, tc := range []testcase{
		{
			name: "one stack",
			layout: []string{
				"s:stack-a",
			},
			want: runExpected{
				Stdout: `stack-a
`,
			},
		},
		{
			name: "empty ordering",
			layout: []string{
				"s:stack:after=[]",
			},
			want: runExpected{
				Stdout: `stack
`,
			},
		},
		{
			name: "independent stacks, consistent ordering (lexicographic)",
			layout: []string{
				"s:batatinha",
				"s:frita",
				"s:1",
				"s:2",
				"s:3",
				"s:boom",
			},
			want: runExpected{
				Stdout: `1
2
3
batatinha
boom
frita
`,
			},
		},
		{
			name: "stack-b after stack-a (relpaths)",
			layout: []string{
				"s:stack-a",
				`s:stack-b:after=["../stack-a"]`,
			},
			want: runExpected{
				Stdout: `stack-a
stack-b
`,
			},
		},
		{
			name: "stack-b after stack-a (abspaths)",
			layout: []string{
				"s:stack-a",
				`s:stack-b:after=["/stack-a"]`,
			},
			want: runExpected{
				Stdout: `stack-a
stack-b
`,
			},
		},
		{
			name: "stack-c after stack-b after stack-a (relpaths)",
			layout: []string{
				"s:stack-a",
				`s:stack-b:after=["../stack-a"]`,
				`s:stack-c:after=["../stack-b"]`,
			},
			want: runExpected{
				Stdout: `stack-a
stack-b
stack-c
`,
			},
		},
		{
			name: "stack-c after stack-b after stack-a (abspaths)",
			layout: []string{
				"s:stack-a",
				`s:stack-b:after=["/stack-a"]`,
				`s:stack-c:after=["/stack-b"]`,
			},
			want: runExpected{
				Stdout: `stack-a
stack-b
stack-c
`,
			},
		},
		{
			name: "stack-a after stack-b after stack-c (relpaths)",
			layout: []string{
				"s:stack-c",
				`s:stack-b:after=["../stack-c"]`,
				`s:stack-a:after=["../stack-b"]`,
			},
			want: runExpected{
				Stdout: `stack-c
stack-b
stack-a
`,
			},
		},
		{
			name: "stack-a after stack-b after stack-c (abspaths)",
			layout: []string{
				"s:stack-c",
				`s:stack-b:after=["/stack-c"]`,
				`s:stack-a:after=["/stack-b"]`,
			},
			want: runExpected{
				Stdout: `stack-c
stack-b
stack-a
`,
			},
		},
		{
			name: "stack-a after stack-b (relpaths)",
			layout: []string{
				`s:stack-a:after=["../stack-b"]`,
				`s:stack-b`,
			},
			want: runExpected{
				Stdout: `stack-b
stack-a
`,
			},
		},
		{
			name: "stack-a after (stack-b, stack-c, stack-d)",
			layout: []string{
				`s:stack-a:after=["../stack-b", "../stack-c", "../stack-d"]`,
				`s:stack-b`,
				`s:stack-c`,
				`s:stack-d`,
			},
			want: runExpected{
				Stdout: `stack-b
stack-c
stack-d
stack-a
`,
			},
		},
		{
			name: "stack-a after (stack-b, stack-c, stack-d) (abspaths)",
			layout: []string{
				`s:stack-a:after=["/stack-b", "/stack-c", "/stack-d"]`,
				`s:stack-b`,
				`s:stack-c`,
				`s:stack-d`,
			},
			want: runExpected{
				Stdout: `stack-b
stack-c
stack-d
stack-a
`,
			},
		},
		{
			name: "stack-c after stack-b after stack-a, stack-d after stack-z (relpaths)",
			layout: []string{
				`s:stack-c:after=["../stack-b"]`,
				`s:stack-b:after=["../stack-a"]`,
				`s:stack-a`,
				`s:stack-d:after=["../stack-z"]`,
				`s:stack-z`,
			},
			want: runExpected{
				Stdout: `stack-a
stack-b
stack-c
stack-z
stack-d
`,
			},
		},
		{
			name: "stack-c after stack-b after stack-a, stack-d after stack-z (abspaths)",
			layout: []string{
				`s:stack-c:after=["/stack-b"]`,
				`s:stack-b:after=["/stack-a"]`,
				`s:stack-a`,
				`s:stack-d:after=["/stack-z"]`,
				`s:stack-z`,
			},
			want: runExpected{
				Stdout: `stack-a
stack-b
stack-c
stack-z
stack-d
`,
			},
		},
		{
			name: "stack-c after stack-b after stack-a, stack-d after stack-b",
			layout: []string{
				`s:stack-c:after=["../stack-b"]`,
				`s:stack-b:after=["../stack-a"]`,
				`s:stack-a`,
				`s:stack-d:after=["../stack-b"]`,
			},
			want: runExpected{
				Stdout: `stack-a
stack-b
stack-c
stack-d
`,
			},
		},
		{
			name: "stack-c after stack-b after stack-a, stack-z after stack-d after stack-b",
			layout: []string{
				`s:stack-c:after=["../stack-b"]`,
				`s:stack-b:after=["../stack-a"]`,
				`s:stack-a`,
				`s:stack-z:after=["../stack-d"]`,
				`s:stack-d:after=["../stack-b"]`,
			},
			want: runExpected{
				Stdout: `stack-a
stack-b
stack-c
stack-d
stack-z
`,
			},
		},
		{
			name: "stack-g after stack-c after stack-b after stack-a, stack-z after stack-d after stack-b",
			layout: []string{
				`s:stack-g:after=["../stack-c"]`,
				`s:stack-c:after=["../stack-b"]`,
				`s:stack-b:after=["../stack-a"]`,
				`s:stack-a`,
				`s:stack-z:after=["../stack-d"]`,
				`s:stack-d:after=["../stack-b"]`,
			},
			want: runExpected{
				Stdout: `stack-a
stack-b
stack-c
stack-d
stack-g
stack-z
`,
			},
		},
		{
			name: "stack-a after (stack-b, stack-c), stack-b after (stack-d, stack-f), stack-c after (stack-g, stack-h)",
			layout: []string{
				`s:stack-a:after=["../stack-b", "../stack-c"]`,
				`s:stack-b:after=["../stack-d", "../stack-f"]`,
				`s:stack-c:after=["../stack-g", "../stack-h"]`,
				`s:stack-d`,
				`s:stack-f`,
				`s:stack-g`,
				`s:stack-h`,
			},
			want: runExpected{
				Stdout: `stack-d
stack-f
stack-b
stack-g
stack-h
stack-c
stack-a
`,
			},
		},
		{
			name: "stack-z after (stack-a, stack-b, stack-c, stack-d), stack-a after (stack-b, stack-c)",
			layout: []string{
				`s:stack-z:after=["../stack-a", "../stack-b", "../stack-c", "../stack-d"]`,
				`s:stack-a:after=["../stack-b", "../stack-c"]`,
				`s:stack-b`,
				`s:stack-c`,
				`s:stack-d`,
			},
			want: runExpected{
				Stdout: `stack-b
stack-c
stack-a
stack-d
stack-z
`,
			},
		},
		{
			name: "stack-z after (stack-a, stack-b, stack-c, stack-d), stack-a after (stack-x, stack-y)",
			layout: []string{
				`s:stack-z:after=["../stack-a", "../stack-b", "../stack-c", "../stack-d"]`,
				`s:stack-a:after=["../stack-x", "../stack-y"]`,
				`s:stack-b`,
				`s:stack-c`,
				`s:stack-d`,
				`s:stack-x`,
				`s:stack-y`,
			},
			want: runExpected{
				Stdout: `stack-x
stack-y
stack-a
stack-b
stack-c
stack-d
stack-z
`,
			},
		},
		{
			name: "stack-a after stack-a - fails",
			layout: []string{
				`s:stack-a:after=["../stack-a"]`,
			},
			want: runExpected{
				Status:      defaultErrExitStatus,
				StderrRegex: dag.ErrCycleDetected.Error(),
			},
		},
		{
			name: "stack-a after . - fails",
			layout: []string{
				`s:stack-a:after=["."]`,
			},
			want: runExpected{
				Status:      defaultErrExitStatus,
				StderrRegex: dag.ErrCycleDetected.Error(),
			},
		},
		{
			name: "stack-a after stack-b after stack-c after stack-a - fails",
			layout: []string{
				`s:stack-a:after=["../stack-b"]`,
				`s:stack-b:after=["../stack-c"]`,
				`s:stack-c:after=["../stack-a"]`,
			},
			want: runExpected{
				Status:      defaultErrExitStatus,
				StderrRegex: dag.ErrCycleDetected.Error(),
			},
		},
		{
			name: "1 after 4 after 20 after 1 - fails",
			layout: []string{
				`s:1:after=["../2", "../3", "../4", "../5", "../6", "../7"]`,
				`s:2:after=["../12", "../13", "../14", "../15", "../16"]`,
				`s:3:after=["../2", "../4"]`,
				`s:4:after=["../6", "../20"]`,
				`s:5`,
				`s:6`,
				`s:7`,
				`s:8`,
				`s:9`,
				`s:10`,
				`s:11`,
				`s:12`,
				`s:13`,
				`s:14`,
				`s:15`,
				`s:16`,
				`s:17`,
				`s:18`,
				`s:19`,
				`s:20:after=["../10", "../1"]`,
			},
			want: runExpected{
				Status:      defaultErrExitStatus,
				StderrRegex: dag.ErrCycleDetected.Error(),
			},
		},
		{
			name: `stack-z after (stack-b, stack-c, stack-d)
				   stack-a after stack-c
				   stack-b before stack-a`,
			layout: []string{
				`s:stack-z:after=["../stack-b", "../stack-c", "../stack-d"]`,
				`s:stack-a:after=["../stack-c"]`,
				`s:stack-b:before=["../stack-a"]`,
				`s:stack-c`,
				`s:stack-d`,
			},
			want: runExpected{
				Stdout: `stack-b
stack-c
stack-a
stack-d
stack-z
`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := sandbox.New(t)
			s.BuildTree(tc.layout)

			cli := newCLI(t, s.RootDir())
			assertRunResult(t, cli.run("plan", "run-order"), tc.want)
		})
	}
}

func TestRunWants(t *testing.T) {
	type testcase struct {
		name   string
		layout []string
		wd     string
		want   runExpected
	}

	for _, tc := range []testcase{
		{
			/* this works but gives a warning */
			name: "stack-a wants stack-a",
			layout: []string{
				`s:stack-a:wants=["/stack-a"]`,
			},
			want: runExpected{
				Stdout: "stack-a\n",
			},
		},
		{
			name: "stack-a wants stack-b",
			layout: []string{
				`s:stack-a:wants=["/stack-b"]`,
				`s:stack-b`,
			},
			want: runExpected{
				Stdout: "stack-a\nstack-b\n",
			},
		},
		{
			name: "stack-b wants stack-a (same ordering)",
			layout: []string{
				`s:stack-b:wants=["/stack-a"]`,
				`s:stack-a`,
			},
			want: runExpected{
				Stdout: "stack-a\nstack-b\n",
			},
		},
		{
			name: "stack-a wants stack-b (from inside stack-a)",
			layout: []string{
				`s:stack-a:wants=["/stack-b"]`,
				`s:stack-b`,
			},
			wd: "/stack-a",
			want: runExpected{
				Stdout: "stack-a\nstack-b\n",
			},
		},
		{
			name: "stack-a wants stack-b (from inside stack-b)",
			layout: []string{
				`s:stack-a:wants=["/stack-b"]`,
				`s:stack-b`,
			},
			wd: "/stack-b",
			want: runExpected{
				Stdout: "stack-b\n",
			},
		},
		{
			name: "stack-b wants stack-a (same ordering) (from inside stack-b)",
			layout: []string{
				`s:stack-b:wants=["/stack-a"]`,
				`s:stack-a`,
			},
			wd: "/stack-b",
			want: runExpected{
				Stdout: "stack-a\nstack-b\n",
			},
		},
		{
			name: "stack-b wants (stack-a, stack-c) (from inside stack-b)",
			layout: []string{
				`s:stack-b:wants=["/stack-a", "/stack-c"]`,
				`s:stack-a`,
				`s:stack-c`,
			},
			wd: "/stack-b",
			want: runExpected{
				Stdout: "stack-a\nstack-b\nstack-c\n",
			},
		},
		{
			name: "stack-b wants (stack-a, stack-c), stack-c wants stack-a (from inside stack-b)",
			layout: []string{
				`s:stack-b:wants=["/stack-a", "/stack-c"]`,
				`s:stack-a`,
				`s:stack-c:wants=["/stack-a"]`,
			},
			wd: "/stack-b",
			want: runExpected{
				Stdout: "stack-a\nstack-b\nstack-c\n",
			},
		},
		{
			name: `stack-a wants (stack-b, stack-c) and stack-b wants (stack-d, stack-e)
					(from inside stack-a) - recursive`,
			layout: []string{
				`s:stack-a:wants=["/stack-b", "/stack-c"]`,
				`s:stack-b:wants=["/stack-d", "/stack-e"]`,
				`s:stack-c`,
				`s:stack-d`,
				`s:stack-e`,
			},
			wd: "/stack-a",
			want: runExpected{
				Stdout: "stack-a\nstack-b\nstack-c\nstack-d\nstack-e\n",
			},
		},
		{
			name: `stack-a wants (stack-b, stack-c) and stack-b wants (stack-d, stack-e)
					(from inside stack-b) - not recursive`,
			layout: []string{
				`s:stack-a:wants=["/stack-b", "/stack-c"]`,
				`s:stack-b:wants=["/stack-d", "/stack-e"]`,
				`s:stack-c`,
				`s:stack-d`,
				`s:stack-e`,
			},
			wd: "/stack-b",
			want: runExpected{
				Stdout: "stack-b\nstack-d\nstack-e\n",
			},
		},
		{
			name: `	stack-a wants (stack-b, stack-c)
					stack-b wants (stack-d, stack-e)
					stack-e wants (stack-a, stack-z)
					(from inside stack-b) - recursive, *circular*
					must pull all stacks`,
			layout: []string{
				`s:stack-a:wants=["/stack-b", "/stack-c"]`,
				`s:stack-b:wants=["/stack-d", "/stack-e"]`,
				`s:stack-c`,
				`s:stack-d`,
				`s:stack-e:wants=["/stack-a", "/stack-z"]`,
				`s:stack-z`,
			},
			wd: "/stack-b",
			want: runExpected{
				Stdout: "stack-a\nstack-b\nstack-c\nstack-d\nstack-e\nstack-z\n",
			},
		},
		{
			name: `wants+order - stack-a after stack-b / stack-d before stack-a
	* stack-a wants (stack-b, stack-c)
	* stack-b wants (stack-d, stack-e)
	* stack-e wants (stack-a, stack-z) (from inside stack-b) - recursive, *circular*`,
			layout: []string{
				`s:stack-a:wants=["/stack-b", "/stack-c"];after=["/stack-b"]`,
				`s:stack-b:wants=["/stack-d", "/stack-e"]`,
				`s:stack-c`,
				`s:stack-d:before=["/stack-a"]`,
				`s:stack-e:wants=["/stack-a", "/stack-z"]`,
				`s:stack-z`,
			},
			wd: "/stack-b",
			want: runExpected{
				Stdout: "stack-b\nstack-d\nstack-a\nstack-c\nstack-e\nstack-z\n",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := sandbox.New(t)
			s.BuildTree(tc.layout)

			cli := newCLI(t, filepath.Join(s.RootDir(), tc.wd))
			assertRunResult(t, cli.run("plan", "run-order"), tc.want)

			// required because `terramate run` requires a clean repo.
			git := s.Git()
			git.CommitAll("everything")

			// TODO(i4k): not portable
			assertRunResult(t, cli.run("run", "sh", "-c", "pwd | xargs basename"), tc.want)
		})
	}
}

func TestRunOrderNotChangedStackIgnored(t *testing.T) {
	const (
		mainTfFileName = "main.tf"
		mainTfContents = "# change is the eternal truth of the universe"
	)

	s := sandbox.New(t)

	// stack must run after stack2 but stack2 didn't change.

	stack2 := s.CreateStack("stack2")

	stack := s.CreateStack("stack")
	stackMainTf := stack.CreateFile(mainTfFileName, "# some code")
	stackConfig, err := hcl.NewConfig(stack.Path(), terramate.DefaultVersionConstraint())
	assert.NoError(t, err)
	stackConfig.Stack = &hcl.Stack{
		After: []string{project.PrjAbsPath(s.RootDir(), stack2.Path())},
	}
	stack.WriteConfig(stackConfig)

	git := s.Git()
	git.CommitAll("first commit")
	git.Push("main")
	git.CheckoutNew("change-stack")

	stackMainTf.Write(mainTfContents)
	git.CommitAll("stack changed")

	cli := newCLI(t, s.RootDir())

	wantList := stack.RelPath() + "\n"
	assertRunResult(t, cli.run("stacks", "list", "--changed"), runExpected{Stdout: wantList})

	cat := test.LookPath(t, "cat")
	wantRun := mainTfContents

	assertRunResult(t, cli.run(
		"run",
		"--changed",
		cat,
		mainTfFileName,
	), runExpected{Stdout: wantRun})

	wantRun = mainTfContents

	cli = newCLI(t, stack.Path())
	assertRunResult(t, cli.run(
		"run",
		"--changed",
		cat,
		mainTfFileName,
	), runExpected{Stdout: wantRun})

	cli = newCLI(t, stack2.Path())
	assertRunResult(t, cli.run(
		"run",
		"--changed",
		cat,
		mainTfFileName,
	), runExpected{})
}

func TestRunOrderAllChangedStacksExecuted(t *testing.T) {
	const (
		mainTfFileName = "main.tf"
		mainTfContents = "# change is the eternal truth of the universe"
	)

	// stack2 must run after stack and both changed.

	s := sandbox.New(t)

	stack2 := s.CreateStack("stack2")
	stack2MainTf := stack2.CreateFile(mainTfFileName, "# some code")

	stack := s.CreateStack("stack")
	stackMainTf := stack.CreateFile(mainTfFileName, "# some code")
	stackConfig, err := hcl.NewConfig(stack.Path(), terramate.DefaultVersionConstraint())
	assert.NoError(t, err)

	stackConfig.Stack = &hcl.Stack{
		After: []string{project.PrjAbsPath(s.RootDir(), stack2.Path())},
	}
	stack.WriteConfig(stackConfig)

	git := s.Git()
	git.CommitAll("first commit")
	git.Push("main")
	git.CheckoutNew("change-stack")

	stackMainTf.Write(mainTfContents)
	stack2MainTf.Write(mainTfContents)
	git.CommitAll("stack changed")

	cli := newCLI(t, s.RootDir())

	wantList := stack.RelPath() + "\n" + stack2.RelPath() + "\n"
	assertRunResult(t, cli.run("stacks", "list", "--changed"), runExpected{Stdout: wantList})

	cat := test.LookPath(t, "cat")
	wantRun := fmt.Sprintf(
		"%s%s",
		mainTfContents,
		mainTfContents,
	)

	assertRunResult(t, cli.run(
		"run",
		"--changed",
		cat,
		mainTfFileName,
	), runExpected{Stdout: wantRun})
}

func TestRunFailIfDirtyRepo(t *testing.T) {
	const (
		mainTfFileName = "main.tf"
		mainTfContents = "# some code"
	)

	s := sandbox.New(t)

	stack := s.CreateStack("stack")
	stack.CreateFile(mainTfFileName, mainTfContents)

	git := s.Git()
	git.CommitAll("first commit")
	git.Push("main")
	git.CheckoutNew("change-stack")

	untracked := stack.CreateFile("untracked-file.txt", `# something`)

	cli := newCLI(t, s.RootDir())
	cat := test.LookPath(t, "cat")

	assertRunResult(t, cli.run(
		"run",
		"--changed",
		cat,
		mainTfFileName,
	), runExpected{
		Status:      defaultErrExitStatus,
		StderrRegex: terramate.ErrDirtyRepo.Error(),
	})

	assertRunResult(t, cli.run(
		"run",
		cat,
		mainTfFileName,
	), runExpected{
		Status:      defaultErrExitStatus,
		StderrRegex: terramate.ErrDirtyRepo.Error(),
	})

	git.Add(untracked.Path())
	git.Commit("commit untracked")

	// everything commited, repo is clean
	assertRunResult(t, cli.run(
		"run",
		"--changed",
		cat,
		mainTfFileName,
	), runExpected{Stdout: mainTfContents})

	// change file, no commit
	untracked.Write("# changed")
	assertRunResult(t, cli.run(
		"run",
		cat,
		mainTfFileName,
	), runExpected{
		Status:      defaultErrExitStatus,
		StderrRegex: terramate.ErrDirtyRepo.Error(),
	})
}

func TestRunFailIfStackGeneratedCodeIsOutdated(t *testing.T) {
	const (
		testFilename   = "test.txt"
		contentsStack1 = "stack-1 file"
		contentsStack2 = "stack-2 file"
	)
	s := sandbox.New(t)

	stack1 := s.CreateStack("stacks/stack-1")
	stack2 := s.CreateStack("stacks/stack-2")

	stack1.CreateFile(testFilename, contentsStack1)
	stack2.CreateFile(testFilename, contentsStack2)

	git := s.Git()
	git.CommitAll("first commit")
	git.Push("main")

	tmcli := newCLI(t, s.RootDir())
	cat := test.LookPath(t, "cat")

	assertRunResult(t, tmcli.run("run", cat, testFilename), runExpected{
		Stdout: contentsStack1 + contentsStack2,
	})

	stack1.CreateConfig(`
		stack {}
		export_as_locals {
		  test = terramate.path
		}
	`)

	git.CheckoutNew("adding-stack1-config")
	git.CommitAll("adding stack-1 config")

	assertRunResult(t, tmcli.run("run", cat, testFilename), runExpected{
		Status:      defaultErrExitStatus,
		StderrRegex: cli.ErrOutdatedGenCodeDetected.Error(),
	})

	assertRunResult(t, tmcli.run("run", "--changed", cat, testFilename), runExpected{
		Status:      defaultErrExitStatus,
		StderrRegex: cli.ErrOutdatedGenCodeDetected.Error(),
	})

	// Check that if inside cwd it should work
	// Ignoring the other stack that has outdated code
	tmcli = newCLI(t, stack2.Path())

	assertRunResult(t, tmcli.run("run", cat, testFilename), runExpected{
		Stdout: contentsStack2,
	})
}

func TestRunLogsUserCommand(t *testing.T) {
	s := sandbox.New(t)

	stack := s.CreateStack("stack")
	testfile := stack.CreateFile("test", "")

	git := s.Git()
	git.CommitAll("first commit")
	git.Push("main")

	cli := newCLIWithLogLevel(t, s.RootDir(), "info")
	assertRunResult(t, cli.run("run", "cat", testfile.Path()), runExpected{
		StderrRegex: `cmd="cat /`,
	})
}