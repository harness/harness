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

package repo

import (
	"bytes"
	"testing"
)

func TestCountingWriter(t *testing.T) {
	var buf bytes.Buffer
	cw := &countingWriter{w: &buf}

	if _, err := cw.Write([]byte("abc")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := cw.Write([]byte("de")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cw.n != 5 {
		t.Errorf("expected 5 bytes counted, got %d", cw.n)
	}
	if buf.String() != "abcde" {
		t.Errorf("unexpected passthrough content %q", buf.String())
	}
}
