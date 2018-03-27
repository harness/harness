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

package model

import (
	"crypto/rand"
	"fmt"
	"testing"
)

func TestBuildTrim(t *testing.T) {
	d := make([]byte, 2000)
	rand.Read(d)

	b := Build{}
	b.Message = fmt.Sprintf("%X", d)

	if len(b.Message) != 4000 {
		t.Errorf("Failed to generate 4000 byte test string")
	}
	b.Trim()
	if len(b.Message) != 2000 {
		t.Errorf("Failed to trim text string to 2000 bytes")
	}
}
