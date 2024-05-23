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
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c controller) List(
	ctx context.Context,
	session *auth.Session,
	opts *types.PrincipalFilter,
) ([]*types.PrincipalInfo, error) {
	// only user search is supported right now!
	if len(opts.Types) != 1 || opts.Types[0] != enum.PrincipalTypeUser {
		return nil, usererror.Newf(
			http.StatusNotImplemented,
			"Only listing of users is supported at this moment (use query '%s=%s').",
			request.QueryParamType,
			enum.PrincipalTypeUser,
		)
	}

	if err := apiauth.Check(
		ctx,
		c.authorizer,
		session,
		&types.Scope{},
		&types.Resource{
			Type: enum.ResourceTypeUser,
		},
		enum.PermissionUserView,
	); err != nil {
		return nil, err
	}

	principals, err := c.principalStore.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	pInfoUsers := make([]*types.PrincipalInfo, len(principals))
	for i := range principals {
		pInfoUsers[i] = principals[i].ToPrincipalInfo()
	}

	return pInfoUsers, nil
}
