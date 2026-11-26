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

package authn

import (
	"testing"

	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func TestCt5ProvideAuthenticator(t *testing.T) {
	t.Run("generates anonymous secret when missing", func(t *testing.T) {
		config := &types.Config{}
		require.NotNil(t, ProvideAuthenticator(config, nil, nil))
		require.Len(t, config.Auth.AnonymousUserSecret, 32)
	})

	t.Run("keeps configured secret", func(t *testing.T) {
		config := &types.Config{}
		config.Auth.AnonymousUserSecret = "configured"
		require.NotNil(t, ProvideAuthenticator(config, nil, nil))
		require.Equal(t, "configured", config.Auth.AnonymousUserSecret)
	})
}
