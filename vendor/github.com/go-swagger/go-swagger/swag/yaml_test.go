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

package swag

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type failJSONMarhal struct {
}

func (f failJSONMarhal) MarshalJSON() ([]byte, error) {
	return nil, errors.New("expected")
}

func TestLoadHTTPBytes(t *testing.T) {

	_, err := loadHTTPBytes("httx://12394:abd")
	assert.Error(t, err)

	serv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer serv.Close()

	_, err = loadHTTPBytes(serv.URL)
	assert.Error(t, err)

	ts2 := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("the content"))
	}))
	defer ts2.Close()

	d, err := loadHTTPBytes(ts2.URL)
	assert.NoError(t, err)
	assert.Equal(t, []byte("the content"), d)
}

func TestYAMLToJSON(t *testing.T) {

	data := make(map[interface{}]interface{})
	data[1] = "the int key value"
	data["name"] = "a string value"

	d, err := YAMLToJSON(data)
	assert.NoError(t, err)
	assert.Equal(t, []byte(`{"1":"the int key value","name":"a string value"}`), []byte(d))

	data[true] = "the bool value"
	d, err = YAMLToJSON(data)
	assert.Error(t, err)
	assert.Nil(t, d)

	delete(data, true)

	tag := make(map[interface{}]interface{})
	tag["name"] = "tag name"
	data["tag"] = tag

	d, err = YAMLToJSON(data)
	assert.NoError(t, err)
	assert.Equal(t, []byte(`{"1":"the int key value","name":"a string value","tag":{"name":"tag name"}}`), []byte(d))

	tag = make(map[interface{}]interface{})
	tag[true] = "bool tag name"
	data["tag"] = tag

	d, err = YAMLToJSON(data)
	assert.Error(t, err)
	assert.Nil(t, d)

	var lst []interface{}
	lst = append(lst, "hello")

	d, err = YAMLToJSON(lst)
	assert.NoError(t, err)
	assert.Equal(t, []byte(`["hello"]`), []byte(d))

	lst = append(lst, data)

	d, err = YAMLToJSON(lst)
	assert.Error(t, err)
	assert.Nil(t, d)

	// _, err := yamlToJSON(failJSONMarhal{})
	// assert.Error(t, err)

	_, err = bytesToYAMLDoc([]byte("- name: hello\n"))
	assert.Error(t, err)

	dd, err := bytesToYAMLDoc([]byte("description: 'object created'\n"))
	assert.NoError(t, err)

	d, err = YAMLToJSON(dd)
	assert.NoError(t, err)
	assert.Equal(t, json.RawMessage(`{"description":"object created"}`), d)

}
