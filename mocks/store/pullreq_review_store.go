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

type PullReqReviewStore struct{ mock.Mock }

func (m *PullReqReviewStore) Find(_ context.Context, id int64) (*types.PullReqReview, error) {
	args := m.Called(id)
	if v, _ := args.Get(0).(*types.PullReqReview); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *PullReqReviewStore) Create(_ context.Context, review *types.PullReqReview) error {
	return m.Called(review).Error(0)
}
