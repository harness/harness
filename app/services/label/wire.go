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

package label

import (
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideLabel,
)

func ProvideLabel(
	tx dbtx.Transactor,
	spaceStore store.SpaceStore,
	labelStore store.LabelStore,
	labelValueStore store.LabelValueStore,
	pullReqLabelStore store.PullReqLabelAssignmentStore,
) *Service {
	return New(tx, spaceStore, labelStore, labelValueStore, pullReqLabelStore)
}
