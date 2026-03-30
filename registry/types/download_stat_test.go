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

package types

import (
	"testing"
)

func TestDownloadCount_Identifier(t *testing.T) {
	dc := &DownloadCount{EntityID: 42, Count: 100}
	if dc.Identifier() != 42 {
		t.Errorf("expected 42, got %d", dc.Identifier())
	}
}

func TestManifestDownloadCount_Identifier(t *testing.T) {
	mdc := &ManifestDownloadCount{Key: "7:sha256:abc", Count: 55}
	if mdc.Identifier() != "7:sha256:abc" {
		t.Errorf("expected 7:sha256:abc, got %s", mdc.Identifier())
	}
}
