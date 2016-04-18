// Copyright 2015 go-swagger maintainers
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

package spec

import (
	"encoding/json"
	"testing"

	testingutil "github.com/go-swagger/go-swagger/internal/testing"
	"github.com/stretchr/testify/assert"
)

func TestUnknownSpecVersion(t *testing.T) {
	_, err := New([]byte{}, "0.9")
	assert.Error(t, err)
}

func TestDefaultsTo20(t *testing.T) {
	d, err := New(testingutil.PetStoreJSONMessage, "")

	assert.NoError(t, err)
	assert.NotNil(t, d)
	assert.Equal(t, "2.0", d.Version())
	// assert.Equal(t, "2.0", d.data["swagger"].(string))
	assert.Equal(t, "/api", d.BasePath())
}

// func TestValidatesValidSchema(t *testing.T) {
// 	d, err := New(testingutil.PetStoreJSONMessage, "")

// 	assert.NoError(t, err)
// 	assert.NotNil(t, d)
// 	res := d.Validate()
// 	assert.NotNil(t, res)
// 	assert.True(t, res.Valid())
// 	assert.Empty(t, res.Errors())

// }

// func TestFailsInvalidSchema(t *testing.T) {
// 	d, err := New(testingutil.InvalidJSONMessage, "")

// 	assert.NoError(t, err)
// 	assert.NotNil(t, d)

// 	res := d.Validate()
// 	assert.NotNil(t, res)
// 	assert.False(t, res.Valid())
// 	assert.NotEmpty(t, res.Errors())
// }

func TestFailsInvalidJSON(t *testing.T) {
	_, err := New(json.RawMessage([]byte("{]")), "")

	assert.Error(t, err)
}
