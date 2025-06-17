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

package locker

import (
	"context"
	"fmt"
	"time"
)

func (l Locker) LockResource(
	ctx context.Context,
	key string,
	expiry time.Duration,
) (func(), error) {
	unlockFn, err := l.lock(ctx, namespaceRegistry, key, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to lock mutex for key [%s]: %w", key, err)
	}

	return unlockFn, nil
}
