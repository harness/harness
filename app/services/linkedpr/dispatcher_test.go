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

package linkedpr_test

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/services/linkedpr"
	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/mock"
)

func linkedRepoLookup(rows []types.LinkedRepo, err error) *mockstore.LinkedRepoStore {
	m := &mockstore.LinkedRepoStore{}
	m.On("ListByProviderID", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(rows, err)
	return m
}

// fakeHandler implements linkedpr.Handler.
type fakeHandler struct {
	calls    int
	gotRepos []int64
	err      error
}

func (f *fakeHandler) Handle(_ context.Context, _ *linkedpr.Event, lr *types.LinkedRepo) error {
	f.calls++
	if lr != nil {
		f.gotRepos = append(f.gotRepos, lr.RepoID)
	}
	return f.err
}

// buildPREvent returns a minimal valid pull_request Event for tests. mutators
// can adjust fields per-case.
func buildPREvent(mutators ...func(*linkedpr.Event)) *linkedpr.Event {
	ev := &linkedpr.Event{
		Provider:   linkedpr.ProviderGitHub,
		AccountID:  "acct-x",
		DeliveryID: "deliv-1",
		Payload: linkedpr.PullRequestPayload{
			Number: 7,
			Repository: linkedpr.Repository{
				ProviderID: "280125018",
			},
		},
	}
	for _, m := range mutators {
		m(ev)
	}
	return ev
}

func handlersWith(h linkedpr.Handler) linkedpr.Handlers {
	return linkedpr.Handlers{
		{Kind: linkedpr.KindPullRequest, Provider: linkedpr.ProviderGitHub}: h,
	}
}

func TestDispatch_HappyPathRoutesToHandler(t *testing.T) {
	lookup := &mockstore.LinkedRepoStore{}
	lookup.On(
		"ListByProviderID",
		"acct-x", string(linkedpr.ProviderGitHub), "280125018", mock.Anything,
	).Return([]types.LinkedRepo{{RepoID: 100}}, nil)
	h := &fakeHandler{}

	d := linkedpr.NewDispatcher(lookup, handlersWith(h))
	res, err := d.Dispatch(context.Background(), buildPREvent())
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if res != linkedpr.ResultDispatched {
		t.Errorf("Result: got %v, want Dispatched", res)
	}
	if h.calls != 1 || h.gotRepos[0] != 100 {
		t.Errorf("handler calls: %d, repos: %v", h.calls, h.gotRepos)
	}
	lookup.AssertExpectations(t)
}

func TestDispatch_FansOutToAllLinkedRepos(t *testing.T) {
	lookup := linkedRepoLookup([]types.LinkedRepo{{RepoID: 1}, {RepoID: 2}, {RepoID: 3}}, nil)
	h := &fakeHandler{}

	d := linkedpr.NewDispatcher(lookup, handlersWith(h))
	res, err := d.Dispatch(context.Background(), buildPREvent())
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if res != linkedpr.ResultDispatched {
		t.Errorf("Result: got %v, want Dispatched", res)
	}
	if h.calls != 3 {
		t.Errorf("handler calls: got %d, want 3", h.calls)
	}
}

// stubPayload implements linkedpr.Payload with a caller-controlled Kind so
// the dispatcher routes it to a key with no registered handler.
type stubPayload struct{ kind linkedpr.Kind }

func (s stubPayload) Kind() linkedpr.Kind    { return s.kind }
func (s stubPayload) RepoProviderID() string { return "R_stub" }

func TestDispatch_DropOnUnsupportedKind(t *testing.T) {
	d := linkedpr.NewDispatcher(linkedRepoLookup(nil, nil), handlersWith(&fakeHandler{}))
	ev := buildPREvent(func(e *linkedpr.Event) { e.Payload = stubPayload{kind: linkedpr.Kind("check_run")} })
	res, _ := d.Dispatch(context.Background(), ev)
	if res != linkedpr.ResultDroppedUnsupportedEvent {
		t.Errorf("Result: got %v, want DroppedUnsupportedEvent", res)
	}
}

func TestDispatch_DropOnEmptyHandlers(t *testing.T) {
	d := linkedpr.NewDispatcher(linkedRepoLookup(nil, nil), nil)
	res, _ := d.Dispatch(context.Background(), buildPREvent())
	if res != linkedpr.ResultDroppedUnsupportedEvent {
		t.Errorf("Result: got %v, want DroppedUnsupportedEvent", res)
	}
}

func TestDispatch_DropOnMissingRepoProviderID(t *testing.T) {
	d := linkedpr.NewDispatcher(linkedRepoLookup(nil, nil), handlersWith(&fakeHandler{}))
	ev := buildPREvent(func(e *linkedpr.Event) {
		p, ok := e.Payload.(linkedpr.PullRequestPayload)
		if !ok {
			t.Fatalf("Payload is not PullRequestPayload: %T", e.Payload)
		}
		p.Repository.ProviderID = ""
		e.Payload = p
	})
	res, _ := d.Dispatch(context.Background(), ev)
	if res != linkedpr.ResultDroppedMalformedEvent {
		t.Errorf("Result: got %v, want DroppedDecodeFailed", res)
	}
}

func TestDispatch_DropOnMissingDeliveryID(t *testing.T) {
	d := linkedpr.NewDispatcher(linkedRepoLookup(nil, nil), handlersWith(&fakeHandler{}))
	ev := buildPREvent(func(e *linkedpr.Event) { e.DeliveryID = "" })
	res, _ := d.Dispatch(context.Background(), ev)
	if res != linkedpr.ResultDroppedMalformedEvent {
		t.Errorf("Result: got %v, want DroppedDecodeFailed", res)
	}
}

func TestDispatch_DropOnNoLinkedRepo(t *testing.T) {
	lookup := linkedRepoLookup(nil, nil)
	h := &fakeHandler{}
	d := linkedpr.NewDispatcher(lookup, handlersWith(h))
	res, _ := d.Dispatch(context.Background(), buildPREvent())
	if res != linkedpr.ResultDroppedNotLinked {
		t.Errorf("Result: got %v, want DroppedNotLinked", res)
	}
	if h.calls != 0 {
		t.Errorf("handler should not have been called, got %d calls", h.calls)
	}
}

func TestDispatch_LookupErrorRetries(t *testing.T) {
	lookup := linkedRepoLookup(nil, errors.New("db down"))
	d := linkedpr.NewDispatcher(lookup, handlersWith(&fakeHandler{}))
	_, err := d.Dispatch(context.Background(), buildPREvent())
	if err == nil {
		t.Errorf("expected error so caller doesn't ack, got nil")
	}
}

func TestDispatch_HandlerErrorRetries(t *testing.T) {
	lookup := linkedRepoLookup([]types.LinkedRepo{{RepoID: 100}}, nil)
	h := &fakeHandler{err: errors.New("transient")}
	d := linkedpr.NewDispatcher(lookup, handlersWith(h))
	_, err := d.Dispatch(context.Background(), buildPREvent())
	if err == nil {
		t.Errorf("expected error so caller doesn't ack, got nil")
	}
}

func TestDispatch_NilEventDropsGracefully(t *testing.T) {
	d := linkedpr.NewDispatcher(linkedRepoLookup(nil, nil), handlersWith(&fakeHandler{}))
	res, err := d.Dispatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if res != linkedpr.ResultDroppedMalformedEvent {
		t.Errorf("Result: got %v, want DroppedDecodeFailed", res)
	}
}
