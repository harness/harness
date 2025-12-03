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

package blob

import (
	"context"
	"fmt"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideStore,
)

func ProvideStore(ctx context.Context, config Config) (Store, error) {
	switch config.Provider {
	case ProviderFileSystem:
		return NewFileSystemStore(config)
	case ProviderGCS:
		return NewGCSStore(ctx, config)
	default:
		return nil, fmt.Errorf("invalid blob store provider: %s", config.Provider)
	}
}
