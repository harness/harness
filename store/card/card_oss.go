// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build oss

package card

import (
	"context"
	"io"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"
)

func New(db *db.DB) core.CardStore {
	return new(noop)
}

type noop struct{}

func (noop) FindCardByBuild(ctx context.Context, build int64) ([]*core.Card, error) {
	return nil, nil
}

func (noop) FindCard(ctx context.Context, step int64) (*core.Card, error) {
	return nil, nil
}

func (noop) FindCardData(ctx context.Context, id int64) (io.Reader, error) {
	return nil, nil
}

func (noop) CreateCard(ctx context.Context, card *core.CreateCard) error {
	return nil
}

func (noop) DeleteCard(ctx context.Context, id int64) error {
	return nil
}
