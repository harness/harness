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

	. "github.com/smartystreets/goconvey/convey"
)

var paths = Paths{
	VendorExtensible: VendorExtensible{Extensions: map[string]interface{}{"x-framework": "go-swagger"}},
	Paths: map[string]PathItem{
		"/": PathItem{
			Refable: Refable{Ref: MustCreateRef("cats")},
		},
	},
}

var pathsJSON = `{"x-framework":"go-swagger","/":{"$ref":"cats"}}`

func TestIntegrationPaths(t *testing.T) {
	Convey("all fields of paths should", t, func() {

		Convey("serialize", func() {
			expected := map[string]interface{}{}
			json.Unmarshal([]byte(pathsJSON), &expected)
			b, err := json.Marshal(paths)
			So(err, ShouldBeNil)
			var actual map[string]interface{}
			err = json.Unmarshal(b, &actual)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})

		Convey("deserialize", func() {

			actual := Paths{}
			err := json.Unmarshal([]byte(pathsJSON), &actual)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, paths)
		})

	})
}
