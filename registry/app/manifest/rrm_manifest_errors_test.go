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

package manifest

import (
	"fmt"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

func TestRrmManifestErrors(t *testing.T) {
	require.Equal(t, "unknown tag=latest", TagUnknownError{Tag: "latest"}.Error())
	require.Equal(t, "unknown registry name=repo", RegistryUnknownError{Name: "repo"}.Error())
	require.Contains(t, RegistryNameInvalidError{Name: "bad", Reason: fmt.Errorf("reason")}.Error(), "invalid")
	require.Contains(t, UnknownError{Name: "img", Tag: "v1"}.Error(), "unknown manifest")
	require.Contains(t, UnknownRevisionError{Name: "img", Revision: "sha256:abc"}.Error(), "revision")
	require.Equal(t, "unverified manifest", UnverifiedError{}.Error())
	require.Contains(t, ReferencesExceedLimitError{References: 3, Limit: 2}.Error(), "exceed reference limit")
	require.Contains(t, PayloadSizeExceedsLimitError{PayloadSize: 3, Limit: 2}.Error(), "exceeds limit")
	require.Contains(t, BlobUnknownError{Digest: digest.Digest("sha256:abc")}.Error(), "unknown blob")
	require.Contains(t, NameInvalidError{Name: "x", Reason: fmt.Errorf("y")}.Error(), "invalid")

	errs := VerificationErrors{fmt.Errorf("one"), fmt.Errorf("two")}
	require.Contains(t, errs.Error(), "errors verifying manifest")
}
