// Copyright 2023 Harness, Inc.
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

// Package sharedrepotest holds conformance tests for the implementations of
// sharedrepo.Runner and sharedrepo.ReadOnlyRunner interfaces.
//
// Call RunnerConformance and ReadOnlyRunnerConformance from a test of the package
// that provides the implementation:
//
//	func TestMyRunner(t *testing.T) {
//		sharedrepotest.RunnerConformance(t, &MyRunner{})
//	}
package sharedrepotest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/git/sharedrepo"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRef = "refs/tags/conformance"

// runFunc runs the function in a shared repository without a reference update.
type runFunc func(
	ctx context.Context,
	tmpDir, repoPath string,
	fn func(s *sharedrepo.SharedRepo) error,
	alternates ...string,
) error

// RunnerConformance checks that the provided sharedrepo.Runner follows the contract of the interface.
func RunnerConformance(t *testing.T, runner sharedrepo.Runner) {
	requireGit(t)

	// Both runners must provide the same view of the repository, regardless of the reference updater.
	run := func(
		ctx context.Context,
		tmpDir, repoPath string,
		fn func(s *sharedrepo.SharedRepo) error,
		alternates ...string,
	) error {
		return runner.Run(ctx, nil, tmpDir, repoPath, fn, alternates...)
	}

	t.Run("sees objects of the original repository and of the alternates", func(t *testing.T) {
		testSeesObjects(t, run)
	})

	t.Run("without ref updater new objects are still added to the original repository", func(t *testing.T) {
		ctx := t.Context()
		repoPath := newBareRepo(t)

		var blob sha.SHA
		var dir string
		err := runner.Run(ctx, nil, t.TempDir(), repoPath, func(s *sharedrepo.SharedRepo) error {
			var err error
			dir = s.Directory()
			blob, err = s.WriteGitObject(ctx, strings.NewReader("new object\n"))
			return err
		})
		require.NoError(t, err)

		assert.True(t, hasObject(t, repoPath, blob))
		assert.NoDirExists(t, dir, "shared repository must be removed")
	})

	t.Run("failed function leaves the original repository untouched", func(t *testing.T) {
		ctx := t.Context()
		repoPath := newBareRepo(t)

		errFn := errors.New("function failed")
		hooks := &fakeHooks{}
		updater := newRefUpdater(t, hooks, repoPath)

		var blob sha.SHA
		var dir string
		err := runner.Run(ctx, updater, t.TempDir(), repoPath, func(s *sharedrepo.SharedRepo) error {
			var err error
			dir = s.Directory()
			blob, err = s.WriteGitObject(ctx, strings.NewReader("new object\n"))
			if err != nil {
				return err
			}
			if err := updater.Init(ctx, createRef(blob)); err != nil {
				return err
			}
			return errFn
		})
		require.ErrorIs(t, err, errFn)

		assert.False(t, hasObject(t, repoPath, blob))
		assert.Empty(t, refValue(t, repoPath, testRef))
		assert.Empty(t, hooks.events, "no hook should have been called")
		assert.NoDirExists(t, dir, "shared repository must be removed")
	})

	t.Run("updates the reference between the pre-receive and post-receive hooks", func(t *testing.T) {
		ctx := t.Context()
		repoPath := newBareRepo(t)

		var blob sha.SHA
		var dir string

		var preRefUpdates []hook.ReferenceUpdate
		var preSeesObject bool
		var postRefValue string
		var postHasObject bool

		hooks := &fakeHooks{}
		hooks.onPre = func(in hook.PreReceiveInput) {
			preRefUpdates = in.RefUpdates
			// the new objects must be accessible to the hook through the provided alternates
			preSeesObject = hasObject(t, repoPath, blob,
				"GIT_ALTERNATE_OBJECT_DIRECTORIES="+strings.Join(in.Environment.AlternateObjectDirs, string(os.PathListSeparator)))
		}
		hooks.onPost = func(hook.PostReceiveInput) {
			postRefValue = refValue(t, repoPath, testRef)
			postHasObject = hasObject(t, repoPath, blob)
		}

		updater := newRefUpdater(t, hooks, repoPath)

		err := runner.Run(ctx, updater, t.TempDir(), repoPath, func(s *sharedrepo.SharedRepo) error {
			var err error
			dir = s.Directory()
			blob, err = s.WriteGitObject(ctx, strings.NewReader("new object\n"))
			if err != nil {
				return err
			}
			return updater.Init(ctx, createRef(blob))
		})
		require.NoError(t, err)

		assert.Equal(t, []string{"pre", "post"}, hooks.events)
		assert.Equal(t, createRef(blob), preRefUpdates)
		assert.True(t, preSeesObject, "pre-receive hook must be able to read the new objects")
		assert.Equal(t, blob.String(), postRefValue, "post-receive hook must be called after the reference update")
		assert.True(t, postHasObject, "post-receive hook must be called after the objects are moved")

		assert.Equal(t, blob.String(), refValue(t, repoPath, testRef))
		assert.True(t, hasObject(t, repoPath, blob))
		assert.NoDirExists(t, dir, "shared repository must be removed")
	})

	t.Run("blocked by pre-receive hook leaves the original repository untouched", func(t *testing.T) {
		ctx := t.Context()
		repoPath := newBareRepo(t)

		hooks := &fakeHooks{preError: "blocked by the test"}
		updater := newRefUpdater(t, hooks, repoPath)

		var blob sha.SHA
		var dir string
		err := runner.Run(ctx, updater, t.TempDir(), repoPath, func(s *sharedrepo.SharedRepo) error {
			var err error
			dir = s.Directory()
			blob, err = s.WriteGitObject(ctx, strings.NewReader("new object\n"))
			if err != nil {
				return err
			}
			return updater.Init(ctx, createRef(blob))
		})
		require.Error(t, err)
		assert.True(t, errors.IsUnprocessableEntity(err), "the error of the hook must be preserved: %v", err)

		assert.Equal(t, []string{"pre"}, hooks.events, "post-receive hook must not be called")
		assert.Empty(t, refValue(t, repoPath, testRef))
		assert.False(t, hasObject(t, repoPath, blob))
		assert.NoDirExists(t, dir, "shared repository must be removed")
	})
}

// ReadOnlyRunnerConformance checks that the provided sharedrepo.ReadOnlyRunner follows the contract of the interface.
func ReadOnlyRunnerConformance(t *testing.T, runner sharedrepo.ReadOnlyRunner) {
	requireGit(t)

	t.Run("sees objects of the original repository and of the alternates", func(t *testing.T) {
		testSeesObjects(t, runner.Run)
	})

	t.Run("discards the changes", func(t *testing.T) {
		ctx := t.Context()
		repoPath := newBareRepo(t)

		var blob sha.SHA
		var dir string
		err := runner.Run(ctx, t.TempDir(), repoPath, func(s *sharedrepo.SharedRepo) error {
			var err error
			dir = s.Directory()
			blob, err = s.WriteGitObject(ctx, strings.NewReader("new object\n"))
			return err
		})
		require.NoError(t, err)

		assert.False(t, hasObject(t, repoPath, blob))
		assert.NoDirExists(t, dir, "shared repository must be removed")
	})

	t.Run("returns the error of the function", func(t *testing.T) {
		ctx := t.Context()
		repoPath := newBareRepo(t)

		errFn := errors.New("function failed")

		var dir string
		err := runner.Run(ctx, t.TempDir(), repoPath, func(s *sharedrepo.SharedRepo) error {
			dir = s.Directory()
			return errFn
		})
		require.ErrorIs(t, err, errFn)

		assert.NoDirExists(t, dir, "shared repository must be removed")
	})
}

// testSeesObjects checks that the shared repository can access the objects
// of the original repository and of the additional alternates.
func testSeesObjects(t *testing.T, run runFunc) {
	ctx := t.Context()

	repoPath := newBareRepo(t)
	repoBlob := writeBlob(t, repoPath, "object of the original repository\n")

	alternatePath := newBareRepo(t)
	alternateBlob := writeBlob(t, alternatePath, "object of the alternate\n")

	var seesRepoBlob, seesAlternateBlob bool
	err := run(ctx, t.TempDir(), repoPath, func(s *sharedrepo.SharedRepo) error {
		seesRepoBlob = hasObject(t, s.Directory(), repoBlob)
		seesAlternateBlob = hasObject(t, s.Directory(), alternateBlob)
		return nil
	}, filepath.Join(alternatePath, "objects"))
	require.NoError(t, err)

	assert.True(t, seesRepoBlob, "object of the original repository must be accessible")
	assert.True(t, seesAlternateBlob, "object of the alternate must be accessible")
}

// fakeHooks is a hook.ClientFactory and hook.Client that records the calls it receives.
type fakeHooks struct {
	events   []string
	preError string
	onPre    func(hook.PreReceiveInput)
	onPost   func(hook.PostReceiveInput)
}

func (f *fakeHooks) NewClient(map[string]string) (hook.Client, error) {
	return f, nil
}

func (f *fakeHooks) PreReceive(_ context.Context, in hook.PreReceiveInput) (hook.Output, error) {
	f.events = append(f.events, "pre")
	if f.onPre != nil {
		f.onPre(in)
	}
	if f.preError != "" {
		return hook.Output{Error: &f.preError}, nil
	}
	return hook.Output{}, nil
}

func (f *fakeHooks) Update(context.Context, hook.UpdateInput) (hook.Output, error) {
	f.events = append(f.events, "update")
	return hook.Output{}, nil
}

func (f *fakeHooks) PostReceive(_ context.Context, in hook.PostReceiveInput) (hook.Output, error) {
	f.events = append(f.events, "post")
	if f.onPost != nil {
		f.onPost(in)
	}
	return hook.Output{}, nil
}

func newRefUpdater(t *testing.T, hooks *fakeHooks, repoPath string) *hook.RefUpdater {
	t.Helper()

	updater, err := hook.CreateRefUpdater(hooks, nil, repoPath)
	require.NoError(t, err)

	return updater
}

func createRef(objectSHA sha.SHA) []hook.ReferenceUpdate {
	return []hook.ReferenceUpdate{{Ref: testRef, Old: sha.Nil, New: objectSHA}}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}
}

// newBareRepo creates an empty bare git repository.
func newBareRepo(t *testing.T) string {
	t.Helper()

	repoPath := t.TempDir()
	runGit(t, repoPath, "", nil, "init", "--bare")

	return repoPath
}

// writeBlob writes a blob directly to the object database of the repository.
func writeBlob(t *testing.T, repoPath, content string) sha.SHA {
	t.Helper()

	objectSHA, err := sha.New(runGit(t, repoPath, content, nil, "hash-object", "-w", "--stdin"))
	require.NoError(t, err)

	return objectSHA
}

// hasObject checks whether the object is available in the repository.
func hasObject(t *testing.T, repoPath string, objectSHA sha.SHA, env ...string) bool {
	t.Helper()

	_, err := git(t, repoPath, "", env, "cat-file", "-e", objectSHA.String())
	if _, ok := errors.AsType[*exec.ExitError](err); ok {
		return false
	}

	require.NoError(t, err)

	return true
}

// refValue returns the object the reference points to or an empty string if the reference doesn't exist.
//
//nolint:unparam
func refValue(t *testing.T, repoPath, ref string) string {
	t.Helper()

	out, err := git(t, repoPath, "", nil, "rev-parse", "--verify", "--quiet", ref)
	if _, ok := errors.AsType[*exec.ExitError](err); ok {
		return ""
	}

	require.NoError(t, err)

	return out
}

// runGit runs a git command and fails the test if the command fails.
func runGit(t *testing.T, repoPath, stdin string, env []string, args ...string) string {
	t.Helper()

	out, err := git(t, repoPath, stdin, env, args...)
	require.NoErrorf(t, err, "git %s failed", strings.Join(args, " "))

	return out
}

// git runs a git command and returns its trimmed standard output.
// It is the only place that starts a process. The arguments are always provided by the tests.
func git(t *testing.T, repoPath, stdin string, env []string, args ...string) (string, error) {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "git", args...) //nolint:gosec
	cmd.Dir = repoPath
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = strings.NewReader(stdin)

	out, err := cmd.Output()

	return strings.TrimSpace(string(out)), err
}
