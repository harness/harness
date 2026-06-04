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

package linkedpr

import (
	"context"
	"fmt"

	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// Handler processes one Event for one matching linked repository. Concrete
// implementations should be registered via SafeHandler to inherit the common
// safety + observability flow.
type Handler interface {
	Handle(ctx context.Context, ev *Event, linkedRepo *types.LinkedRepo) error
}

// TypedHandler is the shape a concrete handler implements: it receives the
// already-validated typed payload, so the body is pure business logic.
type TypedHandler[T Payload] func(
	ctx context.Context,
	ev *Event,
	payload T,
	linkedRepo *types.LinkedRepo,
) error

// SafeHandler decorates a TypedHandler with the mandatory pre/post flow:
// reject nil event / nil payload, type-assert the payload (no panic on
// mismatch), and emit start/end logs tagged with the handler name.
func SafeHandler[T Payload](name string, fn TypedHandler[T]) Handler {
	return &safeHandler[T]{name: name, fn: fn}
}

type safeHandler[T Payload] struct {
	name string
	fn   TypedHandler[T]
}

func (h *safeHandler[T]) Handle(
	ctx context.Context,
	ev *Event,
	linkedRepo *types.LinkedRepo,
) error {
	if ev == nil {
		return fmt.Errorf("linkedpr: %s: nil event", h.name)
	}
	if ev.Payload == nil {
		return fmt.Errorf("linkedpr: %s: nil payload", h.name)
	}
	if linkedRepo == nil {
		return fmt.Errorf("linkedpr: %s: nil linked repo", h.name)
	}
	payload, ok := ev.Payload.(T)
	if !ok {
		// This is a wiring bug: the Handlers registry routed an event whose
		// Kind doesn't match this handler's expected payload type.
		return fmt.Errorf(
			"linkedpr: %s: payload type mismatch (got kind=%s)",
			h.name, ev.Payload.Kind(),
		)
	}

	logger := log.Ctx(ctx).With().
		Str("linkedpr.handler", h.name).
		Str("linkedpr.kind", string(ev.Payload.Kind())).
		Str("linkedpr.provider", string(ev.Provider)).
		Int64("linkedpr.linked_repo_id", linkedRepo.RepoID).
		Logger()
	ctx = logger.WithContext(ctx)

	logger.Debug().Msg("linkedpr: handler start")
	err := h.fn(ctx, ev, payload, linkedRepo)
	if err != nil {
		return fmt.Errorf("linkedpr: %s: %w", h.name, err)
	}
	logger.Debug().Msg("linkedpr: handler ok")
	return nil
}
