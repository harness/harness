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

// Package linkedpr dispatches inbound linked-PR events. Broker-agnostic;
// transport adapters live outside this package.
package linkedpr

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// LinkedRepoLookup is the subset of store.LinkedRepoStore the dispatcher uses.
type LinkedRepoLookup interface {
	ListByProviderID(
		ctx context.Context,
		accountID, provider, providerID string,
		pagination types.Pagination,
	) ([]types.LinkedRepo, error)
}

// Result classifies one Dispatch call's outcome.
// TODO(observability): emit a counter keyed by Result.String().
type Result int

const (
	ResultUnknown Result = iota
	ResultDispatched
	ResultDroppedNotLinked
	ResultDroppedUnsupportedEvent
	ResultDroppedMalformedEvent
)

func (r Result) String() string {
	switch r {
	case ResultUnknown:
		return "unknown"
	case ResultDispatched:
		return "dispatched"
	case ResultDroppedNotLinked:
		return "dropped_not_linked"
	case ResultDroppedUnsupportedEvent:
		return "dropped_unsupported_event"
	case ResultDroppedMalformedEvent:
		return "dropped_malformed_event"
	default:
		return "unknown"
	}
}

// HandlerKey routes a (Kind, Provider) to its handler.
type HandlerKey struct {
	Kind     Kind
	Provider Provider
}

// Handlers maps (Kind, Provider) to its Handler. Unmatched events are dropped.
type Handlers map[HandlerKey]Handler

// Dispatcher orchestrates inbound linked-PR events.
type Dispatcher struct {
	linkedRepos LinkedRepoLookup
	handlers    Handlers
}

// NewDispatcher wires the dependencies. An empty handlers map is valid.
func NewDispatcher(
	linkedRepos LinkedRepoLookup,
	handlers Handlers,
) *Dispatcher {
	return &Dispatcher{
		linkedRepos: linkedRepos,
		handlers:    handlers,
	}
}

// Dispatch processes one event. Returns (Result, nil) for happy-path or a
// deliberate drop; (Result, err) for a transient failure the broker should retry.
func (d *Dispatcher) Dispatch(ctx context.Context, ev *Event) (Result, error) {
	if ev == nil {
		log.Ctx(ctx).Warn().Msg("linkedpr: nil event")
		return ResultDroppedMalformedEvent, nil
	}

	if ev.Payload == nil {
		log.Ctx(ctx).Warn().Str("account_id", ev.AccountID).Str("delivery_id", ev.DeliveryID).
			Msg("linkedpr: event has no payload")
		return ResultDroppedMalformedEvent, nil
	}

	handler, ok := d.handlers[HandlerKey{Kind: ev.Payload.Kind(), Provider: ev.Provider}]
	if !ok || handler == nil {
		return ResultDroppedUnsupportedEvent, nil
	}

	repoProviderID := ev.Payload.RepoProviderID()
	if repoProviderID == "" {
		log.Ctx(ctx).Warn().
			Str("kind", string(ev.Payload.Kind())).
			Str("provider", string(ev.Provider)).
			Str("delivery_id", ev.DeliveryID).
			Msg("linkedpr: payload missing repo provider id")
		return ResultDroppedMalformedEvent, nil
	}
	if ev.DeliveryID == "" {
		log.Ctx(ctx).Warn().
			Str("kind", string(ev.Payload.Kind())).
			Str("provider", string(ev.Provider)).
			Msg("linkedpr: missing delivery id")
		return ResultDroppedMalformedEvent, nil
	}
	if ev.AccountID == "" {
		log.Ctx(ctx).Warn().
			Str("kind", string(ev.Payload.Kind())).
			Str("provider", string(ev.Provider)).
			Str("delivery_id", ev.DeliveryID).
			Msg("linkedpr: missing account id")
		return ResultDroppedMalformedEvent, nil
	}

	linked, err := listAllByProviderID(ctx, d.linkedRepos, ev.AccountID, string(ev.Provider), repoProviderID)
	if err != nil {
		return ResultUnknown, fmt.Errorf("dispatch: lookup linked repos: %w", err)
	}
	if len(linked) == 0 {
		return ResultDroppedNotLinked, nil
	}

	// Fan out across every linked repo so one broken connector doesn't
	// starve the others; handlers are idempotent.
	var errs []error
	for i := range linked {
		if err := handler.Handle(ctx, ev, &linked[i]); err != nil {
			errs = append(errs, fmt.Errorf("repo %d: %w", linked[i].RepoID, err))
		}
	}
	if len(errs) > 0 {
		return ResultUnknown, fmt.Errorf("dispatch: handler failures: %w",
			errors.Join(errs...))
	}

	return ResultDispatched, nil
}

// listAllByProviderID walks every page of linked repos matching the provider
// identity so webhook fan-out is not capped by a single query limit.
func listAllByProviderID(
	ctx context.Context,
	lookup LinkedRepoLookup,
	accountID, provider, providerID string,
) ([]types.LinkedRepo, error) {
	const pageSize = 100

	var all []types.LinkedRepo
	for page := 1; ; page++ {
		batch, err := lookup.ListByProviderID(ctx, accountID, provider, providerID, types.Pagination{
			Page: page,
			Size: pageSize,
		})
		if err != nil {
			return nil, err
		}

		all = append(all, batch...)
		if len(batch) < pageSize {
			break
		}
	}

	return all, nil
}
