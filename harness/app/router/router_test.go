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

package router

import "testing"

// this unit test ensures routes that require authorization
// return a 401 unauthorized if no token, or an invalid token
// is provided.
func TestTokenGate(t *testing.T) {
	t.Skip()
}

// this unit test ensures routes that require pipeline access
// return a 403 forbidden if the user does not have acess
// to the pipeline.
func TestPipelineGate(t *testing.T) {
	t.Skip()
}

// this unit test ensures routes that require system access
// return a 403 forbidden if the user does not have acess
// to the pipeline.
func TestSystemGate(t *testing.T) {
	t.Skip()
}
