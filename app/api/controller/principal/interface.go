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

package principal

import (
	"context"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
)

// Controller interface provides an abstraction that allows to have different implementations of
// principal related information.
type Controller interface {
	// List lists the principals based on the provided filter.
	List(ctx context.Context, session *auth.Session, opts *types.PrincipalFilter) ([]*types.PrincipalInfo, error)
	Find(ctx context.Context, session *auth.Session, principalID int64) (*types.PrincipalInfo, error)
	CheckExistenceByEmails(ctx context.Context, session *auth.Session, input *CheckUsersInput) (*CheckUsersOutput, error)
}
