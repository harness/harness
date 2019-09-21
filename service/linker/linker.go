// Copyright 2019 Drone IO, Inc.
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

package linker

import (
	"context"

	"github.com/drone/drone/core"
	"github.com/drone/go-scm/scm"
)

// New returns a new Linker server.
func New(client *scm.Client) core.Linker {
	return &service{
		client: client,
	}
}

type service struct {
	client *scm.Client
}

func (s *service) Link(ctx context.Context, repo, ref, sha string) (string, error) {
	return s.client.Linker.Resource(ctx, repo, scm.Reference{
		Path: ref,
		Sha:  sha,
	})
}
