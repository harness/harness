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

package api

import (
	"context"
	"os"
	"testing"
)

// TestGetMergeBase_CanceledContext guards against a nil pointer dereference
// in GetMergeBase.
//
// cmd.Run returns the raw ctx.Err() (not wrapped as *command.Error) when the
// context is already done before/while the git process runs. GetMergeBase
// then does `command.AsError(err).IsExitCode(1)`, but AsError returns a nil
// *Error for an unwrapped error, so the method call on it panics
// (git/command/error.go:45, called from git/api/merge.go:83).
func TestGetMergeBase_CanceledContext(t *testing.T) {
	requireGit(t)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // simulate the context being canceled while merge-base is running, as happens under load

	g := &Git{}
	_, _, err = g.GetMergeBase(ctx, cwd, "origin", "main", "HEAD", false)
	if err == nil {
		t.Fatal("expected an error for a canceled context, got nil")
	}
}
