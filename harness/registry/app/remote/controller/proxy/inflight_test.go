// Source: https://github.com/goharbor/harbor

// Copyright 2016 Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInflightRequest(t *testing.T) {
	artName := "hello-world:latest"
	inflightChecker.addRequest(artName)
	_, ok := inflightChecker.reqMap[artName]
	assert.True(t, ok)
	inflightChecker.removeRequest(artName)
	_, exist := inflightChecker.reqMap[artName]
	assert.False(t, exist)
}
