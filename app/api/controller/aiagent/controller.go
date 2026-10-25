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

// Package aiagent provides the controller of the AI agent panel of a space.
package aiagent

import (
	"context"

	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

type (
	spaceFinder interface {
		FindByRef(ctx context.Context, spaceRef string) (*types.SpaceCore, error)
	}

	repoFinder interface {
		FindByID(ctx context.Context, repoID int64) (*types.RepositoryCore, error)
	}

	repoLister interface {
		List(ctx context.Context, parentID int64, filter *types.RepoFilter) ([]*types.Repository, error)
	}

	// quotaSource is the part of the AI gateway used to read the quota planner.
	quotaSource interface {
		Enabled() bool
		Snapshot() aigateway.QuotaSnapshot
		Remaining(class aigateway.Class) int
	}
)

// Controller handles the AI agent panel API.
type Controller struct {
	authorizer    authz.Authorizer
	spaceFinder   spaceFinder
	repoFinder    repoFinder
	repoStore     repoLister
	taskStore     store.TaskStore
	activityStore store.TaskActivityStore
	aiTaskStore   store.AITaskStore
	modelStore    store.AIModelStore
	usageStore    store.AIUsageStore
	quota         quotaSource
}

// NewController returns a new Controller.
func NewController(
	authorizer authz.Authorizer,
	spaceFinder spaceFinder,
	repoFinder repoFinder,
	repoStore repoLister,
	taskStore store.TaskStore,
	activityStore store.TaskActivityStore,
	aiTaskStore store.AITaskStore,
	modelStore store.AIModelStore,
	usageStore store.AIUsageStore,
	quota quotaSource,
) *Controller {
	return &Controller{
		authorizer:    authorizer,
		spaceFinder:   spaceFinder,
		repoFinder:    repoFinder,
		repoStore:     repoStore,
		taskStore:     taskStore,
		activityStore: activityStore,
		aiTaskStore:   aiTaskStore,
		modelStore:    modelStore,
		usageStore:    usageStore,
		quota:         quota,
	}
}

// gatewayQuota adapts the AI gateway to quotaSource; a nil gateway is disabled.
type gatewayQuota struct {
	gateway *aigateway.Gateway
}

func (g gatewayQuota) Enabled() bool { return g.gateway.Enabled() }

func (g gatewayQuota) Snapshot() aigateway.QuotaSnapshot { return g.gateway.Quota() }

func (g gatewayQuota) Remaining(class aigateway.Class) int {
	return g.gateway.Planner().Remaining(class)
}

// ProvideController provides the AI agent panel controller.
func ProvideController(
	authorizer authz.Authorizer,
	spaceFinder refcache.SpaceFinder,
	repoFinder refcache.RepoFinder,
	repoStore store.RepoStore,
	taskStore store.TaskStore,
	activityStore store.TaskActivityStore,
	aiTaskStore store.AITaskStore,
	modelStore store.AIModelStore,
	usageStore store.AIUsageStore,
	gateway *aigateway.Gateway,
) *Controller {
	return NewController(
		authorizer, spaceFinder, repoFinder, repoStore, taskStore, activityStore,
		aiTaskStore, modelStore, usageStore, gatewayQuota{gateway: gateway})
}
