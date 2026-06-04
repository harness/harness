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
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/linkedpr"
	"github.com/harness/gitness/types"
)

// otherPayload is a different Payload impl used to assert SafeHandler rejects
// a mistyped payload without panicking.
type otherPayload struct{}

func (otherPayload) Kind() linkedpr.Kind    { return linkedpr.Kind("other") }
func (otherPayload) RepoProviderID() string { return "R_other" }

func TestSafeHandler_DispatchesPayloadToTypedFn(t *testing.T) {
	var got linkedpr.PullRequestPayload
	fn := func(_ context.Context, _ *linkedpr.Event, p linkedpr.PullRequestPayload, _ *types.LinkedRepo) error {
		got = p
		return nil
	}
	h := linkedpr.SafeHandler[linkedpr.PullRequestPayload]("test", fn)
	ev := &linkedpr.Event{
		Provider:   linkedpr.ProviderGitHub,
		AccountID:  "a",
		DeliveryID: "d",
		Payload: linkedpr.PullRequestPayload{
			Number:     42,
			Repository: linkedpr.Repository{ProviderID: "280125018"},
		},
	}
	if err := h.Handle(context.Background(), ev, &types.LinkedRepo{RepoID: 1}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Number != 42 || got.Repository.ProviderID != "280125018" {
		t.Errorf("typed payload not delivered to fn: %+v", got)
	}
}

func TestSafeHandler_RejectsNilEvent(t *testing.T) {
	h := linkedpr.SafeHandler[linkedpr.PullRequestPayload]("test",
		func(context.Context, *linkedpr.Event, linkedpr.PullRequestPayload, *types.LinkedRepo) error {
			t.Fatal("inner fn must not be called for nil event")
			return nil
		})
	if err := h.Handle(context.Background(), nil, &types.LinkedRepo{}); err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestSafeHandler_RejectsNilPayload(t *testing.T) {
	h := linkedpr.SafeHandler[linkedpr.PullRequestPayload]("test",
		func(context.Context, *linkedpr.Event, linkedpr.PullRequestPayload, *types.LinkedRepo) error {
			t.Fatal("inner fn must not be called for nil payload")
			return nil
		})
	ev := &linkedpr.Event{Provider: linkedpr.ProviderGitHub, DeliveryID: "d"}
	if err := h.Handle(context.Background(), ev, &types.LinkedRepo{}); err == nil {
		t.Fatal("expected error for nil payload")
	}
}

func TestSafeHandler_RejectsMismatchedPayloadType(t *testing.T) {
	h := linkedpr.SafeHandler[linkedpr.PullRequestPayload]("test",
		func(context.Context, *linkedpr.Event, linkedpr.PullRequestPayload, *types.LinkedRepo) error {
			t.Fatal("inner fn must not be called for mismatched payload type")
			return nil
		})
	ev := &linkedpr.Event{
		Provider:   linkedpr.ProviderGitHub,
		DeliveryID: "d",
		Payload:    otherPayload{},
	}
	err := h.Handle(context.Background(), ev, &types.LinkedRepo{})
	if err == nil {
		t.Fatal("expected error for payload type mismatch")
	}
	if !strings.Contains(err.Error(), "payload type mismatch") {
		t.Errorf("error should mention type mismatch: %v", err)
	}
}

func TestSafeHandler_PropagatesInnerError(t *testing.T) {
	want := errors.New("inner boom")
	h := linkedpr.SafeHandler[linkedpr.PullRequestPayload]("test",
		func(context.Context, *linkedpr.Event, linkedpr.PullRequestPayload, *types.LinkedRepo) error {
			return want
		})
	ev := &linkedpr.Event{
		Provider:   linkedpr.ProviderGitHub,
		DeliveryID: "d",
		Payload: linkedpr.PullRequestPayload{
			Repository: linkedpr.Repository{ProviderID: "280125018"},
		},
	}
	if err := h.Handle(context.Background(), ev, &types.LinkedRepo{}); !errors.Is(err, want) {
		t.Fatalf("expected wrapped inner error, got %v", err)
	}
}
