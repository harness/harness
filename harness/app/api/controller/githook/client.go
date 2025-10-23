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

package githook

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

var _ hook.ClientFactory = (*ControllerClientFactory)(nil)
var _ hook.Client = (*ControllerClient)(nil)

// ControllerClientFactory creates clients that directly call the controller to execute githooks.
type ControllerClientFactory struct {
	githookCtrl *Controller
	git         git.Interface
}

func (f *ControllerClientFactory) NewClient(envVars map[string]string) (hook.Client, error) {
	payload, err := hook.LoadPayloadFromMap[githook.Payload](envVars)
	if err != nil {
		return nil, fmt.Errorf("failed to load payload from provided map of environment variables: %w", err)
	}

	// ensure we return disabled message in case it's explicitly disabled
	if payload.Disabled {
		return hook.NewNoopClient([]string{"hook disabled"}), nil
	}

	if err := payload.Validate(); err != nil {
		return nil, fmt.Errorf("payload validation failed: %w", err)
	}

	return &ControllerClient{
		baseInput:   githook.GetInputBaseFromPayload(payload),
		githookCtrl: f.githookCtrl,
		git:         f.git,
	}, nil
}

// ControllerClient directly calls the controller to execute githooks.
type ControllerClient struct {
	baseInput   types.GithookInputBase
	githookCtrl *Controller
	git         RestrictedGIT
}

func (c *ControllerClient) PreReceive(
	ctx context.Context,
	in hook.PreReceiveInput,
) (hook.Output, error) {
	log.Ctx(ctx).Debug().Int64("repo_id", c.baseInput.RepoID).Msg("calling pre-receive")

	out, err := c.githookCtrl.PreReceive(
		ctx,
		c.git, // Harness doesn't require any custom git connector.
		nil,   // TODO: update once githooks are auth protected
		types.GithookPreReceiveInput{
			GithookInputBase: c.baseInput,
			PreReceiveInput:  in,
		},
	)
	if err != nil {
		return hook.Output{}, translateControllerError(err)
	}

	return out, nil
}

func (c *ControllerClient) Update(
	ctx context.Context,
	in hook.UpdateInput,
) (hook.Output, error) {
	log.Ctx(ctx).Debug().Int64("repo_id", c.baseInput.RepoID).Msg("calling update")

	out, err := c.githookCtrl.Update(
		ctx,
		c.git, // Harness doesn't require any custom git connector.
		nil,   // TODO: update once githooks are auth protected
		types.GithookUpdateInput{
			GithookInputBase: c.baseInput,
			UpdateInput:      in,
		},
	)
	if err != nil {
		return hook.Output{}, translateControllerError(err)
	}

	return out, nil
}

func (c *ControllerClient) PostReceive(
	ctx context.Context,
	in hook.PostReceiveInput,
) (hook.Output, error) {
	log.Ctx(ctx).Debug().Int64("repo_id", c.baseInput.RepoID).Msg("calling post-receive")

	out, err := c.githookCtrl.PostReceive(
		ctx,
		c.git, // Harness doesn't require any custom git connector.
		nil,   // TODO: update once githooks are auth protected
		types.GithookPostReceiveInput{
			GithookInputBase: c.baseInput,
			PostReceiveInput: in,
		},
	)
	if err != nil {
		return hook.Output{}, translateControllerError(err)
	}

	return out, nil
}

func translateControllerError(err error) error {
	if errors.Is(err, store.ErrResourceNotFound) {
		return hook.ErrNotFound
	}

	return err
}
