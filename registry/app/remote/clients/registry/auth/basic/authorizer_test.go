// Source: https://github.com/goharbor/harbor

// Copyright 2016 Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package basic

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModify(t *testing.T) {
	authorizer := NewAuthorizer("u", "p")
	req, _ := http.NewRequest(http.MethodGet, "", nil)
	err := authorizer.Modify(req)
	require.Nil(t, err)
	u, p, ok := req.BasicAuth()
	require.True(t, ok)
	assert.Equal(t, "u", u)
	assert.Equal(t, "p", p)
}
