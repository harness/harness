// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
// to the pipeline
func TestPipelineGate(t *testing.T) {
	t.Skip()
}

// this unit test ensures routes that require system access
// return a 403 forbidden if the user does not have acess
// to the pipeline
func TestSystemGate(t *testing.T) {
	t.Skip()
}
