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

package capabilities

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/types/capabilities"
)

// newLogic This function helps us adhere to the same function definition, across different capabilities
// Each capability takes in an input and returns an output, by using this function, each capability can have its own
// input and output definition.
func newLogic[T capabilities.Input, U capabilities.Output](
	logic func(ctx context.Context, input T) (U, error)) capabilities.Logic {
	return func(ctx context.Context, input capabilities.Input) (capabilities.Output, error) {
		myInput, ok := input.(T)
		if !ok {
			return nil, fmt.Errorf("invalid input type")
		}
		return logic(ctx, myInput)
	}
}

func NewRegistry() *Registry {
	return &Registry{
		capabilities: make(map[capabilities.Type]capabilities.Capability),
	}
}

type Registry struct {
	capabilities map[capabilities.Type]capabilities.Capability
}

func (r *Registry) register(c capabilities.Capability) error {
	if _, ok := r.capabilities[c.Type]; ok {
		return fmt.Errorf("capability %q already registered", c.Type)
	}

	r.capabilities[c.Type] = c

	return nil
}

func (r *Registry) Exists(t capabilities.Type) bool {
	_, ok := r.capabilities[t]
	return ok
}

func (r *Registry) ReturnToUser(t capabilities.Type) (bool, error) {
	c, ok := r.capabilities[t]
	if !ok {
		return false, fmt.Errorf("unknown capability type %q", t)
	}
	return c.ReturnToUser, nil
}

func (r *Registry) Get(t capabilities.Type) (capabilities.Capability, bool) {
	c, ok := r.capabilities[t]
	return c, ok
}

func (r *Registry) Execute(
	ctx context.Context, t capabilities.Type, in capabilities.Input) (capabilities.Output, error) {
	c, ok := r.Get(t)
	if !ok {
		return nil, fmt.Errorf("unknown capability type %q", t)
	}

	out, err := c.Logic(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed execution: %w", err)
	}

	return out, nil
}

func DeserializeInput(cr *Registry, t capabilities.Type, raw json.RawMessage) (capabilities.Input, error) {
	capability, ok := cr.Get(t)
	if !ok {
		return nil, fmt.Errorf("unknown type: %s", t)
	}

	input := capability.NewInput()

	err := json.Unmarshal(raw, input)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal input: %w", err)
	}

	return input, nil
}

func (r *Registry) Capabilities() []capabilities.CapabilityReference {
	var capabilitiesList []capabilities.CapabilityReference
	for _, capability := range r.capabilities {
		c := capabilities.CapabilityReference{
			Type:    capability.Type,
			Version: capability.Version,
		}
		capabilitiesList = append(capabilitiesList, c)
	}
	return capabilitiesList
}

type RepoRef struct {
	Ref string `json:"ref"`
}

func (RepoRef) IsCapabilityOutput() {}
