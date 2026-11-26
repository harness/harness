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

package repo

import (
	"context"
	"net/http"
	"testing"

	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// crpWebhookService is an importer.WebhookService with a fixed result that records the call.
type crpWebhookService struct {
	err   error
	calls *[]importer.UpsertWebhookInput
}

func (s crpWebhookService) UpsertWebhook(_ context.Context, in importer.UpsertWebhookInput) error {
	if s.calls != nil {
		*s.calls = append(*s.calls, in)
	}

	return s.err
}

func crpLinkedCreateInput() *LinkedCreateInput {
	return &LinkedCreateInput{
		ParentRef: "3", Identifier: "linked", ConnectorRef: "conn", RepoIdentifier: " owner/name ",
	}
}

// crpLinkedCreateEnv prepares an environment where LinkedCreate succeeds.
func crpLinkedCreateEnv(t *testing.T, hooks *[]importer.UpsertWebhookInput) *crpEnv {
	t.Helper()

	e := crpNewEnv(t)
	e.c.connectorService = crpConnectorService{
		info:         importer.AccessInfo{URL: "https://github.example.com/owner/name.git", Username: "u", Password: "p"},
		providerInfo: importer.ProviderRepoInfo{RepoID: "12345", Type: importer.ProviderTypeGitHub},
	}
	e.c.webhookService = crpWebhookService{calls: hooks}
	e.c.linkedRepoStore = &crpLinkedRepoCreateStore{}
	e.withLinkJobs(t, crpJobStore{})
	e.git.On("CreateRepository", mock.Anything, mock.Anything).Return(&git.CreateRepositoryOutput{UID: "tmpuid"}, nil)
	e.git.On("GetRemoteDefaultBranch", mock.Anything, mock.MatchedBy(func(p *git.GetRemoteDefaultBranchParams) bool {
		return p.Source == "https://u:p@github.example.com/owner/name.git" && p.RepoUID == "tmpuid"
	})).Return(&git.GetRemoteDefaultBranchOutput{BranchName: "trunk"}, nil)
	e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)

	return e
}

// crpLinkedRepoCreateStore is a store.LinkedRepoStore recording created linked repositories.
type crpLinkedRepoCreateStore struct {
	crpLinkedRepoStore
	created []int64
	err     error
}

func (s *crpLinkedRepoCreateStore) Create(_ context.Context, v *types.LinkedRepo) error {
	if s.err != nil {
		return s.err
	}

	s.created = append(s.created, v.RepoID)

	return nil
}

func TestCrpLinkedCreate_Success(t *testing.T) {
	ctx := context.Background()
	var hooks []importer.UpsertWebhookInput
	e := crpLinkedCreateEnv(t, &hooks)

	out, err := e.c.LinkedCreate(ctx, crpSession(), crpLinkedCreateInput())

	require.NoError(t, err)
	assert.Equal(t, enum.RepoTypeLinked, out.Type)
	assert.Equal(t, "linked", out.Identifier)
	assert.Equal(t, "trunk", out.DefaultBranch)
	assert.Equal(t, "https://git.example.com/space/linked.git", out.GitURL)
	require.Len(t, hooks, 1)
	assert.Equal(t, "space", hooks[0].SpacePath)
	assert.Equal(t, "https://github.example.com/owner/name.git", hooks[0].CloneURL)
	assert.Equal(t, []int64{out.ID}, crpLinkedStoreOf(t, e).created)
	e.git.AssertNumberOfCalls(t, "DeleteRepository", 1)
}

func TestCrpLinkedCreate_Failures(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		setup func(e *crpEnv)
		in    func() *LinkedCreateInput
		check func(t *testing.T, e *crpEnv, err error)
	}{
		{
			name:  "invalid input",
			in:    func() *LinkedCreateInput { in := crpLinkedCreateInput(); in.ConnectorRef = ""; return in },
			check: func(t *testing.T, _ *crpEnv, err error) { require.Error(t, err) },
		},
		{
			name:  "invalid identifier",
			setup: func(e *crpEnv) { e.c.identifierCheck = func(string, *auth.Session) error { return errCrpBoom } },
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name:  "no permission to create in the parent",
			setup: func(e *crpEnv) { e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom}) },
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name:  "public access support lookup failure",
			setup: func(e *crpEnv) { e.public.supportedErr = errCrpBoom },
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name:  "public repositories are disabled",
			setup: func(e *crpEnv) { e.public.unsupported = true },
			in:    func() *LinkedCreateInput { in := crpLinkedCreateInput(); in.IsPublic = true; return in },
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errPublicRepoCreationDisabled) },
		},
		{
			name:  "connector access info failure",
			setup: func(e *crpEnv) { e.c.connectorService = crpConnectorService{err: errCrpBoom} },
			check: func(t *testing.T, _ *crpEnv, err error) {
				require.Error(t, err)
				assert.True(t, errors.IsInvalidArgument(err))
			},
		},
		{
			name: "unusable connector url",
			setup: func(e *crpEnv) {
				e.c.connectorService = crpConnectorService{info: importer.AccessInfo{URL: "://bad"}}
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.Error(t, err) },
		},
		{
			name: "temporary git repository failure",
			setup: func(e *crpEnv) {
				e.git.ExpectedCalls = nil
				e.git.On("CreateRepository", mock.Anything, mock.Anything).Return(nil, errCrpBoom)
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.Error(t, err) },
		},
		{
			name: "remote default branch failure",
			setup: func(e *crpEnv) {
				e.git.ExpectedCalls = nil
				e.git.On("CreateRepository", mock.Anything, mock.Anything).Return(&git.CreateRepositoryOutput{UID: "t"}, nil)
				e.git.On("GetRemoteDefaultBranch", mock.Anything, mock.Anything).Return(nil, errCrpBoom)
				e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(errCrpBoom)
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.Error(t, err) },
		},
		{
			name: "provider info failure",
			setup: func(e *crpEnv) {
				e.c.connectorService = crpConnectorService{
					info: importer.AccessInfo{
						URL: "https://github.example.com/owner/name.git", Username: "u", Password: "p",
					},
					providerErr: errCrpBoom,
				}
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.Error(t, err) },
		},
		{
			name:  "webhook user fault",
			setup: func(e *crpEnv) { e.c.webhookService = crpWebhookService{err: importer.ErrWebhookUpsertUserFault} },
			check: func(t *testing.T, e *crpEnv, err error) {
				require.Error(t, err)
				assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, err))
				assert.False(t, e.repoStore.called("Create"))
			},
		},
		{
			name:  "webhook infra failure",
			setup: func(e *crpEnv) { e.c.webhookService = crpWebhookService{err: errCrpBoom} },
			check: func(t *testing.T, e *crpEnv, err error) {
				require.Error(t, err)
				assert.False(t, e.repoStore.called("Create"))
			},
		},
		{
			name:  "repo quota reached",
			setup: func(e *crpEnv) { e.c.resourceLimiter = crpRepoCountLimiter{err: errCrpBoom} },
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, limiter.ErrMaxNumReposReached) },
		},
		{
			name: "storage over limit",
			setup: func(e *crpEnv) {
				e.c.resourceLimiter = restoreLimiterStub{
					rootSpaceStorageErr: &limiter.TotalQuotaStorageError{Level: limiter.TotalQuotaLevelOverLimit},
				}
			},
			check: func(t *testing.T, _ *crpEnv, err error) { require.Error(t, err) },
		},
		{
			name:  "space lookup failure",
			setup: func(e *crpEnv) { e.spaces.err = errCrpBoom },
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name:  "repo store failure",
			setup: func(e *crpEnv) { e.repoStore.errs["Create"] = errCrpBoom },
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name:  "linked repo store failure",
			setup: func(e *crpEnv) { e.c.linkedRepoStore = &crpLinkedRepoCreateStore{err: errCrpBoom} },
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
		{
			name:  "job start failure",
			setup: func(e *crpEnv) { e.withLinkJobs(t, crpJobStore{createErr: errCrpBoom}) },
			check: func(t *testing.T, _ *crpEnv, err error) { require.ErrorIs(t, err, errCrpBoom) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := crpLinkedCreateEnv(t, nil)
			if tt.setup != nil {
				tt.setup(e)
			}
			in := crpLinkedCreateInput()
			if tt.in != nil {
				in = tt.in()
			}

			_, err := e.c.LinkedCreate(ctx, crpSession(), in)

			tt.check(t, e, err)
		})
	}
}
