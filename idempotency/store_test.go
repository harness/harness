// Copyright 2026 Harness, Inc.
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

// Package idempotency provides stores that remember which operations were already performed.
package idempotency

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const inFlight = 30 * time.Second

func TestStateString(t *testing.T) {
	t.Parallel()

	require.Equal(t, "claimed", StateClaimed.String())
	require.Equal(t, "in-flight", StateInFlight.String())
	require.Equal(t, "done", StateDone.String())
	require.Equal(t, "unknown", State(42).String())
}
