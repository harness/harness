// Copyright 2026 Harness, Inc.
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

package migrate

import (
	"context"
	"fmt"
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type ctlEnv struct {
	*ctlBase
	ctrl  *Controller
	space *types.Space
	repo  *types.Repository
}

func ctlNewEnv(t *testing.T) *ctlEnv {
	t.Helper()
	b := ctlNewBase(t)
	space := b.createSpace(t, "root", 0)
	repo := b.createRepo(t, space, "repo1")

	ctrl := NewController(
		ctlAuthorizer{mode: ctlAllow}, nil, nil, nil, nil, nil, nil, nil,
		limiter.NewResourceLimiter(), audit.New(), check.RepoIdentifierDefault,
		b.tx, b.spaces, b.repos, b.spaceFinder, b.repoFinder, nil,
	)
	return &ctlEnv{ctlBase: b, ctrl: ctrl, space: space, repo: repo}
}

func TestCtlMigrateController_UpdateRepoState(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	tests := []struct {
		name    string
		from    enum.RepoState
		to      enum.RepoState
		force   bool
		wantErr bool
	}{
		{"active to data import", enum.RepoStateActive, enum.RepoStateMigrateDataImport, false, false},
		{"data import to active", enum.RepoStateMigrateDataImport, enum.RepoStateActive, false, false},
		{"git push to active", enum.RepoStateMigrateGitPush, enum.RepoStateActive, false, false},
		{"git push to data import", enum.RepoStateMigrateGitPush, enum.RepoStateMigrateDataImport, false, false},
		{"active to git push rejected", enum.RepoStateActive, enum.RepoStateMigrateGitPush, false, true},
		{"git import to active rejected", enum.RepoStateGitImport, enum.RepoStateActive, false, true},
		{"git import to active forced", enum.RepoStateGitImport, enum.RepoStateActive, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identifier := fmt.Sprintf("repo-%d-%d-%t", tt.from, tt.to, tt.force)
			repo := e.createRepo(t, e.space, identifier) //nolint:contextcheck // helper uses the test context.
			repo.State = tt.from
			require.NoError(t, e.repos.Update(ctx, repo))
			e.repoFinder.MarkChanged(ctx, repo.Core())

			out, err := e.ctrl.UpdateRepoState(ctx, s, "root/"+repo.Identifier, &UpdateStateInput{State: tt.to, Force: tt.force})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.to, out.State)
		})
	}

	_, err := e.ctrl.UpdateRepoState(ctx, s, "root/unknown", &UpdateStateInput{State: enum.RepoStateActive})
	require.Error(t, err)
}

func TestCtlMigrateController_ImportStateChecks(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	// The repo is active, so none of the import calls is acceptable.
	_, err := e.ctrl.Rules(ctx, s, "root/repo1", &RulesInput{})
	require.Error(t, err)
	_, err = e.ctrl.Webhooks(ctx, s, "root/repo1", &WebhooksInput{})
	require.Error(t, err)
	_, err = e.ctrl.PullRequests(ctx, s, "root/repo1", &PullreqsInput{})
	require.Error(t, err)
	require.True(t, errors.IsPreconditionFailed(err))
}

func TestCtlMigrateController_AccessFailures(t *testing.T) {
	ctx := context.Background()
	allow := ctlNewEnv(t)
	s := allow.session()

	for _, mode := range []ctlAuthMode{ctlDeny, ctlError} {
		c := *allow.ctrl
		c.authorizer = ctlAuthorizer{mode: mode}

		errs := map[string]error{}
		_, errs["state"] = c.UpdateRepoState(ctx, s, "root/repo1", &UpdateStateInput{})
		_, errs["rules"] = c.Rules(ctx, s, "root/repo1", &RulesInput{})
		_, errs["webhooks"] = c.Webhooks(ctx, s, "root/repo1", &WebhooksInput{})
		_, errs["pullreqs"] = c.PullRequests(ctx, s, "root/repo1", &PullreqsInput{})
		_, errs["labels"] = c.Labels(ctx, s, "root", &LabelsInput{})
		_, errs["create repo"] = c.CreateRepo(ctx, s, &CreateRepoInput{ParentRef: "root", Identifier: "new"})
		for name, err := range errs {
			require.Error(t, err, name)
			if mode == ctlError {
				require.ErrorIs(t, err, errCtlAuthz, name)
			} else {
				require.ErrorIs(t, err, apiauth.ErrForbidden, name)
			}
		}
	}

	// Missing references.
	_, err := allow.ctrl.Rules(ctx, s, "", &RulesInput{})
	require.Error(t, err)
	_, err = allow.ctrl.Rules(ctx, s, "root/unknown", &RulesInput{})
	require.Error(t, err)
}

func TestCtlMigrateController_CreateRepoValidation(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	_, err := e.ctrl.CreateRepo(ctx, s, &CreateRepoInput{ParentRef: "", Identifier: "new"})
	require.Error(t, err)
	_, err = e.ctrl.CreateRepo(ctx, s, &CreateRepoInput{ParentRef: "0", Identifier: "new"})
	require.Error(t, err)
	_, err = e.ctrl.CreateRepo(ctx, s, &CreateRepoInput{ParentRef: "root", Identifier: ""})
	require.Error(t, err)
	_, err = e.ctrl.CreateRepo(ctx, s, &CreateRepoInput{ParentRef: "unknown", Identifier: "new"})
	require.Error(t, err)
}

func TestCtlMigrateController_LabelsUnknownSpace(t *testing.T) {
	e := ctlNewEnv(t)
	_, err := e.ctrl.Labels(context.Background(), e.session(), "unknown", &LabelsInput{})
	require.Error(t, err)
}
