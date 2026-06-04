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

package store

import (
	"context"

	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/mock"
)

type LinkedPullReqStore struct{ mock.Mock }

func (m *LinkedPullReqStore) Find(_ context.Context, pullReqID int64) (*types.LinkedPullReq, error) {
	args := m.Called(pullReqID)
	if len(args) == 1 {
		if rf, ok := args.Get(0).(func(int64) (*types.LinkedPullReq, error)); ok {
			return rf(pullReqID)
		}
	}
	var v *types.LinkedPullReq
	if args.Get(0) != nil {
		v, _ = args.Get(0).(*types.LinkedPullReq)
	}
	if len(args) > 1 {
		return v, args.Error(1)
	}
	return v, nil
}

func (m *LinkedPullReqStore) FindByLinkedRepoAndProviderPR(
	_ context.Context,
	linkedRepoID int64,
	provider, providerID string,
	providerPRNumber int,
) (*types.LinkedPullReq, error) {
	args := m.Called(linkedRepoID, provider, providerID, providerPRNumber)
	if len(args) == 1 {
		if rf, ok := args.Get(0).(func(int64, string, string, int) (*types.LinkedPullReq, error)); ok {
			return rf(linkedRepoID, provider, providerID, providerPRNumber)
		}
	}
	var v *types.LinkedPullReq
	if args.Get(0) != nil {
		v, _ = args.Get(0).(*types.LinkedPullReq)
	}
	if len(args) > 1 {
		return v, args.Error(1)
	}
	return v, nil
}

func (m *LinkedPullReqStore) Create(_ context.Context, v *types.LinkedPullReq) error {
	return m.Called(v).Error(0)
}

func (m *LinkedPullReqStore) Update(_ context.Context, v *types.LinkedPullReq) error {
	return m.Called(v).Error(0)
}
