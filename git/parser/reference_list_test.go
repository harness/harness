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

package parser

import (
	"strings"
	"testing"

	"github.com/harness/gitness/git/sha"

	"golang.org/x/exp/maps"
)

func TestReferenceList(t *testing.T) {
	test := "" + "" +
		"3dffbe6490139d56d57a3bbb5f3a9a9e8cc316bb\trefs/heads/main\n" +
		"20e8b3475740f528f0b6f62d29ce5098ad491bfd\trefs/heads/master\n"

	m, err := ReferenceList(strings.NewReader(test))
	if err != nil {
		t.Errorf("failed with error: %s", err.Error())
		return
	}

	expected := map[string]sha.SHA{
		"refs/heads/main":   sha.Must("3dffbe6490139d56d57a3bbb5f3a9a9e8cc316bb"),
		"refs/heads/master": sha.Must("20e8b3475740f528f0b6f62d29ce5098ad491bfd"),
	}

	if !maps.Equal(m, expected) {
		t.Errorf("expected %v, got %v", expected, m)
	}
}
