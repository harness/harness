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

package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	appstore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types/enum"
)

// KeyValue is a struct used for upserting many entries.
type KeyValue struct {
	Key   Key
	Value any
}

// SettingHandler is an abstraction of a component that's handling a single setting value as part of
// calling service.Map.
type SettingHandler interface {
	Key() Key
	Required() bool
	Handle(ctx context.Context, raw []byte) error
}

// Service is used to enhance interaction with the settings store.
type Service struct {
	settingsStore appstore.SettingsStore
}

func NewService(
	settingsStore appstore.SettingsStore,
) *Service {
	return &Service{
		settingsStore: settingsStore,
	}
}

// Set sets the value of the setting with the given key for the given scope.
func (s *Service) Set(
	ctx context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	key Key,
	value any,
) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal setting value: %w", err)
	}

	err = s.settingsStore.Upsert(
		ctx,
		scope,
		scopeID,
		string(key),
		raw,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert setting in store: %w", err)
	}

	return nil
}

// SetMany sets the value of the settings with the given keys for the given scope.
func (s *Service) SetMany(
	ctx context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	keyValues ...KeyValue,
) error {
	// TODO: batch upsert
	for _, kv := range keyValues {
		if err := s.Set(ctx, scope, scopeID, kv.Key, kv.Value); err != nil {
			return fmt.Errorf("failed to set setting for key %q: %w", kv.Key, err)
		}
	}

	return nil
}

// Get returns the value of the setting with the given key for the given scope.
func (s *Service) Get(
	ctx context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	key Key,
	out any,
) (bool, error) {
	raw, err := s.settingsStore.Find(
		ctx,
		scope,
		scopeID,
		string(key),
	)
	if errors.Is(err, store.ErrResourceNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to find setting in store: %w", err)
	}

	err = json.Unmarshal(raw, &out)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal setting value: %w", err)
	}

	return true, nil
}

// Map maps all available settings using the provided handlers for the given scope.
func (s *Service) Map(
	ctx context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	handlers ...SettingHandler,
) error {
	if len(handlers) == 0 {
		return nil
	}

	keys := make([]string, len(handlers))
	for i, m := range handlers {
		keys[i] = string(m.Key())
	}

	rawValues, err := s.settingsStore.FindMany(
		ctx,
		scope,
		scopeID,
		keys...,
	)
	if err != nil {
		return fmt.Errorf("failed to find settings in store: %w", err)
	}

	for _, m := range handlers {
		rawValue, found := rawValues[string(m.Key())]
		if !found && m.Required() {
			return fmt.Errorf("required setting %q not found", m.Key())
		}
		if !found {
			continue
		}

		if err = m.Handle(ctx, rawValue); err != nil {
			return fmt.Errorf("failed to handle value for setting %q: %w", m.Key(), err)
		}
	}

	return nil
}
