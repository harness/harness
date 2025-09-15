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
	"net/http"

	"github.com/harness/gitness/app/api/controller/gitspace"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func HandleListAllGitspaces(gitspaceCtrl *gitspace.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		deleted := false
		markedForDeletion := false
		filter := types.GitspaceFilter{
			GitspaceInstanceFilter: types.GitspaceInstanceFilter{UserIdentifier: session.Principal.UID},
			Deleted:                &deleted,
			MarkedForDeletion:      &markedForDeletion,
		}
		filter.Owner = enum.GitspaceOwnerSelf
		maxListing := types.Pagination{
			Page: 0,
			Size: 10000,
		}
		filter.QueryFilter = types.ListQueryFilter{
			Pagination: maxListing,
		}

		// For List all gitspaces api in gitness, we will send allSpaceIDs as true
		// This is fetch all the root spaces IDs and list all gitspaces within these root space IDs.
		// For gitness we can show gitspaces from all root IDs(ideally there will be 1 root space).
		// This could not be done for cde-manager that different root space IDs map to different accounts.
		const allSpaceIDs = true

		gitspaces, err := gitspaceCtrl.ListAllGitspaces(ctx, session, filter, allSpaceIDs)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		render.JSON(w, http.StatusOK, gitspaces)
	}
}
