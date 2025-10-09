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
	"fmt"
)

// Mapping returns a SettingHandler that maps the value of the setting with the given key to the target.
func Mapping[T any](key Key, target *T) SettingHandler {
	if target == nil {
		panic("mapping target can't be nil")
	}
	return &settingHandlerMapping[T]{
		key:      key,
		required: false,
		target:   target,
	}
}

// MappingRequired returns a SettingHandler that maps the value of the setting with the given key to the target.
// If the setting wasn't found an error is returned.
func MappingRequired[T any](key Key, target *T) SettingHandler {
	if target == nil {
		panic("mapping target can't be nil")
	}
	return &settingHandlerMapping[T]{
		key:      key,
		required: true,
		target:   target,
	}
}

var _ SettingHandler = (*settingHandlerMapping[any])(nil)

// settingHandlerMapping is a setting handler that maps the value of a setting to the provided target.
type settingHandlerMapping[T any] struct {
	key      Key
	required bool
	target   *T
}

func (q *settingHandlerMapping[T]) Key() Key {
	return q.key
}

func (q *settingHandlerMapping[T]) Required() bool {
	return q.required
}

func (q *settingHandlerMapping[T]) Handle(_ context.Context, raw []byte) error {
	err := json.Unmarshal(raw, q.target)
	if err != nil {
		return fmt.Errorf("failed to unmarshal setting value: %w", err)
	}

	return nil
}
