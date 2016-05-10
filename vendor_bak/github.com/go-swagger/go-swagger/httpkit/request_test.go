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

package httpkit

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONRequest(t *testing.T) {
	req, err := JSONRequest("GET", "/swagger.json", nil)
	assert.NoError(t, err)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, JSONMime, req.Header.Get(HeaderContentType))
	assert.Equal(t, JSONMime, req.Header.Get(HeaderAccept))

	req, err = JSONRequest("GET", "%2", nil)
	assert.Error(t, err)
	assert.Nil(t, req)
}

func TestCanHaveBody(t *testing.T) {
	assert.True(t, CanHaveBody("put"))
	assert.True(t, CanHaveBody("post"))
	assert.True(t, CanHaveBody("patch"))
	assert.True(t, CanHaveBody("delete"))
	assert.False(t, CanHaveBody(""))
	assert.False(t, CanHaveBody("get"))
	assert.False(t, CanHaveBody("options"))
	assert.False(t, CanHaveBody("head"))
	assert.False(t, CanHaveBody("invalid"))
}

func TestReadSingle(t *testing.T) {
	values := url.Values(make(map[string][]string))
	values.Add("something", "the thing")
	assert.Equal(t, "the thing", ReadSingleValue(tv(values), "something"))
	assert.Empty(t, ReadSingleValue(tv(values), "notthere"))
}

func TestReadCollection(t *testing.T) {
	values := url.Values(make(map[string][]string))
	values.Add("something", "value1,value2")
	assert.Equal(t, []string{"value1", "value2"}, ReadCollectionValue(tv(values), "something", "csv"))
	assert.Empty(t, ReadCollectionValue(tv(values), "notthere", ""))
}

type tv map[string][]string

func (v tv) GetOK(key string) (value []string, hasKey bool, hasValue bool) {
	value, hasKey = v[key]
	if !hasKey {
		return
	}
	if len(value) == 0 {
		return
	}
	hasValue = true
	return

}
