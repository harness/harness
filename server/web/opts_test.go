// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package web

import (
	"testing"
	"time"
)

func TestWithSync(t *testing.T) {
	opts := new(Options)
	WithSync(time.Minute)(opts)
	if got, want := opts.sync, time.Minute; got != want {
		t.Errorf("Want sync duration %v, got %v", want, got)
	}
}

func TestWithDir(t *testing.T) {
	opts := new(Options)
	WithDir("/tmp/www")(opts)
	if got, want := opts.path, "/tmp/www"; got != want {
		t.Errorf("Want www directory %q, got %q", want, got)
	}
}

func TestWithDocs(t *testing.T) {
	opts := new(Options)
	WithDocs("http://docs.drone.io")(opts)
	if got, want := opts.docs, "http://docs.drone.io"; got != want {
		t.Errorf("Want documentation url %q, got %q", want, got)
	}
}
