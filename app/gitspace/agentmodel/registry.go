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

// Package agentmodel carries the model chosen for a fix agent from the launcher to the gitspace orchestrator.
package agentmodel

import (
	"context"
	"sync"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(NewRegistry)

type ctxKey struct{}

// WithConfigID returns a context that carries the id of the gitspace config being resumed.
func WithConfigID(ctx context.Context, gitspaceConfigID int64) context.Context {
	return context.WithValue(ctx, ctxKey{}, gitspaceConfigID)
}

// ConfigIDFrom returns the gitspace config id set by WithConfigID.
func ConfigIDFrom(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(ctxKey{}).(int64)
	return id, ok
}

// Registry maps a gitspace config to the model its next agent run must use. It is in memory: after a
// restart the orchestrator falls back to its deterministic model choice.
type Registry struct {
	mu     sync.RWMutex
	models map[int64]string
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{models: map[int64]string{}}
}

// Set records the model for the gitspace config.
func (r *Registry) Set(gitspaceConfigID int64, model string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[gitspaceConfigID] = model
}

// Get returns the model recorded for the gitspace config.
func (r *Registry) Get(gitspaceConfigID int64) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.models[gitspaceConfigID]
	return m, ok
}

// Delete forgets the model of the gitspace config.
func (r *Registry) Delete(gitspaceConfigID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.models, gitspaceConfigID)
}
