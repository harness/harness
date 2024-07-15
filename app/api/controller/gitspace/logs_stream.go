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

package gitspace

import (
	"context"
	"encoding/json"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) LogsStream(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	identifier string,
) (<-chan *sse.Event, <-chan error, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find space: %w", err)
	}

	err = apiauth.CheckGitspace(ctx, c.authorizer, session, space.Path, identifier, enum.PermissionGitspaceView)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to authorize: %w", err)
	}

	gitspaceConfig, err := c.gitspaceConfigStore.FindByIdentifier(ctx, space.ID, identifier)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find gitspace config: %w", err)
	}

	linec, errc := c.statefulLogger.TailLogStream(ctx, gitspaceConfig.ID)

	if linec == nil {
		return nil, nil, fmt.Errorf("log stream not present, failed to tail log stream")
	}

	evenc := make(chan *sse.Event)
	errch := make(chan error)

	go func() {
		defer close(evenc)
		defer close(errch)

		for {
			select {
			case <-ctx.Done():
				return
			case line, ok := <-linec:
				if !ok {
					return
				}
				event := sse.Event{
					Type: enum.SSETypeLogLineAppended,
					Data: marshalLine(line),
				}
				evenc <- &event
			case err = <-errc:
				if err != nil {
					errch <- err
					return
				}
			}
		}
	}()

	return evenc, errch, nil
}

func marshalLine(line *livelog.Line) []byte {
	data, _ := json.Marshal(line)
	return data
}
