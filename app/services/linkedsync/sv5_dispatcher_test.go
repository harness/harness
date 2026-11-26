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

package linkedsync_test

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/services/linkedsync"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type sv5Handler struct{ calls int }

func (h *sv5Handler) Handle(context.Context, *linkedsync.Event, *types.LinkedRepo) error {
	h.calls++
	return nil
}

type sv5Lookup struct{ called bool }

func (l *sv5Lookup) ListByProviderID(
	context.Context, string, string, string, types.Pagination,
) ([]types.LinkedRepo, error) {
	l.called = true
	return nil, nil
}

func sv5Event(mutate func(*linkedsync.Event)) *linkedsync.Event {
	ev := &linkedsync.Event{
		Provider:   linkedsync.ProviderGitHub,
		AccountID:  "acct-x",
		DeliveryID: "deliv-1",
		Payload: linkedsync.PullRequestPayload{
			Number:     7,
			Repository: linkedsync.Repository{ProviderID: "280125018"},
		},
	}
	if mutate != nil {
		mutate(ev)
	}
	return ev
}

func TestSv5ResultString(t *testing.T) {
	tests := map[linkedsync.Result]string{
		linkedsync.ResultUnknown:                 "unknown",
		linkedsync.ResultDispatched:              "dispatched",
		linkedsync.ResultDroppedNotLinked:        "dropped_not_linked",
		linkedsync.ResultDroppedUnsupportedEvent: "dropped_unsupported_event",
		linkedsync.ResultDroppedMalformedEvent:   "dropped_malformed_event",
		linkedsync.Result(99):                    "unknown",
	}
	for result, want := range tests {
		require.Equal(t, want, result.String())
	}
}

func TestSv5NewHandlersRejectsInvalidRegistrations(t *testing.T) {
	handler := &sv5Handler{}

	_, err := linkedsync.NewHandlers(linkedsync.ProviderHandlers{
		Handlers: map[linkedsync.Kind]linkedsync.Handler{linkedsync.KindPullRequest: handler},
	})
	require.ErrorContains(t, err, "has no provider")

	_, err = linkedsync.NewHandlers(linkedsync.ProviderHandlers{
		Provider: linkedsync.ProviderGitHub,
		Handlers: map[linkedsync.Kind]linkedsync.Handler{"": handler},
	})
	require.ErrorContains(t, err, "invalid handler")

	_, err = linkedsync.NewHandlers(linkedsync.ProviderHandlers{
		Provider: linkedsync.ProviderGitHub,
		Handlers: map[linkedsync.Kind]linkedsync.Handler{linkedsync.KindPush: nil},
	})
	require.ErrorContains(t, err, "invalid handler")
}

func TestSv5DispatchDropsMalformedEventsWithoutLookup(t *testing.T) {
	handler := &sv5Handler{}
	handlers := linkedsync.Handlers{
		{Kind: linkedsync.KindPullRequest, Provider: linkedsync.ProviderGitHub}: handler,
	}
	lookup := &sv5Lookup{}
	dispatcher := linkedsync.NewDispatcher(lookup, handlers)

	tests := map[string]*linkedsync.Event{
		"nil event":   nil,
		"no payload":  sv5Event(func(ev *linkedsync.Event) { ev.Payload = nil }),
		"no account":  sv5Event(func(ev *linkedsync.Event) { ev.AccountID = "" }),
		"no delivery": sv5Event(func(ev *linkedsync.Event) { ev.DeliveryID = "" }),
	}
	for name, ev := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := dispatcher.Dispatch(context.Background(), ev)
			require.NoError(t, err)
			require.Equal(t, linkedsync.ResultDroppedMalformedEvent, result)
		})
	}

	require.False(t, lookup.called)
	require.Zero(t, handler.calls)
}

func TestSv5DispatchDropsEventsOfOtherProviders(t *testing.T) {
	handlers := linkedsync.Handlers{
		{Kind: linkedsync.KindPullRequest, Provider: linkedsync.ProviderGitHub}: &sv5Handler{},
	}
	dispatcher := linkedsync.NewDispatcher(&sv5Lookup{}, handlers)

	result, err := dispatcher.Dispatch(context.Background(), sv5Event(func(ev *linkedsync.Event) {
		ev.Provider = linkedsync.ProviderBitbucket
	}))
	require.NoError(t, err)
	require.Equal(t, linkedsync.ResultDroppedUnsupportedEvent, result)
}
