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
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/sha"

	"github.com/rs/zerolog/log"
)

// CreateRefUpdater creates new RefUpdater object. Options WithHooks and WithNoUpdate can be provided.
func CreateRefUpdater(
	repoPath string,
	opts ...func(updater *RefUpdater) error,
) (*RefUpdater, error) {
	if repoPath == "" {
		return nil, errors.Internal(nil, "repo path can't be empty")
	}

	u := &RefUpdater{
		state:    stateInit,
		repoPath: repoPath,
		refs:     nil,
	}

	for _, opt := range opts {
		if err := opt(u); err != nil {
			return nil, fmt.Errorf("failed to apply RefUpdater option: %w", err)
		}
	}

	return u, nil
}

// WithHooks is an option for CreateRefUpdater that enables git hooks calling (pre-receive and post-receive).
func WithHooks(hookClientFactory ClientFactory, envVars map[string]string) func(*RefUpdater) error {
	return func(u *RefUpdater) error {
		if hookClientFactory == nil {
			return errors.Internal(nil, "hook client factory can't be nil")
		}

		client, err := hookClientFactory.NewClient(envVars)
		if err != nil {
			return fmt.Errorf("failed to create hook.Client: %w", err)
		}

		if client == nil {
			return errors.Internal(nil, "hook client factory returned nil client")
		}

		u.hookClient = client

		return nil
	}
}

// WithNoUpdate is an option for CreateRefUpdater that disables the update of the reference in UpdateRef.
func WithNoUpdate(u *RefUpdater) error {
	u.noUpdate = true
	return nil
}

// RefUpdater is an entity that is responsible for update of a reference.
// It will call pre-receive hook prior to the update and post-receive hook after the update.
// It has a state machine to guarantee that methods are called in the correct order (Init, Pre, Update, Post).
type RefUpdater struct {
	state      refUpdaterState
	hookClient Client
	repoPath   string
	refs       []ReferenceUpdate
	noUpdate   bool
}

// refUpdaterState represents state of the ref updater internal state machine.
type refUpdaterState byte

func (t refUpdaterState) String() string {
	switch t {
	case stateInit:
		return "INIT"
	case statePre:
		return "PRE"
	case stateUpdate:
		return "UPDATE"
	case statePost:
		return "POST"
	case stateDone:
		return "DONE"
	}
	return "INVALID"
}

const (
	stateInit refUpdaterState = iota
	statePre
	stateUpdate
	statePost
	stateDone
)

// Do runs full ref update by executing all methods in the correct order.
// It discards pre- and post-receive output messages (except Error which is returned as an error).
func (u *RefUpdater) Do(ctx context.Context, refs []ReferenceUpdate) error {
	if err := u.Init(ctx, refs); err != nil {
		return fmt.Errorf("init failed: %w", err)
	}

	if _, err := u.Pre(ctx); err != nil {
		return fmt.Errorf("pre failed: %w", err)
	}

	if err := u.UpdateRef(ctx); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	if _, err := u.Post(ctx); err != nil {
		return fmt.Errorf("post failed: %w", err)
	}

	return nil
}

// DoOne runs full ref update of only one reference.
func (u *RefUpdater) DoOne(ctx context.Context, ref string, oldValue, newValue sha.SHA) error {
	return u.Do(ctx, []ReferenceUpdate{
		{
			Ref: ref,
			Old: oldValue,
			New: newValue,
		},
	})
}

func (u *RefUpdater) Init(ctx context.Context, refs []ReferenceUpdate) error {
	if u.state != stateInit {
		return fmt.Errorf("invalid operation order: init old requires state=%s, current state=%s",
			stateInit, u.state)
	}

	u.refs = make([]ReferenceUpdate, 0, len(refs))
	for _, ref := range refs {
		oldValue := ref.Old
		newValue := ref.New

		var oldValueKnown bool

		if oldValue.IsEmpty() {
			// if no old value was provided, use current value (as required for hooks)
			val, err := u.getRef(ctx, ref.Ref)
			if errors.IsNotFound(err) { //nolint:gocritic
				oldValue = sha.Nil
			} else if err != nil {
				return fmt.Errorf("failed to get current value of reference %q: %w", ref.Ref, err)
			} else {
				oldValue = val
			}

			oldValueKnown = true
		}

		if newValue.IsEmpty() {
			// don't break existing interface - user calls with empty value to delete the ref.
			newValue = sha.Nil
		}

		if oldValueKnown && oldValue == newValue {
			// skip the unchanged refs
			continue
		}

		u.refs = append(u.refs, ReferenceUpdate{
			Ref: ref.Ref,
			Old: oldValue,
			New: newValue,
		})
	}

	sort.Slice(u.refs, func(i, j int) bool {
		return u.refs[i].Ref < u.refs[j].Ref
	})

	u.state = statePre

	return nil
}

// Pre runs the pre-receive git hook if hooks were enabled with WithHooks option.
//
// The returned Output holds the messages produced by the hook, and it should be inspected even if an error is returned.
// When the hook ran and rejected the update, the error is returned together with the hook's Output, because the
// messages explain the rejection and are meant to be shown to the user.
// The Output is zero if the hook wasn't called or if the call to it failed.
func (u *RefUpdater) Pre(ctx context.Context, alternateDirs ...string) (Output, error) {
	if u == nil {
		return Output{}, nil
	}

	if u.state != statePre {
		return Output{}, fmt.Errorf("invalid operation order: pre-receive hook requires state=%s, current state=%s",
			statePre, u.state)
	}

	if len(u.refs) == 0 || u.hookClient == nil {
		u.state = stateUpdate
		return Output{}, nil
	}

	out, err := u.hookClient.PreReceive(ctx, PreReceiveInput{
		RefUpdates: u.refs,
		Environment: Environment{
			AlternateObjectDirs: alternateDirs,
		},
	})
	if err != nil {
		return Output{}, fmt.Errorf("pre-receive call failed with: %w", err)
	}
	if out.Error != nil {
		log.Ctx(ctx).Debug().
			Str("err", *out.Error).
			Msgf("Pre-receive blocked ref update\nMessages\n%v", out.Messages)
		blockErr := errors.UnprocessableEntityf("pre-receive hook blocked reference update: %q", *out.Error)
		// Carry structured violations on the error so the API layer can surface them.
		if len(out.RuleViolations) > 0 {
			blockErr = blockErr.SetDetails(map[string]any{RuleViolationsErrorDetailsKey: out.RuleViolations})
		}
		return out, blockErr
	}

	u.state = stateUpdate

	return out, nil
}

// UpdateRef updates the git reference.
func (u *RefUpdater) UpdateRef(ctx context.Context) error {
	if u == nil {
		return nil
	}

	if u.state != stateUpdate {
		return fmt.Errorf("invalid operation order: ref update requires state=%s, current state=%s",
			stateUpdate, u.state)
	}

	if len(u.refs) == 0 || u.noUpdate {
		u.state = statePost
		return nil
	}

	input := bytes.NewBuffer(nil)
	for _, ref := range u.refs {
		switch {
		case ref.New.IsNil():
			_, _ = fmt.Fprintf(input, "delete %s\000%s\000", ref.Ref, ref.Old)
		case ref.Old.IsNil():
			_, _ = fmt.Fprintf(input, "create %s\000%s\000", ref.Ref, ref.New)
		default:
			_, _ = fmt.Fprintf(input, "update %s\000%s\000%s\000", ref.Ref, ref.New, ref.Old)
		}
	}

	input.WriteString("commit\000")

	cmd := command.New("update-ref", command.WithFlag("--stdin"), command.WithFlag("-z"))
	if err := cmd.Run(ctx, command.WithStdin(input), command.WithDir(u.repoPath)); err != nil {
		// Check for specific error conditions
		if cErr := command.AsError(err); cErr != nil {
			if cErr.IsSHAMismatchErr() {
				// Include the git error which contains the actual SHAs
				return errors.PreconditionFailedf(
					"ref update failed: SHA mismatch (optimistic locking failure): %s",
					cErr.Error(),
				)
			}
		}

		msg := err.Error()
		if strings.Contains(msg, "reference already exists") {
			return errors.Conflict("reference already exists")
		}

		return fmt.Errorf("update of references %v failed: %w", u.refs, err)
	}

	u.state = statePost

	return nil
}

// SkipUpdateRef skips the update of the git reference. The reference is left untouched, but the updater moves on to
// the state in which Post can be called, so the post-receive hook still runs.
func (u *RefUpdater) SkipUpdateRef(context.Context) error {
	if u == nil {
		return nil
	}

	if u.state != stateUpdate {
		return fmt.Errorf("invalid operation order: skip ref update requires state=%s, current state=%s",
			stateUpdate, u.state)
	}

	u.state = statePost
	return nil
}

// Post runs the post-receive git hook if hooks were enabled with WithHooks option.
//
// The returned Output holds the messages produced by the hook, and it should be inspected even if an error is returned.
// When the hook ran and reported an error, the error is returned together with the hook's Output, because the
// messages describe what went wrong and are meant to be shown to the user.
// The Output is zero if the hook wasn't called or if the call to it failed.
func (u *RefUpdater) Post(ctx context.Context, alternateDirs ...string) (Output, error) {
	if u == nil {
		return Output{}, nil
	}

	if u.state != statePost {
		return Output{}, fmt.Errorf("invalid operation order: post-receive hook requires state=%s, current state=%s",
			statePost, u.state)
	}

	if len(u.refs) == 0 || u.hookClient == nil {
		u.state = stateDone
		return Output{}, nil
	}

	out, err := u.hookClient.PostReceive(ctx, PostReceiveInput{
		RefUpdates: u.refs,
		Environment: Environment{
			AlternateObjectDirs: alternateDirs,
		},
	})
	if err != nil {
		return Output{}, fmt.Errorf("post-receive call failed with: %w", err)
	}
	if out.Error != nil {
		return out, fmt.Errorf("post-receive call returned error: %q", *out.Error)
	}

	u.state = stateDone

	return out, nil
}

func (u *RefUpdater) Refs() []ReferenceUpdate {
	if u == nil {
		return nil
	}

	return u.refs
}

func (u *RefUpdater) getRef(ctx context.Context, ref string) (sha.SHA, error) {
	cmd := command.New("show-ref",
		command.WithFlag("--verify"),
		command.WithFlag("-s"),
		command.WithArg(ref),
	)
	output := &bytes.Buffer{}
	err := cmd.Run(ctx, command.WithDir(u.repoPath), command.WithStdout(output))
	if cErr := command.AsError(err); cErr != nil && cErr.IsExitCode(128) && cErr.IsInvalidRefErr() {
		return sha.None, errors.NotFoundf("reference %q not found", ref)
	}

	if err != nil {
		return sha.None, err
	}

	return sha.New(output.String())
}
