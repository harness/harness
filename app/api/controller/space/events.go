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

package space

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Events(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
) (<-chan *sse.Event, <-chan error, func(context.Context) error, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to find space ref: %w", err)
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to authorize stream: %w", err)
	}

	chEvents, chErr, sseCancel := c.sseStreamer.Stream(ctx, space.ID)

	return chEvents, chErr, sseCancel, nil
}
