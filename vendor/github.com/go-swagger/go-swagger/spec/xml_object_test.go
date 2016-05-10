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

func TestXmlObject(t *testing.T) {

	Convey("an xml object should", t, func() {
		Convey("serialize", func() {
			Convey("an empty object", func() {
				obj := XMLObject{}
				expected := "{}"
				actual, err := json.Marshal(obj)
				So(err, ShouldBeNil)
				So(string(actual), ShouldEqual, expected)
			})
			Convey("a completed object", func() {
				obj := XMLObject{
					Name:      "the name",
					Namespace: "the namespace",
					Prefix:    "the prefix",
					Attribute: true,
					Wrapped:   true,
				}
				actual, err := json.Marshal(obj)
				So(err, ShouldBeNil)
				var ad map[string]interface{}
				err = json.Unmarshal(actual, &ad)
				So(err, ShouldBeNil)
				So(ad["name"], ShouldEqual, obj.Name)
				So(ad["namespace"], ShouldEqual, obj.Namespace)
				So(ad["prefix"], ShouldEqual, obj.Prefix)
				So(ad["attribute"], ShouldBeTrue)
				So(ad["wrapped"], ShouldBeTrue)
			})
		})
		Convey("deserialize", func() {
			Convey("an empty object", func() {
				expected := XMLObject{}
				actual := XMLObject{}
				err := json.Unmarshal([]byte("{}"), &actual)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, expected)
			})
			Convey("a completed object", func() {
				completed := `{"name":"the name","namespace":"the namespace","prefix":"the prefix","attribute":true,"wrapped":true}`
				expected := XMLObject{"the name", "the namespace", "the prefix", true, true}
				actual := XMLObject{}
				err := json.Unmarshal([]byte(completed), &actual)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, expected)
			})
		})
	})

}
