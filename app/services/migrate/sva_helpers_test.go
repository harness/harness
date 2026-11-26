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

package migrate

import (
	"context"
	"errors"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
)

var errSvaBoom = errors.New("boom")

// svaTx is a transactor that simply runs the function (optionally failing before running it).
type svaTx struct {
	err error
}

func (t svaTx) WithTx(ctx context.Context, fn func(ctx context.Context) error, _ ...any) error {
	if t.err != nil {
		return t.err
	}
	return fn(ctx)
}

// svaSpaceStore returns a fixed ancestor chain.
type svaSpaceStore struct {
	store.SpaceStore

	ancestors []int64
	err       error
	calls     int
}

func (s *svaSpaceStore) GetAncestorIDs(context.Context, int64) ([]int64, error) {
	s.calls++
	return s.ancestors, s.err
}

// svaLabelStore is an in-memory label store.
type svaLabelStore struct {
	store.LabelStore

	labels       map[string]*types.Label
	nextID       int64
	findErr      error
	defineErr    error
	incrementErr error
	increments   map[int64]int
	defined      []*types.Label
}

func svaNewLabelStore() *svaLabelStore {
	return &svaLabelStore{labels: map[string]*types.Label{}, increments: map[int64]int{}, nextID: 100}
}

func (s *svaLabelStore) Find(_ context.Context, _, _ *int64, key string) (*types.Label, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	label, ok := s.labels[key]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	return label, nil
}

func (s *svaLabelStore) Define(_ context.Context, label *types.Label) error {
	if s.defineErr != nil {
		return s.defineErr
	}
	s.nextID++
	label.ID = s.nextID
	s.labels[label.Key] = label
	s.defined = append(s.defined, label)
	return nil
}

func (s *svaLabelStore) IncrementValueCount(_ context.Context, labelID int64, inc int) (int64, error) {
	if s.incrementErr != nil {
		return 0, s.incrementErr
	}
	s.increments[labelID] += inc
	return int64(s.increments[labelID]), nil
}

// svaLabelValueStore is an in-memory label value store.
type svaLabelValueStore struct {
	store.LabelValueStore

	values    map[int64][]*types.LabelValue
	nextID    int64
	listErr   error
	defineErr error
	findErr   error
}

func svaNewLabelValueStore() *svaLabelValueStore {
	return &svaLabelValueStore{values: map[int64][]*types.LabelValue{}, nextID: 500}
}

func (s *svaLabelValueStore) List(
	_ context.Context, labelID int64, _ types.ListQueryFilter,
) ([]*types.LabelValue, error) {
	return s.values[labelID], s.listErr
}

func (s *svaLabelValueStore) Define(_ context.Context, value *types.LabelValue) error {
	if s.defineErr != nil {
		return s.defineErr
	}
	s.nextID++
	value.ID = s.nextID
	s.values[value.LabelID] = append(s.values[value.LabelID], value)
	return nil
}

func (s *svaLabelValueStore) FindByLabelID(_ context.Context, labelID int64, value string) (*types.LabelValue, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	for _, v := range s.values[labelID] {
		if v.Value == value {
			return v, nil
		}
	}
	return nil, gitness_store.ErrResourceNotFound
}

// svaPrincipalStore resolves principals by email.
type svaPrincipalStore struct {
	store.PrincipalStore

	byEmail map[string]*types.Principal
	err     error
}

func (s *svaPrincipalStore) FindByEmail(_ context.Context, email string) (*types.Principal, error) {
	if s.err != nil {
		return nil, s.err
	}
	if p, ok := s.byEmail[email]; ok {
		return p, nil
	}
	return nil, gitness_store.ErrResourceNotFound
}
