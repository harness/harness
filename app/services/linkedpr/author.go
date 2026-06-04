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

package linkedpr

import (
	"context"

	"github.com/harness/gitness/app/bootstrap"
)

// AuthorResolver maps a linked-PR author to a Harness principal id used for
// pullreqs.pullreq_created_by. The provider-side identity (login, avatar URL,
// profile URL) is stored separately on linked_pullreqs.
type AuthorResolver interface {
	Resolve(ctx context.Context, ev *Event) (principalID int64, err error)
}

// SystemPrincipalResolver returns the gitness system service principal id.
// PrincipalID > 0 overrides (used by tests); 0 reads from bootstrap.
type SystemPrincipalResolver struct {
	PrincipalID int64
}

func (r *SystemPrincipalResolver) Resolve(_ context.Context, _ *Event) (int64, error) {
	if r.PrincipalID > 0 {
		return r.PrincipalID, nil
	}
	return bootstrap.NewSystemServiceSession().Principal.ID, nil
}
