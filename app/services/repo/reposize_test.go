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
	"errors"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
)

type recordingListener struct {
	calls      int
	measuredAt int64
}

func (r *recordingListener) AfterSizeCalculation(_ context.Context, measuredAt int64) {
	r.calls++
	r.measuredAt = measuredAt
}

// fakeRepoStoreFailing satisfies store.RepoStore via a nil embedded field, overriding only
// ListSizeInfos. The rest panic if ever called.
type fakeRepoStoreFailing struct {
	store.RepoStore
}

func (*fakeRepoStoreFailing) ListSizeInfos(context.Context) ([]*types.RepositorySizeInfo, error) {
	return nil, errors.New("boom")
}

// fakeSpaceStoreFailing makes sendMetric's own failure path harmless, so these tests exercise only
// the listener.
type fakeSpaceStoreFailing struct {
	store.SpaceStore
}

func (*fakeSpaceStoreFailing) GetRootSpacesSize(context.Context) ([]types.SpaceStorage, error) {
	return nil, errors.New("boom")
}

// A run that fails before it updates a single size must still notify: the listener reads the sizes
// that are in the database, which this run did not invalidate.
func TestSizeCalculator_NotifiesListenerOnFailedRun(t *testing.T) {
	listener := &recordingListener{}
	calc := &SizeCalculator{
		enabled:    true,
		repoStore:  &fakeRepoStoreFailing{},
		spaceStore: &fakeSpaceStoreFailing{},
		listener:   listener,
	}

	if _, err := calc.Handle(context.Background(), "", nil); err == nil {
		t.Fatal("Handle succeeded, want the ListSizeInfos error to propagate")
	}
	if listener.calls != 1 {
		t.Errorf("listener called %d times, want 1", listener.calls)
	}
	if listener.measuredAt == 0 {
		t.Error("listener called with measuredAt 0, want the run's end time")
	}
}

// A disabled calculator did not measure anything, so there is nothing to notify about.
func TestSizeCalculator_DisabledDoesNotNotify(t *testing.T) {
	listener := &recordingListener{}
	calc := &SizeCalculator{
		enabled:    false,
		spaceStore: &fakeSpaceStoreFailing{},
		listener:   listener,
	}

	if _, err := calc.Handle(context.Background(), "", nil); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if listener.calls != 0 {
		t.Errorf("listener called %d times, want 0", listener.calls)
	}
}
