// Copyright 2026 Harness, Inc.
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

package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gtxRequireGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath(GitExecutable); err != nil {
		t.Skip("git binary not available on PATH")
	}
}

func gtxNewRepo(ctx context.Context, t *testing.T) string {
	t.Helper()
	gtxRequireGit(t)

	dir := t.TempDir()
	require.NoError(t, New("init", WithFlag("-q", "--bare")).Run(ctx, WithDir(dir)))

	return dir
}

func TestGtxBuilder_Args(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		flags       []string
		args        []string
		postSepArgs []string
		want        []string
		wantErr     bool
	}{
		{
			name: "no flags no end of options", command: "log",
			args: []string{"main"}, want: []string{"main"},
		},
		{
			name: "flags add end of options", command: "log",
			flags: []string{"--oneline"}, args: []string{"main"},
			want: []string{"--oneline", "--end-of-options", "main"},
		},
		{
			name: "command without end of options support", command: "rev-parse",
			flags: []string{"--verify"}, args: []string{"main"},
			want: []string{"--verify", "main"},
		},
		{
			name: "post separator args", command: "log",
			args: []string{"main"}, postSepArgs: []string{"-dash.txt", "b.txt"},
			want: []string{"main", "--", "-dash.txt", "b.txt"},
		},
		{
			name: "post separator args after end of options need no extra separator", command: "log",
			flags: []string{"--oneline"}, postSepArgs: []string{"a.txt"},
			want: []string{"--oneline", "--end-of-options", "a.txt"},
		},
		{
			name: "post separator args only", command: "log",
			postSepArgs: []string{"a.txt"},
			want:        []string{"a.txt"},
		},
		{name: "positional arg with dash", command: "log", args: []string{"--output=x"}, wantErr: true},
		{
			name: "rev-list accepts pseudo revisions", command: "rev-list",
			args: []string{"--all", "--not", "main"}, want: []string{"--all", "--not", "main"},
		},
		{name: "rev-list rejects other options", command: "rev-list", args: []string{"--all", "--output=x"}, wantErr: true},
		{
			name: "archive accepts compression level", command: "archive",
			args: []string{"-9", "HEAD"}, want: []string{"-9", "HEAD"},
		},
		{name: "archive rejects flag smuggling", command: "archive", args: []string{"-9", "--output=x"}, wantErr: true},
		{name: "archive rejects non numeric dash", command: "archive", args: []string{"-x"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := descriptions[test.command].args(test.flags, test.args, test.postSepArgs)
			if test.wantErr {
				require.ErrorIs(t, err, ErrInvalidArg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestGtxBuilder_SupportsEndOfOptions(t *testing.T) {
	assert.True(t, descriptions["log"].supportsEndOfOptions())
	assert.False(t, descriptions["blame"].supportsEndOfOptions())
	assert.False(t, descriptions["update-index"].supportsEndOfOptions())
}

func TestGtxThreadsConfigValue(t *testing.T) {
	tests := []struct {
		cpus int
		want string
	}{
		{cpus: 0, want: "1"},
		{cpus: 1, want: "1"},
		{cpus: 2, want: "1"},
		{cpus: 3, want: "1"},
		{cpus: 4, want: "2"},
		{cpus: 8, want: "3"},
		{cpus: 16, want: "4"},
		{cpus: 64, want: "6"},
	}

	for _, test := range tests {
		assert.Equal(t, test.want, threadsConfigValue(test.cpus), "cpus=%d", test.cpus)
	}
}

func TestGtxCommand_MakeArgs(t *testing.T) {
	t.Run("globals action flags args", func(t *testing.T) {
		c := New("remote",
			WithGlobal("--no-pager"),
			WithAction("set-url"),
			WithFlag("--push"),
			WithArg("origin", "https://example.com/repo.git"),
		)

		args, err := c.makeArgs()
		require.NoError(t, err)
		assert.Equal(t, []string{"--no-pager", "remote", "set-url", "--push", "origin", "https://example.com/repo.git"}, args)
	})

	t.Run("unknown command", func(t *testing.T) {
		_, err := New("gtx-unknown").makeArgs()
		require.ErrorIs(t, err, ErrInvalidArg)
	})

	t.Run("invalid actions", func(t *testing.T) {
		for _, action := range []string{"-bad", "bad action", "bad;rm", "--flag", "a/b"} {
			_, err := New("remote", WithAction(action)).makeArgs()
			require.ErrorIs(t, err, ErrInvalidArg, action)
		}
	})

	t.Run("invalid arg", func(t *testing.T) {
		_, err := New("log", WithArg("-x")).makeArgs()
		require.ErrorIs(t, err, ErrInvalidArg)
	})

	t.Run("pack options are added", func(t *testing.T) {
		c := New("pack-objects", WithConfig("gtx.key", "value"))
		_, err := c.makeArgs()
		require.NoError(t, err)

		assert.Equal(t, "3", c.Envs["GIT_CONFIG_COUNT"])
		assert.Equal(t, "gtx.key", c.Envs["GIT_CONFIG_KEY_0"])
		assert.Equal(t, "pack.windowMemory", c.Envs["GIT_CONFIG_KEY_1"])
		assert.Equal(t, "100m", c.Envs["GIT_CONFIG_VALUE_1"])
		assert.Equal(t, "pack.threads", c.Envs["GIT_CONFIG_KEY_2"])
		assert.Equal(t, threadsConfigValue(runtime.NumCPU()), c.Envs["GIT_CONFIG_VALUE_2"])
	})

	t.Run("upload pack options are added", func(t *testing.T) {
		c := New("upload-pack")
		_, err := c.makeArgs()
		require.NoError(t, err)

		assert.Equal(t, "4", c.Envs["GIT_CONFIG_COUNT"])
		keys := map[string]string{}
		for i := range 4 {
			keys[c.Envs["GIT_CONFIG_KEY_"+string(rune('0'+i))]] = c.Envs["GIT_CONFIG_VALUE_"+string(rune('0'+i))]
		}
		assert.Equal(t, "true", keys["uploadpack.allowFilter"])
		assert.Equal(t, "true", keys["uploadpack.allowAnySHA1InWant"])
		assert.Equal(t, "100m", keys["pack.windowMemory"])
		assert.Contains(t, keys, "pack.threads")
	})
}

func TestGtxCmdOptions(t *testing.T) {
	date := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)

	tests := []struct {
		name string
		opt  CmdOptionFunc
		want func(t *testing.T, c *Command)
	}{
		{
			name: "global",
			opt:  WithGlobal("--a", "--b"),
			want: func(t *testing.T, c *Command) { assert.Equal(t, []string{"--a", "--b"}, c.Globals) },
		},
		{
			name: "action",
			opt:  WithAction("get-url"),
			want: func(t *testing.T, c *Command) { assert.Equal(t, "get-url", c.Action) },
		},
		{
			name: "post sep arg",
			opt:  WithPostSepArg("a", "b"),
			want: func(t *testing.T, c *Command) { assert.Equal(t, []string{"a", "b"}, c.PostSepArgs) },
		},
		{
			name: "env",
			opt:  WithEnv("K1", "V1", "K2", "V2"),
			want: func(t *testing.T, c *Command) { assert.Equal(t, Envs{"K1": "V1", "K2": "V2"}, c.Envs) },
		},
		{
			name: "committer",
			opt:  WithCommitter("Gtx C", "c@example.com"),
			want: func(t *testing.T, c *Command) {
				assert.Equal(t, Envs{GitCommitterName: "Gtx C", GitCommitterEmail: "c@example.com"}, c.Envs)
			},
		},
		{
			name: "committer and date",
			opt:  WithCommitterAndDate("Gtx C", "c@example.com", date),
			want: func(t *testing.T, c *Command) {
				assert.Equal(t, "2024-05-06T07:08:09Z", c.Envs[GitCommitterDate])
				assert.Equal(t, "Gtx C", c.Envs[GitCommitterName])
				assert.Equal(t, "c@example.com", c.Envs[GitCommitterEmail])
			},
		},
		{
			name: "author",
			opt:  WithAuthor("Gtx A", "a@example.com"),
			want: func(t *testing.T, c *Command) {
				assert.Equal(t, Envs{GitAuthorName: "Gtx A", GitAuthorEmail: "a@example.com"}, c.Envs)
			},
		},
		{
			name: "author and date",
			opt:  WithAuthorAndDate("Gtx A", "a@example.com", date),
			want: func(t *testing.T, c *Command) {
				assert.Equal(t, "2024-05-06T07:08:09Z", c.Envs[GitAuthorDate])
				assert.Equal(t, "Gtx A", c.Envs[GitAuthorName])
				assert.Equal(t, "a@example.com", c.Envs[GitAuthorEmail])
			},
		},
		{
			name: "alternate object dirs",
			opt:  WithAlternateObjectDirs("/one", "/two"),
			want: func(t *testing.T, c *Command) { assert.Equal(t, "/one:/two", c.Envs[GitAlternateObjectDirs]) },
		},
		{
			name: "no alternate object dirs",
			opt:  WithAlternateObjectDirs(),
			want: func(t *testing.T, c *Command) { assert.NotContains(t, c.Envs, GitAlternateObjectDirs) },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.want(t, New("log", test.opt))
		})
	}

	t.Run("config counter", func(t *testing.T) {
		c := New("log", WithConfig("a.b", "1"), WithConfig("c.d", "2"))
		assert.Equal(t, Envs{
			"GIT_CONFIG_COUNT":   "2",
			"GIT_CONFIG_KEY_0":   "a.b",
			"GIT_CONFIG_VALUE_0": "1",
			"GIT_CONFIG_KEY_1":   "c.d",
			"GIT_CONFIG_VALUE_1": "2",
		}, c.Envs)
	})

	t.Run("add returns the command", func(t *testing.T) {
		c := New("log")
		assert.Same(t, c, c.Add(WithArg("x")))
		assert.Equal(t, []string{"x"}, c.Args)
	})
}

func TestGtxRunOptions(t *testing.T) {
	o := &RunOption{}
	stdin, stdout, stderr := strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}

	for _, f := range []RunOptionFunc{
		WithDir("/some/dir"), WithStdin(stdin), WithStdout(stdout), WithStderr(stderr),
		WithEnvs("A=1"), WithEnvs("B=2", "C=3"),
	} {
		f(o)
	}

	assert.Equal(t, "/some/dir", o.Dir)
	assert.Same(t, stdin, o.Stdin)
	assert.Same(t, stdout, o.Stdout)
	assert.Same(t, stderr, o.Stderr)
	assert.Equal(t, []string{"A=1", "B=2", "C=3"}, o.Envs)
}

func TestGtxEnvs_Args(t *testing.T) {
	assert.Empty(t, Envs{}.Args())
	assert.ElementsMatch(t, []string{"A=1", "B=2"}, Envs{"A": "1", "B": "2"}.Args())
}

func TestGtxCommand_Clone(t *testing.T) {
	orig := New("log",
		WithAction("act"),
		WithFlag("--oneline"),
		WithArg("main"),
		WithEnv("K", "V"),
		WithConfig("a.b", "c"),
	)

	clone := orig.Clone()
	assert.Equal(t, orig.Name, clone.Name)
	assert.Equal(t, orig.Action, clone.Action)
	assert.Equal(t, orig.Flags, clone.Flags)
	assert.Equal(t, orig.Args, clone.Args)
	assert.Equal(t, orig.Envs, clone.Envs)
	assert.Equal(t, orig.configEnvCounter, clone.configEnvCounter)

	// the clone is independent of the original.
	clone.Add(WithFlag("--graph"), WithArg("dev"), WithEnv("K2", "V2"))
	assert.Equal(t, []string{"--oneline"}, orig.Flags)
	assert.Equal(t, []string{"main"}, orig.Args)
	assert.NotContains(t, orig.Envs, "K2")
	assert.Equal(t, []string{"--oneline", "--graph"}, clone.Flags)
	assert.Equal(t, []string{"main", "dev"}, clone.Args)
}

func TestGtxCommand_Run(t *testing.T) {
	ctx := context.Background()
	repo := gtxNewRepo(ctx, t)

	t.Run("stdin to stdout", func(t *testing.T) {
		var out bytes.Buffer
		err := New("hash-object", WithFlag("--stdin")).Run(ctx,
			WithDir(repo), WithStdin(strings.NewReader("hello\n")), WithStdout(&out))
		require.NoError(t, err)
		assert.Equal(t, "ce013625030ba8dba906f756967f9e9ca394464a", strings.TrimSpace(out.String()))
	})

	t.Run("discards stdout by default", func(t *testing.T) {
		err := New("hash-object", WithFlag("--stdin")).Run(ctx, WithDir(repo), WithStdin(strings.NewReader("x")))
		require.NoError(t, err)
	})

	t.Run("environment variables", func(t *testing.T) {
		var out bytes.Buffer
		err := New("var",
			WithArg("GIT_AUTHOR_IDENT"),
			WithAuthorAndDate("Gtx A", "a@example.com", time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)),
		).Run(ctx, WithDir(repo), WithStdout(&out), WithEnvs("HOME="+t.TempDir()))
		// git var isn't part of the curated list of commands.
		require.ErrorIs(t, err, ErrInvalidArg)

		out.Reset()
		err = New("commit-tree",
			WithArg("4b825dc642cb6eb9a060e54bf8d69288fbee4904"),
			WithAuthorAndDate("Gtx A", "a@example.com", time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)),
			WithCommitterAndDate("Gtx C", "c@example.com", time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)),
			WithConfig("gtx.dummy", "1"),
		).Run(ctx, WithDir(repo), WithStdout(&out), WithStdin(strings.NewReader("msg")))
		require.NoError(t, err)

		var raw bytes.Buffer
		require.NoError(t, New("cat-file", WithFlag("-p"), WithArg(strings.TrimSpace(out.String()))).
			Run(ctx, WithDir(repo), WithStdout(&raw)))
		assert.Contains(t, raw.String(), "author Gtx A <a@example.com> 1704164645 +0000")
		assert.Contains(t, raw.String(), "committer Gtx C <c@example.com> 1704164645 +0000")
	})

	t.Run("custom stderr writer", func(t *testing.T) {
		var stderr bytes.Buffer
		err := New("cat-file", WithFlag("-p"), WithArg("1234567890123456789012345678901234567890")).
			Run(ctx, WithDir(repo), WithStderr(&stderr))
		require.Error(t, err)

		cmdErr := AsError(err)
		require.NotNil(t, cmdErr)
		assert.Empty(t, cmdErr.StdErr, "stderr isn't captured in the error if the caller provides a writer")
		assert.NotEmpty(t, stderr.String())
	})

	t.Run("invalid command", func(t *testing.T) {
		err := New("gtx-unknown").Run(ctx, WithDir(repo))
		require.ErrorIs(t, err, ErrInvalidArg)
	})

	t.Run("start failure", func(t *testing.T) {
		err := New("version").Run(ctx, WithDir(filepath.Join(repo, "does-not-exist")))
		require.Error(t, err)
		assert.Nil(t, AsError(err))
	})

	t.Run("canceled while running", func(t *testing.T) {
		cctx, cancel := context.WithCancel(ctx)
		// an *os.File is handed to the child process directly, so waiting for it doesn't depend on a copy goroutine.
		pr, pw, err := os.Pipe()
		require.NoError(t, err)
		t.Cleanup(func() { _ = pw.Close(); _ = pr.Close() })

		done := make(chan error, 1)
		go func() {
			done <- New("hash-object", WithFlag("--stdin")).Run(cctx, WithDir(repo), WithStdin(pr))
		}()

		time.Sleep(100 * time.Millisecond)
		cancel()

		// git might be a wrapper that leaves its child alive after being killed, which holds the pipes open.
		time.AfterFunc(500*time.Millisecond, func() { _ = pw.Close() })

		select {
		case err := <-done:
			require.ErrorIs(t, err, context.Canceled)
		case <-time.After(10 * time.Second):
			t.Fatal("command did not return after the context was canceled")
		}
	})
}

func TestGtxError_FromRealGit(t *testing.T) {
	ctx := context.Background()
	repo := gtxNewRepo(ctx, t)
	unknownSHA := "1234567890123456789012345678901234567890"

	t.Run("exit code", func(t *testing.T) {
		err := New("cat-file", WithFlag("-e"), WithArg(unknownSHA)).Run(ctx, WithDir(repo))
		require.Error(t, err)

		cmdErr := AsError(err)
		require.NotNil(t, cmdErr)
		assert.NotZero(t, cmdErr.ExitCode())
		assert.True(t, cmdErr.IsExitCode(cmdErr.ExitCode()))
		assert.False(t, cmdErr.IsExitCode(0))
		require.NotNil(t, cmdErr.Unwrap())

		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
	})

	t.Run("ambiguous argument", func(t *testing.T) {
		err := New("log", WithArg("gtx-unknown-branch")).Run(ctx, WithDir(repo))
		require.Error(t, err)

		cmdErr := AsError(err)
		require.NotNil(t, cmdErr)
		assert.True(t, cmdErr.IsAmbiguousArgErr(), cmdErr.Error())
		assert.True(t, cmdErr.IsUnknownRevision())
		assert.False(t, cmdErr.IsBadObject())
	})

	t.Run("bad object", func(t *testing.T) {
		err := New("show", WithFlag("--no-patch"), WithArg(unknownSHA)).Run(ctx, WithDir(repo))
		require.Error(t, err)

		cmdErr := AsError(err)
		require.NotNil(t, cmdErr)
		assert.True(t, cmdErr.IsBadObject(), cmdErr.Error())
		assert.Contains(t, cmdErr.Error(), unknownSHA)
	})

	t.Run("error with and without stderr", func(t *testing.T) {
		base := errors.New("exit status 1")

		assert.Equal(t, "exit status 1", NewError(base, nil).Error())
		assert.Equal(t, "exit status 1: boom", NewError(base, []byte("boom")).Error())
		assert.Zero(t, NewError(base, nil).ExitCode(), "no exit error wrapped")
		assert.True(t, NewError(base, nil).IsExitCode(0))
	})
}

func TestGtxError_Predicates(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		check  func(e *Error) bool
		want   bool
	}{
		{name: "invalid ref", stderr: "fatal: not a valid ref", check: (*Error).IsInvalidRefErr, want: true},
		{name: "valid ref", stderr: "fatal: other", check: (*Error).IsInvalidRefErr},
		{name: "ambiguous", stderr: "fatal: ambiguous argument 'x'", check: (*Error).IsAmbiguousArgErr, want: true},
		{name: "not ambiguous", stderr: "fatal: other", check: (*Error).IsAmbiguousArgErr},
		{name: "bad object", stderr: "fatal: bad object abc", check: (*Error).IsBadObject, want: true},
		{name: "not a bad object", stderr: "fatal: other", check: (*Error).IsBadObject},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := NewError(errors.New("exit status 128"), []byte(test.stderr))
			assert.Equal(t, test.want, test.check(e))
		})
	}

	t.Run("as error", func(t *testing.T) {
		inner := NewError(errors.New("x"), nil)
		assert.Same(t, inner, AsError(inner))
		assert.Same(t, inner, AsError(errors.Join(errors.New("outer"), inner)))
		assert.Nil(t, AsError(errors.New("plain")))
		assert.Nil(t, AsError(nil))
	})
}

func TestGtxGitExecutableNotFound(t *testing.T) {
	// the global is restored when the test ends: no other test runs in parallel in this package.
	orig := GitExecutable
	t.Cleanup(func() { GitExecutable = orig })
	GitExecutable = filepath.Join(t.TempDir(), "gtx-no-such-git")

	err := New("version").Run(context.Background())
	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist) || errors.Is(err, exec.ErrNotFound), err.Error())
}
