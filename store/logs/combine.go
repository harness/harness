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

package logs

import (
	"context"
	"io"

	"github.com/drone/drone/core"
)

// NewCombined returns a new combined log store that will fallback
// to a secondary log store when necessary. This can be useful when
// migrating from database logs to s3, where logs for older builds
// are still being stored in the database, and newer logs in s3.
func NewCombined(primary, secondary core.LogStore) core.LogStore {
	return &combined{
		primary:   primary,
		secondary: secondary,
	}
}

type combined struct {
	primary, secondary core.LogStore
}

func (s *combined) Find(ctx context.Context, step int64) (io.ReadCloser, error) {
	rc, err := s.primary.Find(ctx, step)
	if err == nil {
		return rc, err
	}
	return s.secondary.Find(ctx, step)
}

func (s *combined) Create(ctx context.Context, step int64, r io.Reader) error {
	return s.primary.Create(ctx, step, r)
}

func (s *combined) Update(ctx context.Context, step int64, r io.Reader) error {
	return s.primary.Update(ctx, step, r)
}

func (s *combined) Delete(ctx context.Context, step int64) error {
	err := s.primary.Delete(ctx, step)
	if err != nil {
		err = s.secondary.Delete(ctx, step)
	}
	return err
}
