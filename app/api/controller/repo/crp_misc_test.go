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
	"encoding/json"
	"testing"

	"github.com/harness/gitness/app/auth/authz/authztest"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCrpCodeOwners(t *testing.T) {
	ctx := context.Background()

	newEnv := func(t *testing.T) *crpEnv {
		t.Helper()
		e := crpNewEnv(t)
		e.c.codeOwners = codeowners.New(
			e.c.repoStore, e.git, codeowners.Config{FilePaths: []string{"CODEOWNERS"}}, nil, nil)

		return e
	}

	t.Run("raw file", func(t *testing.T) {
		e := newEnv(t)
		e.git.On("GetTreeNode", mock.Anything, mock.MatchedBy(func(p *git.GetTreeNodeParams) bool {
			return p.Path == "CODEOWNERS" && p.GitREF == "refs/heads/"+crpDefaultBranch
		})).Return(&git.GetTreeNodeOutput{Node: crpTreeNode(git.TreeNodeModeFile, git.TreeNodeTypeBlob, "CODEOWNERS")}, nil)
		e.git.On("GetBlob", mock.Anything, mock.Anything).Return(crpBlob("# owners\n"), nil)

		file, err := e.c.CodeOwnersRaw(ctx, crpSession(), crpRepoRef, "")

		require.NoError(t, err)
		assert.Equal(t, "# owners\n", file.Content)
		assert.EqualValues(t, 9, file.TotalSize)
	})

	t.Run("raw file is missing", func(t *testing.T) {
		e := newEnv(t)
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(nil, errors.NotFound("no such file"))

		_, err := e.c.CodeOwnersRaw(ctx, crpSession(), crpRepoRef, crpFeatureBranch)

		require.Error(t, err)
	})

	t.Run("validation of a file without owners", func(t *testing.T) {
		e := newEnv(t)
		e.git.On("GetTreeNode", mock.Anything, mock.MatchedBy(func(p *git.GetTreeNodeParams) bool {
			return p.GitREF == crpDefaultBranch
		})).Return(&git.GetTreeNodeOutput{Node: crpTreeNode(git.TreeNodeModeFile, git.TreeNodeTypeBlob, "CODEOWNERS")}, nil)
		e.git.On("GetBlob", mock.Anything, mock.Anything).Return(crpBlob("# nothing to see\n"), nil)

		validation, err := e.c.CodeOwnersValidate(ctx, crpSession(), crpRepoRef, "")

		require.NoError(t, err)
		require.NotNil(t, validation)
		assert.Empty(t, validation.Violations)
	})

	t.Run("validation failure", func(t *testing.T) {
		e := newEnv(t)
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(nil, errCrpBoom)

		_, err := e.c.CodeOwnersValidate(ctx, crpSession(), crpRepoRef, crpFeatureBranch)

		require.ErrorIs(t, err, errCrpBoom)
	})
}

func TestCrpNewController(t *testing.T) {
	env := crpNewEnv(t)
	cfg := &types.Config{}
	cfg.Git.DefaultBranch = "trunk"

	constructors := map[string]func() *Controller{
		"NewController": func() *Controller {
			return NewController(
				cfg,                          // config
				crpTx{},                      // tx
				crpURLProvider{},             // urlProvider
				authztest.AllowAuthorizer{},  // authorizer
				nil,                          // repoStore
				nil,                          // repoActivityStore
				nil,                          // linkedRepoStore
				nil,                          // spaceStore
				nil,                          // pipelineStore
				nil,                          // executionStore
				nil,                          // principalStore
				nil,                          // ruleStore
				nil,                          // checkStore
				nil,                          // pullReqStore
				nil,                          // settings
				nil,                          // principalInfoCache
				nil,                          // protectionManager
				nil,                          // git
				env.c.spaceFinder,            // spaceFinder
				env.c.repoFinder,             // repoFinder
				nil,                          // importer
				nil,                          // referenceSync
				nil,                          // importLinked
				nil,                          // codeOwners
				nil,                          // contentValidators
				nil,                          // eventReporter
				nil,                          // indexer
				nil,                          // limiter
				crpNewLocker(),               // locker
				nil,                          // auditService
				nil,                          // mtxManager
				env.c.identifierCheck,        // identifierCheck
				NewNoOpRepoChecks(),          // repoCheck
				nil,                          // publicAccess
				nil,                          // labelSvc
				instrument.Noop{},            // instrumentation
				nil,                          // userGroupStore
				nil,                          // userGroupService
				nil,                          // rulesSvc
				nil,                          // sseStreamer
				nil,                          // lfsCtrl
				nil,                          // favoriteStore
				env.c.signatureVerifyService, // signatureVerifyService
				nil,                          // autolinkSvc
				nil,                          // dotRangeService
				nil,                          // connectorService
				nil,                          // webhookService
				nil,                          // repoLangStore
				nil,                          // mergeQueueService
			)
		},
		"ProvideController": func() *Controller {
			return ProvideController(
				cfg,                          // config
				crpTx{},                      // tx
				crpURLProvider{},             // urlProvider
				authztest.AllowAuthorizer{},  // authorizer
				nil,                          // repoStore
				nil,                          // repoActivityStore
				nil,                          // linkedRepoStore
				nil,                          // spaceStore
				nil,                          // pipelineStore
				nil,                          // executionStore
				nil,                          // principalStore
				nil,                          // ruleStore
				nil,                          // checkStore
				nil,                          // pullReqStore
				nil,                          // settings
				nil,                          // principalInfoCache
				nil,                          // protectionManager
				nil,                          // git
				env.c.spaceFinder,            // spaceFinder
				env.c.repoFinder,             // repoFinder
				nil,                          // importer
				nil,                          // referenceSync
				nil,                          // importLinked
				nil,                          // codeOwners
				nil,                          // contentValidators
				nil,                          // eventReporter
				nil,                          // indexer
				nil,                          // limiter
				crpNewLocker(),               // locker
				nil,                          // auditService
				nil,                          // mtxManager
				env.c.identifierCheck,        // identifierCheck
				NewNoOpRepoChecks(),          // repoCheck
				nil,                          // publicAccess
				nil,                          // labelSvc
				instrument.Noop{},            // instrumentation
				nil,                          // userGroupStore
				nil,                          // userGroupService
				nil,                          // rulesSvc
				nil,                          // sseStreamer
				nil,                          // lfsCtrl
				nil,                          // favoriteStore
				env.c.signatureVerifyService, // signatureVerifyService
				nil,                          // autolinkSvc
				nil,                          // dotRangeService
				nil,                          // connectorService
				nil,                          // webhookService
				nil,                          // repoLangStore
				nil,                          // mergeQueueService
			)
		},
	}

	for name, construct := range constructors {
		t.Run(name, func(t *testing.T) {
			c := construct()

			require.NotNil(t, c)
			assert.Equal(t, "trunk", c.defaultBranch)
			assert.Equal(t, crpTx{}, c.tx)
			assert.Equal(t, crpURLProvider{}, c.urlProvider)
			assert.Equal(t, authztest.AllowAuthorizer{}, c.authorizer)
			assert.Equal(t, env.c.spaceFinder, c.spaceFinder)
			assert.Equal(t, env.c.repoFinder, c.repoFinder)
		})
	}
}

func TestCrpRepositoryOutput_MarshalJSON(t *testing.T) {
	out := RepositoryOutput{
		Repository: types.Repository{ID: 3, Identifier: "my-repo", State: enum.RepoStateActive},
		IsPublic:   true,
	}

	raw, err := json.Marshal(out)

	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, "my-repo", decoded["identifier"])
	assert.Equal(t, "my-repo", decoded["uid"], "the deprecated uid mirrors the identifier")
	assert.Equal(t, true, decoded["is_public"])
}

func TestCrpContentTypesImplementContent(t *testing.T) {
	contents := []Content{&FileContent{}, &SymlinkContent{}, &DirContent{}, &SubmoduleContent{}}

	for _, content := range contents {
		assert.NotPanics(t, content.isContent)
	}
}

func TestCrpProvideRepoCheck(t *testing.T) {
	check := ProvideRepoCheck()

	require.NotNil(t, check)
	require.NoError(t, check.Create(context.Background(), crpSession(), &CheckInput{}))
	require.NoError(t, check.LifecycleRestriction(context.Background(), crpSession(), &types.RepositoryCore{}))
}
