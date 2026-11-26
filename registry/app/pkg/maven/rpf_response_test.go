//  Copyright 2023 Harness, Inc.
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

package maven

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRpfMavenResponses(t *testing.T) {
	boom := errors.New("boom")
	responses := map[string]Response{
		"head": &HeadArtifactResponse{Errors: []error{errors.New("old"), errors.New("older")}},
		"get":  &GetArtifactResponse{},
		"put":  &PutArtifactResponse{Errors: []error{errors.New("old")}},
	}
	for name, r := range responses {
		t.Run(name, func(t *testing.T) {
			r.SetError(boom)
			require.Equal(t, []error{boom}, r.GetErrors())
		})
	}
}
