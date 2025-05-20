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

package protection

import (
	"github.com/harness/gitness/app/store"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideManager,
)

func ProvideManager(ruleStore store.RuleStore) (*Manager, error) {
	m := NewManager(ruleStore)

	if err := m.Register(TypeBranch, func() Definition { return &Branch{} }); err != nil {
		return nil, err
	}

	if err := m.Register(TypeTag, func() Definition { return &Tag{} }); err != nil {
		return nil, err
	}

	return m, nil
}
