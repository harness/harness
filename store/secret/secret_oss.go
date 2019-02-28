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

package secret

import (
	"context"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"
	"github.com/drone/drone/store/shared/encrypt"
)

// New returns a new Secret database store.
func New(db *db.DB, enc encrypt.Encrypter) core.SecretStore {
	return new(noop)
}

type noop struct{}

func (noop) List(ctx context.Context, id int64) ([]*core.Secret, error) {
	return nil, nil
}

func (noop) Find(ctx context.Context, id int64) (*core.Secret, error) {
	return nil, nil
}

func (noop) FindName(ctx context.Context, id int64, name string) (*core.Secret, error) {
	return nil, nil
}

func (noop) Create(ctx context.Context, secret *core.Secret) error {
	return nil
}

func (noop) Update(context.Context, *core.Secret) error {
	return nil
}

func (noop) Delete(context.Context, *core.Secret) error {
	return nil
}
