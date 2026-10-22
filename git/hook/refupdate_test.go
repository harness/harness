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

package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/sha"

	"github.com/stretchr/testify/require"
)

const (
	testBranchRef  = "refs/heads/test"
	testBranchRefA = "refs/heads/a"
	testBranchRefB = "refs/heads/b"
)

func TestRefUpdaterOptions(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}

	tests := []struct {
		name       string
		withHooks  bool
		noUpdate   bool
		wantEvents []string
		wantRef    bool
	}{
		{
			name:       "no options",
			wantEvents: nil,
			wantRef:    true,
		},
		{
			name:       "with hooks",
			withHooks:  true,
			wantEvents: []string{"pre", "post"},
			wantRef:    true,
		},
		{
			name:       "with hooks and no update",
			withHooks:  true,
			noUpdate:   true,
			wantEvents: []string{"pre", "post"},
			wantRef:    false,
		},
		{
			name:       "no update",
			noUpdate:   true,
			wantEvents: nil,
			wantRef:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repoPath, commitSHA := newBareRepoWithCommit(t)
			client := &recordingClient{}

			var opts []func(*RefUpdater) error
			if test.withHooks {
				opts = append(opts, WithHooks(&recordingClientFactory{client: client}, nil))
			}
			if test.noUpdate {
				opts = append(opts, WithNoUpdate)
			}

			u, err := CreateRefUpdater(repoPath, opts...)
			require.NoError(t, err)

			err = u.DoOne(context.Background(), testBranchRef, sha.Nil, commitSHA)
			require.NoError(t, err)

			require.Equal(t, test.wantEvents, client.events)

			refs := runGit(t, repoPath, "for-each-ref", "--format=%(refname)")
			if test.wantRef {
				require.Equal(t, testBranchRef, refs)
			} else {
				require.Empty(t, refs)
			}
		})
	}
}

func TestCreateRefUpdaterRequiresRepoPath(t *testing.T) {
	_, err := CreateRefUpdater("")
	require.Error(t, err)
	require.True(t, errors.IsInternal(err))
}

func TestCreateRefUpdaterWithHooksRequiresClientFactory(t *testing.T) {
	_, err := CreateRefUpdater(t.TempDir(), WithHooks(nil, nil))
	require.Error(t, err)
}

func TestCreateRefUpdaterWithHooksReturnsClientFactoryError(t *testing.T) {
	_, err := CreateRefUpdater(t.TempDir(), WithHooks(erroringClientFactory{}, nil))
	require.ErrorContains(t, err, "client factory failure")
}

func TestCreateRefUpdaterWithHooksRejectsNilClient(t *testing.T) {
	_, err := CreateRefUpdater(t.TempDir(), WithHooks(nilClientFactory{}, nil))
	require.Error(t, err)
	require.True(t, errors.IsInternal(err))
}

func TestCreateRefUpdaterWithHooksPassesEnvVars(t *testing.T) {
	factory := &recordingClientFactory{client: &recordingClient{}}
	envVars := map[string]string{"KEY": "value"}

	_, err := CreateRefUpdater(t.TempDir(), WithHooks(factory, envVars))
	require.NoError(t, err)
	require.Equal(t, envVars, factory.envVars)
}

func TestCreateRefUpdaterReturnsOptionError(t *testing.T) {
	optErr := errors.New("option failure")

	_, err := CreateRefUpdater(t.TempDir(), func(*RefUpdater) error { return optErr })
	require.ErrorIs(t, err, optErr)
}

func TestRefUpdaterDo(t *testing.T) {
	const ref = testBranchRef

	tests := []struct {
		name        string
		existing    map[string]string
		changes     []refChange
		wantRefs    map[string]string
		wantApplied []refChange
	}{
		{
			name:        "create",
			existing:    map[string]string{},
			changes:     []refChange{{ref, "nil", "c1"}},
			wantRefs:    map[string]string{ref: "c1"},
			wantApplied: []refChange{{ref, "nil", "c1"}},
		},
		{
			name:        "create with unknown old value",
			existing:    map[string]string{},
			changes:     []refChange{{ref, "", "c1"}},
			wantRefs:    map[string]string{ref: "c1"},
			wantApplied: []refChange{{ref, "nil", "c1"}},
		},
		{
			name:        "update",
			existing:    map[string]string{ref: "c1"},
			changes:     []refChange{{ref, "c1", "c2"}},
			wantRefs:    map[string]string{ref: "c2"},
			wantApplied: []refChange{{ref, "c1", "c2"}},
		},
		{
			name:        "update with unknown old value",
			existing:    map[string]string{ref: "c1"},
			changes:     []refChange{{ref, "", "c2"}},
			wantRefs:    map[string]string{ref: "c2"},
			wantApplied: []refChange{{ref, "c1", "c2"}},
		},
		{
			name:        "delete",
			existing:    map[string]string{ref: "c1"},
			changes:     []refChange{{ref, "c1", "nil"}},
			wantRefs:    map[string]string{},
			wantApplied: []refChange{{ref, "c1", "nil"}},
		},
		{
			name:        "delete with empty new value",
			existing:    map[string]string{ref: "c1"},
			changes:     []refChange{{ref, "c1", ""}},
			wantRefs:    map[string]string{},
			wantApplied: []refChange{{ref, "c1", "nil"}},
		},
		{
			name:        "unchanged ref is skipped",
			existing:    map[string]string{ref: "c1"},
			changes:     []refChange{{ref, "", "c1"}},
			wantRefs:    map[string]string{ref: "c1"},
			wantApplied: []refChange{},
		},
		{
			name:     "multiple refs are applied sorted by name",
			existing: map[string]string{},
			changes: []refChange{
				{testBranchRefB, "nil", "c2"},
				{testBranchRefA, "nil", "c1"},
			},
			wantRefs: map[string]string{testBranchRefA: "c1", testBranchRefB: "c2"},
			wantApplied: []refChange{
				{testBranchRefA, "nil", "c1"},
				{testBranchRefB, "nil", "c2"},
			},
		},
		{
			name:     "create, update and delete at once",
			existing: map[string]string{testBranchRefA: "c1", testBranchRefB: "c1", "refs/heads/c": "c1"},
			changes: []refChange{
				{testBranchRefA, "c1", "c2"},
				{testBranchRefB, "c1", "nil"},
				{"refs/heads/d", "nil", "c1"},
			},
			wantRefs: map[string]string{testBranchRefA: "c2", "refs/heads/c": "c1", "refs/heads/d": "c1"},
			wantApplied: []refChange{
				{testBranchRefA, "c1", "c2"},
				{testBranchRefB, "c1", "nil"},
				{"refs/heads/d", "nil", "c1"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := newTestRepo(t)
			r.setRefs(test.existing)

			client := &recordingClient{}
			u, err := CreateRefUpdater(r.path, WithHooks(&recordingClientFactory{client: client}, nil))
			require.NoError(t, err)

			err = u.Do(context.Background(), r.updates(test.changes))
			require.NoError(t, err)

			require.Equal(t, test.wantRefs, r.refs())
			require.Equal(t, test.wantApplied, r.changes(u.Refs()))

			if len(test.wantApplied) == 0 {
				// hooks are not called when there is nothing to update
				require.Empty(t, client.events)
				return
			}

			require.Equal(t, []string{"pre", "post"}, client.events)
			require.Equal(t, test.wantApplied, r.changes(client.preInputs[0].RefUpdates))
			require.Equal(t, test.wantApplied, r.changes(client.postInputs[0].RefUpdates))
		})
	}
}

func TestRefUpdaterDoWithoutRefs(t *testing.T) {
	r := newTestRepo(t)
	client := &recordingClient{}

	u, err := CreateRefUpdater(r.path, WithHooks(&recordingClientFactory{client: client}, nil))
	require.NoError(t, err)

	require.NoError(t, u.Do(context.Background(), nil))
	require.Empty(t, u.Refs())
	require.Empty(t, client.events)
	require.Equal(t, map[string]string{}, r.refs())

	// the updater is finished
	require.Error(t, u.Post(context.Background()))
}

func TestRefUpdaterAlternateObjectDirs(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	client := &recordingClient{}

	u, err := CreateRefUpdater(r.path, WithHooks(&recordingClientFactory{client: client}, nil))
	require.NoError(t, err)

	require.NoError(t, u.Init(ctx, []ReferenceUpdate{{Ref: testBranchRef, Old: sha.Nil, New: r.c1}}))
	require.NoError(t, u.Pre(ctx, "/pre/dir1", "/pre/dir2"))
	require.NoError(t, u.UpdateRef(ctx))
	require.NoError(t, u.Post(ctx, "/post/dir"))

	require.Equal(t, []string{"/pre/dir1", "/pre/dir2"}, client.preInputs[0].Environment.AlternateObjectDirs)
	require.Equal(t, []string{"/post/dir"}, client.postInputs[0].Environment.AlternateObjectDirs)
}

func TestRefUpdaterPreReceiveBlocked(t *testing.T) {
	const blockMsg = "branch is protected"

	tests := []struct {
		name       string
		violations json.RawMessage
	}{
		{name: "without rule violations"},
		{name: "with rule violations", violations: json.RawMessage(`[{"rule":"protect-main"}]`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			r := newTestRepo(t)
			r.setRefs(map[string]string{testBranchRef: "c1"})

			msg := blockMsg
			client := &recordingClient{preOutput: Output{Error: &msg, RuleViolations: test.violations}}

			u, err := CreateRefUpdater(r.path, WithHooks(&recordingClientFactory{client: client}, nil))
			require.NoError(t, err)

			err = u.DoOne(ctx, testBranchRef, r.c1, r.c2)
			require.Error(t, err)
			require.True(t, errors.IsUnprocessableEntity(err), "unexpected error: %v", err)
			require.ErrorContains(t, err, blockMsg)

			details, hasViolations := errors.Details(err)[RuleViolationsErrorDetailsKey]
			if test.violations == nil {
				require.False(t, hasViolations)
			} else {
				require.Equal(t, test.violations, details)
			}

			// the reference is untouched and post-receive hook is not called
			require.Equal(t, map[string]string{testBranchRef: "c1"}, r.refs())
			require.Equal(t, []string{"pre"}, client.events)

			// a blocked update can't be forced by calling the following steps
			require.Error(t, u.UpdateRef(ctx))
			require.Error(t, u.SkipUpdateRef(ctx))
			require.Error(t, u.Post(ctx))
			require.Equal(t, map[string]string{testBranchRef: "c1"}, r.refs())
			require.Equal(t, []string{"pre"}, client.events)
		})
	}
}

func TestRefUpdaterPreReceiveCallFailure(t *testing.T) {
	r := newTestRepo(t)
	callErr := errors.New("pre-receive is down")
	client := &recordingClient{preErr: callErr}

	u, err := CreateRefUpdater(r.path, WithHooks(&recordingClientFactory{client: client}, nil))
	require.NoError(t, err)

	err = u.DoOne(context.Background(), testBranchRef, sha.Nil, r.c1)
	require.ErrorIs(t, err, callErr)

	require.Equal(t, map[string]string{}, r.refs())
	require.Equal(t, []string{"pre"}, client.events)
}

func TestRefUpdaterPostReceiveFailure(t *testing.T) {
	postMsg := "post-receive failed"
	callErr := errors.New("post-receive is down")

	tests := []struct {
		name    string
		client  *recordingClient
		wantErr string
	}{
		{
			name:    "call failure",
			client:  &recordingClient{postErr: callErr},
			wantErr: callErr.Error(),
		},
		{
			name:    "output error",
			client:  &recordingClient{postOutput: Output{Error: &postMsg}},
			wantErr: postMsg,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			r := newTestRepo(t)

			u, err := CreateRefUpdater(r.path, WithHooks(&recordingClientFactory{client: test.client}, nil))
			require.NoError(t, err)

			err = u.DoOne(ctx, testBranchRef, sha.Nil, r.c1)
			require.ErrorContains(t, err, test.wantErr)

			// post-receive runs after the update, so the reference has been changed already
			require.Equal(t, map[string]string{testBranchRef: "c1"}, r.refs())
			require.Equal(t, []string{"pre", "post"}, test.client.events)
		})
	}
}

func TestRefUpdaterUpdateRefFailure(t *testing.T) {
	tests := []struct {
		name     string
		existing map[string]string
		changes  []refChange
		check    func(error) bool
		wantRefs map[string]string
	}{
		{
			name:     "old value mismatch",
			existing: map[string]string{testBranchRef: "c2"},
			changes:  []refChange{{testBranchRef, "c1", "c3"}},
			check:    errors.IsPreconditionFailed,
			wantRefs: map[string]string{testBranchRef: "c2"},
		},
		{
			name:     "create of existing reference",
			existing: map[string]string{testBranchRef: "c1"},
			changes:  []refChange{{testBranchRef, "nil", "c2"}},
			check:    errors.IsConflict,
			wantRefs: map[string]string{testBranchRef: "c1"},
		},
		{
			name:     "delete with old value mismatch",
			existing: map[string]string{testBranchRef: "c2"},
			changes:  []refChange{{testBranchRef, "c1", "nil"}},
			check:    errors.IsPreconditionFailed,
			wantRefs: map[string]string{testBranchRef: "c2"},
		},
		{
			name:     "invalid reference name",
			existing: map[string]string{},
			changes:  []refChange{{"refs/heads/invalid..name", "nil", "c1"}},
			check:    errors.IsInternal,
			wantRefs: map[string]string{},
		},
		{
			name:     "all references are left untouched if one of them fails",
			existing: map[string]string{testBranchRefB: "c2"},
			changes: []refChange{
				{testBranchRefA, "nil", "c1"},
				{testBranchRefB, "c1", "c3"},
			},
			check:    errors.IsPreconditionFailed,
			wantRefs: map[string]string{testBranchRefB: "c2"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			r := newTestRepo(t)
			r.setRefs(test.existing)

			client := &recordingClient{}
			u, err := CreateRefUpdater(r.path, WithHooks(&recordingClientFactory{client: client}, nil))
			require.NoError(t, err)

			err = u.Do(ctx, r.updates(test.changes))
			require.Error(t, err)
			require.True(t, test.check(err), "unexpected error: %v", err)

			require.Equal(t, test.wantRefs, r.refs())

			// post-receive hook must not be called for an update that didn't happen
			require.Equal(t, []string{"pre"}, client.events)
			require.Error(t, u.Post(ctx))
			require.Equal(t, []string{"pre"}, client.events)
		})
	}
}

func TestRefUpdaterSkipUpdateRef(t *testing.T) {
	tests := []struct {
		name       string
		withHooks  bool
		wantEvents []string
	}{
		{name: "with hooks", withHooks: true, wantEvents: []string{"pre", "post"}},
		{name: "without hooks", withHooks: false, wantEvents: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			r := newTestRepo(t)
			client := &recordingClient{}

			var opts []func(*RefUpdater) error
			if test.withHooks {
				opts = append(opts, WithHooks(&recordingClientFactory{client: client}, nil))
			}

			u, err := CreateRefUpdater(r.path, opts...)
			require.NoError(t, err)

			require.NoError(t, u.Init(ctx, []ReferenceUpdate{{Ref: testBranchRef, Old: sha.Nil, New: r.c1}}))
			require.NoError(t, u.Pre(ctx))
			require.NoError(t, u.SkipUpdateRef(ctx))

			// the reference isn't touched, but the hook still has to be called
			require.Equal(t, map[string]string{}, r.refs())

			require.NoError(t, u.Post(ctx))
			require.Equal(t, map[string]string{}, r.refs())
			require.Equal(t, test.wantEvents, client.events)

			// the updater is finished
			require.Error(t, u.Post(ctx))
			require.Error(t, u.SkipUpdateRef(ctx))
		})
	}
}

func TestRefUpdaterOperationOrder(t *testing.T) {
	const (
		stepInit = "init"
		stepPre  = "pre"
		stepUpd  = "update"
		stepSkip = "skip"
		stepPost = "post"
	)

	tests := []struct {
		name string
		// steps are the steps to execute. All steps except the last one are expected to succeed,
		// and the last one is expected to fail because of the invalid operation order.
		steps []string
	}{
		{"pre before init", []string{stepPre}},
		{"update before init", []string{stepUpd}},
		{"skip before init", []string{stepSkip}},
		{"post before init", []string{stepPost}},
		{"init twice", []string{stepInit, stepInit}},
		{"update before pre", []string{stepInit, stepUpd}},
		{"skip before pre", []string{stepInit, stepSkip}},
		{"post before pre", []string{stepInit, stepPost}},
		{"init after pre", []string{stepInit, stepPre, stepInit}},
		{"pre twice", []string{stepInit, stepPre, stepPre}},
		{"post before update", []string{stepInit, stepPre, stepPost}},
		{"update twice", []string{stepInit, stepPre, stepUpd, stepUpd}},
		{"skip after update", []string{stepInit, stepPre, stepUpd, stepSkip}},
		{"update after skip", []string{stepInit, stepPre, stepSkip, stepUpd}},
		{"skip twice", []string{stepInit, stepPre, stepSkip, stepSkip}},
		{"pre after update", []string{stepInit, stepPre, stepUpd, stepPre}},
		{"init after post", []string{stepInit, stepPre, stepUpd, stepPost, stepInit}},
		{"pre after post", []string{stepInit, stepPre, stepUpd, stepPost, stepPre}},
		{"update after post", []string{stepInit, stepPre, stepUpd, stepPost, stepUpd}},
		{"skip after post", []string{stepInit, stepPre, stepUpd, stepPost, stepSkip}},
		{"post twice", []string{stepInit, stepPre, stepUpd, stepPost, stepPost}},
	}

	// The repository is shared between the subtests to make them faster. Each subtest uses its own reference.
	r := newTestRepo(t)

	var n int

	for _, withHooks := range []bool{false, true} {
		for _, test := range tests {
			name := test.name + map[bool]string{false: " without hooks", true: " with hooks"}[withHooks]
			ref := fmt.Sprintf("refs/heads/order-%d", n)
			n++

			t.Run(name, func(t *testing.T) {
				ctx := context.Background()

				var opts []func(*RefUpdater) error
				if withHooks {
					opts = append(opts, WithHooks(&recordingClientFactory{client: &recordingClient{}}, nil))
				}

				u, err := CreateRefUpdater(r.path, opts...)
				require.NoError(t, err)

				run := func(step string) error {
					switch step {
					case stepInit:
						return u.Init(ctx, []ReferenceUpdate{{Ref: ref, Old: sha.Nil, New: r.c1}})
					case stepPre:
						return u.Pre(ctx)
					case stepUpd:
						return u.UpdateRef(ctx)
					case stepSkip:
						return u.SkipUpdateRef(ctx)
					case stepPost:
						return u.Post(ctx)
					}
					t.Fatalf("unknown step %q", step)
					return nil
				}

				last := len(test.steps) - 1
				for _, step := range test.steps[:last] {
					require.NoErrorf(t, run(step), "step %q should succeed", step)
				}

				err = run(test.steps[last])
				require.Errorf(t, err, "step %q should fail", test.steps[last])
				require.ErrorContains(t, err, "invalid operation order")
			})
		}
	}
}

func TestRefUpdaterNilReceiver(t *testing.T) {
	ctx := context.Background()

	var u *RefUpdater

	require.NoError(t, u.Pre(ctx))
	require.NoError(t, u.UpdateRef(ctx))
	require.NoError(t, u.SkipUpdateRef(ctx))
	require.NoError(t, u.Post(ctx))
	require.Nil(t, u.Refs())
}

// recordingClient is a Client that records the hooks that have been called and the data they were called with.
// The returned values of the hooks can be set with the preOutput, preErr, postOutput and postErr fields.
type recordingClient struct {
	events     []string
	preInputs  []PreReceiveInput
	postInputs []PostReceiveInput

	preOutput  Output
	preErr     error
	postOutput Output
	postErr    error
}

func (c *recordingClient) PreReceive(_ context.Context, in PreReceiveInput) (Output, error) {
	c.events = append(c.events, "pre")
	c.preInputs = append(c.preInputs, in)
	return c.preOutput, c.preErr
}

func (c *recordingClient) Update(context.Context, UpdateInput) (Output, error) {
	c.events = append(c.events, "update")
	return Output{}, nil
}

func (c *recordingClient) PostReceive(_ context.Context, in PostReceiveInput) (Output, error) {
	c.events = append(c.events, "post")
	c.postInputs = append(c.postInputs, in)
	return c.postOutput, c.postErr
}

// recordingClientFactory is a ClientFactory that returns the provided client and records the provided env variables.
type recordingClientFactory struct {
	client  *recordingClient
	envVars map[string]string
}

func (f *recordingClientFactory) NewClient(envVars map[string]string) (Client, error) {
	f.envVars = envVars
	return f.client, nil
}

// erroringClientFactory is a ClientFactory that always fails to create a Client.
type erroringClientFactory struct{}

func (erroringClientFactory) NewClient(map[string]string) (Client, error) {
	return nil, errors.New("client factory failure")
}

// nilClientFactory is a misbehaving ClientFactory that returns neither a Client nor an error.
type nilClientFactory struct{}

func (nilClientFactory) NewClient(map[string]string) (Client, error) {
	return nil, nil //nolint:nilnil
}

// refChange is a ReferenceUpdate that uses the names of the commits of a testRepo instead of SHAs.
// Besides the commit names (c1, c2, c3), the old and new values can be "nil" (the null SHA, "0000...")
// and an empty string (no SHA, which the RefUpdater resolves by itself).
type refChange struct {
	ref, old, new string
}

// testRepo is a bare git repository with three unreferenced commits:
// c1 is the root commit, c2 and c3 are two different children of c1.
type testRepo struct {
	t          *testing.T
	path       string
	c1, c2, c3 sha.SHA
}

func newTestRepo(t *testing.T) *testRepo {
	t.Helper()

	path := newBareRepo(t)
	emptyTree := runGit(t, path, "mktree")
	c1 := commitTree(t, path, emptyTree, "c1")

	return &testRepo{
		t:    t,
		path: path,
		c1:   c1,
		c2:   commitTree(t, path, emptyTree, "c2", c1),
		c3:   commitTree(t, path, emptyTree, "c3", c1),
	}
}

func (r *testRepo) sha(name string) sha.SHA {
	switch name {
	case "":
		return sha.None
	case "nil":
		return sha.Nil
	case "c1":
		return r.c1
	case "c2":
		return r.c2
	case "c3":
		return r.c3
	}

	r.t.Fatalf("unknown commit name %q", name)

	return sha.None
}

func (r *testRepo) name(s sha.SHA) string {
	switch s {
	case sha.None:
		return ""
	case sha.Nil:
		return "nil"
	case r.c1:
		return "c1"
	case r.c2:
		return "c2"
	case r.c3:
		return "c3"
	}

	return s.String()
}

// setRefs creates the provided references. The map values are the names of the commits.
func (r *testRepo) setRefs(refs map[string]string) {
	r.t.Helper()

	for ref, name := range refs {
		runGit(r.t, r.path, "update-ref", ref, r.sha(name).String())
	}
}

// refs returns all references in the repository. The map values are the names of the commits.
func (r *testRepo) refs() map[string]string {
	r.t.Helper()

	refs := map[string]string{}

	out := runGit(r.t, r.path, "for-each-ref", "--format=%(refname) %(objectname)")
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}

		ref, objectSHA, ok := strings.Cut(line, " ")
		require.True(r.t, ok, "unexpected for-each-ref output: %q", line)

		refs[ref] = r.name(sha.Must(objectSHA))
	}

	return refs
}

func (r *testRepo) updates(changes []refChange) []ReferenceUpdate {
	updates := make([]ReferenceUpdate, len(changes))
	for i, change := range changes {
		updates[i] = ReferenceUpdate{Ref: change.ref, Old: r.sha(change.old), New: r.sha(change.new)}
	}

	return updates
}

func (r *testRepo) changes(updates []ReferenceUpdate) []refChange {
	changes := make([]refChange, len(updates))
	for i, update := range updates {
		changes[i] = refChange{ref: update.Ref, old: r.name(update.Old), new: r.name(update.New)}
	}

	return changes
}

// newBareRepo creates an empty bare git repository.
func newBareRepo(t *testing.T) string {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}

	repoPath := t.TempDir()
	runGit(t, repoPath, "init", "--bare", "--quiet")

	return repoPath
}

// newBareRepoWithCommit creates a bare git repository with a single unreferenced commit.
func newBareRepoWithCommit(t *testing.T) (string, sha.SHA) {
	t.Helper()

	repoPath := newBareRepo(t)
	emptyTree := runGit(t, repoPath, "mktree")

	return repoPath, commitTree(t, repoPath, emptyTree, "test")
}

// commitTree creates a commit with the provided tree and parents.
func commitTree(t *testing.T, repoPath, tree, message string, parents ...sha.SHA) sha.SHA {
	t.Helper()

	args := []string{"commit-tree", "-m", message}
	for _, parent := range parents {
		args = append(args, "-p", parent.String())
	}
	args = append(args, tree)

	return sha.Must(runGit(t, repoPath, args...))
}

func runGit(t *testing.T, repoPath string, args ...string) string {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = repoPath
	cmd.Env = append(os.Environ(),
		// ignore the git configuration of the machine that runs the tests (for example commit signing)
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
	)

	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %s: %s", strings.Join(args, " "), out)

	return strings.TrimSpace(string(out))
}
