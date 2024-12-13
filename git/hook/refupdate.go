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
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/sha"

	"github.com/rs/zerolog/log"
)

// CreateRefUpdater creates new RefUpdater object using the provided git hook ClientFactory.
func CreateRefUpdater(
	hookClientFactory ClientFactory,
	envVars map[string]string,
	repoPath string,
	ref string,
) (*RefUpdater, error) {
	if repoPath == "" {
		return nil, errors.Internal(nil, "repo path can't be empty")
	}

	client, err := hookClientFactory.NewClient(envVars)
	if err != nil {
		return nil, fmt.Errorf("failed to create hook.Client: %w", err)
	}

	return &RefUpdater{
		state:      stateInitOld,
		hookClient: client,
		envVars:    envVars,
		repoPath:   repoPath,
		ref:        ref,
		oldValue:   sha.None,
		newValue:   sha.None,
	}, nil
}

// RefUpdater is an entity that is responsible for update of a reference.
// It will call pre-receive hook prior to the update and post-receive hook after the update.
// It has a state machine to guarantee that methods are called in the correct order (Init, Pre, Update, Post).
type RefUpdater struct {
	state      refUpdaterState
	hookClient Client
	envVars    map[string]string
	repoPath   string
	ref        string
	oldValue   sha.SHA
	newValue   sha.SHA
}

// refUpdaterState represents state of the ref updater internal state machine.
type refUpdaterState byte

func (t refUpdaterState) String() string {
	switch t {
	case stateInitOld:
		return "INIT_OLD"
	case stateInitNew:
		return "INIT_NEW"
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
	stateInitOld refUpdaterState = iota
	stateInitNew
	statePre
	stateUpdate
	statePost
	stateDone
)

// Do runs full ref update by executing all methods in the correct order.
func (u *RefUpdater) Do(ctx context.Context, oldValue, newValue sha.SHA) error {
	if err := u.Init(ctx, oldValue, newValue); err != nil {
		return fmt.Errorf("init failed: %w", err)
	}

	if err := u.Pre(ctx); err != nil {
		return fmt.Errorf("pre failed: %w", err)
	}

	if err := u.UpdateRef(ctx); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	if err := u.Post(ctx); err != nil {
		return fmt.Errorf("post failed: %w", err)
	}

	return nil
}

func (u *RefUpdater) Init(ctx context.Context, oldValue, newValue sha.SHA) error {
	if err := u.InitOld(ctx, oldValue); err != nil {
		return fmt.Errorf("init old failed: %w", err)
	}
	if err := u.InitNew(ctx, newValue); err != nil {
		return fmt.Errorf("init new failed: %w", err)
	}

	return nil
}

func (u *RefUpdater) InitOld(ctx context.Context, oldValue sha.SHA) error {
	if u == nil {
		return nil
	}

	if u.state != stateInitOld {
		return fmt.Errorf("invalid operation order: init old requires state=%s, current state=%s",
			stateInitOld, u.state)
	}

	if oldValue.IsEmpty() {
		// if no old value was provided, use current value (as required for hooks)
		val, err := u.getRef(ctx)
		if errors.IsNotFound(err) { //nolint:gocritic
			oldValue = sha.Nil
		} else if err != nil {
			return fmt.Errorf("failed to get current value of reference: %w", err)
		} else {
			oldValue = val
		}
	}

	u.state = stateInitNew
	u.oldValue = oldValue

	return nil
}

func (u *RefUpdater) InitNew(_ context.Context, newValue sha.SHA) error {
	if u == nil {
		return nil
	}

	if u.state != stateInitNew {
		return fmt.Errorf("invalid operation order: init new requires state=%s, current state=%s",
			stateInitNew, u.state)
	}

	if newValue.IsEmpty() {
		// don't break existing interface - user calls with empty value to delete the ref.
		newValue = sha.Nil
	}

	u.state = statePre
	u.newValue = newValue

	return nil
}

// Pre runs the pre-receive git hook.
func (u *RefUpdater) Pre(ctx context.Context, alternateDirs ...string) error {
	if u.state != statePre {
		return fmt.Errorf("invalid operation order: pre-receive hook requires state=%s, current state=%s",
			statePre, u.state)
	}

	// fail in case someone tries to delete a reference that doesn't exist.
	if u.oldValue.IsEmpty() && u.newValue.IsNil() {
		return errors.NotFound("reference %q not found", u.ref)
	}

	if u.oldValue.IsNil() && u.newValue.IsNil() {
		return fmt.Errorf("provided values cannot be both empty")
	}

	out, err := u.hookClient.PreReceive(ctx, PreReceiveInput{
		RefUpdates: []ReferenceUpdate{
			{
				Ref: u.ref,
				Old: u.oldValue,
				New: u.newValue,
			},
		},
		Environment: Environment{
			AlternateObjectDirs: alternateDirs,
		},
	})
	if err != nil {
		return fmt.Errorf("pre-receive call failed with: %w", err)
	}
	if out.Error != nil {
		log.Ctx(ctx).Debug().
			Str("err", *out.Error).
			Msgf("Pre-receive blocked ref update\nMessages\n%v", out.Messages)
		return errors.PreconditionFailed("pre-receive hook blocked reference update: %q", *out.Error)
	}

	u.state = stateUpdate

	return nil
}

// UpdateRef updates the git reference.
func (u *RefUpdater) UpdateRef(ctx context.Context) error {
	if u.state != stateUpdate {
		return fmt.Errorf("invalid operation order: ref update requires state=%s, current state=%s",
			stateUpdate, u.state)
	}

	cmd := command.New("update-ref")
	if u.newValue.IsNil() {
		cmd.Add(command.WithFlag("-d", u.ref))
	} else {
		cmd.Add(command.WithArg(u.ref, u.newValue.String()))
	}

	cmd.Add(command.WithArg(u.oldValue.String()))

	if err := cmd.Run(ctx, command.WithDir(u.repoPath)); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "reference already exists") {
			return errors.Conflict("reference already exists")
		}

		return fmt.Errorf("update of ref %q from %q to %q failed: %w", u.ref, u.oldValue, u.newValue, err)
	}

	u.state = statePost

	return nil
}

// Post runs the pre-receive git hook.
func (u *RefUpdater) Post(ctx context.Context, alternateDirs ...string) error {
	if u.state != statePost {
		return fmt.Errorf("invalid operation order: post-receive hook requires state=%s, current state=%s",
			statePost, u.state)
	}

	out, err := u.hookClient.PostReceive(ctx, PostReceiveInput{
		RefUpdates: []ReferenceUpdate{
			{
				Ref: u.ref,
				Old: u.oldValue,
				New: u.newValue,
			},
		},
		Environment: Environment{
			AlternateObjectDirs: alternateDirs,
		},
	})
	if err != nil {
		return fmt.Errorf("post-receive call failed with: %w", err)
	}
	if out.Error != nil {
		return fmt.Errorf("post-receive call returned error: %q", *out.Error)
	}

	u.state = stateDone

	return nil
}

func (u *RefUpdater) getRef(ctx context.Context) (sha.SHA, error) {
	cmd := command.New("show-ref",
		command.WithFlag("--verify"),
		command.WithFlag("-s"),
		command.WithArg(u.ref),
	)
	output := &bytes.Buffer{}
	err := cmd.Run(ctx, command.WithDir(u.repoPath), command.WithStdout(output))
	if cErr := command.AsError(err); cErr != nil {
		if cErr.IsExitCode(128) && cErr.IsInvalidRefErr() {
			return sha.None, errors.NotFound("reference %q not found", u.ref)
		}
		return sha.None, err
	}

	return sha.New(output.String())
}
