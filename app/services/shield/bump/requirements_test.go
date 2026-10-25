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

package bump

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBumpRequirements(t *testing.T) {
	in := "# deps\nDjango==3.2.1  # web\nrequests[security] == 2.0.0 ; python_version>'3'\nother==1.0\n"
	out, err := BumpRequirements([]byte(in), "django", "3.2.1", "3.2.25")
	require.NoError(t, err)
	require.Equal(t, "# deps\nDjango==3.2.25  # web\nrequests[security] == 2.0.0 ; python_version>'3'\nother==1.0\n",
		string(out))

	out, err = BumpRequirements([]byte(in), "Requests", "2.0.0", "2.0.1")
	require.NoError(t, err)
	require.Contains(t, string(out), "requests[security] == 2.0.1 ; python_version>'3'")

	// PEP 503 name normalisation and CRLF
	out, err = BumpRequirements([]byte("Foo_Bar==1.0\r\n"), "foo-bar", "1.0", "1.1")
	require.NoError(t, err)
	require.Equal(t, "Foo_Bar==1.1\r\n", string(out))
}

func TestBumpRequirementsRefusals(t *testing.T) {
	_, err := BumpRequirements([]byte("django==3.2.1 \\\n    --hash=sha256:abc\n"), "django", "3.2.1", "3.2.2")
	require.True(t, IsNeedsAgent(err), err)

	_, err = BumpRequirements([]byte("django==3.2.1\n"), "django", "3.0.0", "3.2.2")
	require.True(t, IsNeedsAgent(err), err)

	_, err = BumpRequirements([]byte("django>=3.2.1\n"), "django", "3.2.1", "3.2.2")
	require.ErrorIs(t, err, ErrUnsupported)
}
