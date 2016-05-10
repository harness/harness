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
	"gopkg.in/yaml.v2"
)

var contactInfoJSON = `{"name":"wordnik api team","url":"http://developer.wordnik.com","email":"some@mailayada.dkdkd"}`
var contactInfoYAML = `name: wordnik api team
url: http://developer.wordnik.com
email: some@mailayada.dkdkd
`
var contactInfo = ContactInfo{
	Name:  "wordnik api team",
	URL:   "http://developer.wordnik.com",
	Email: "some@mailayada.dkdkd",
}

func TestIntegrationContactInfo(t *testing.T) {
	Convey("all fields of contact info should", t, func() {
		Convey("serialize to JSON", func() {
			b, err := json.Marshal(contactInfo)
			So(err, ShouldBeNil)
			So(string(b), ShouldEqual, contactInfoJSON)
		})

		Convey("serialize to YAML", func() {
			b, err := yaml.Marshal(contactInfo)
			So(err, ShouldBeNil)
			So(string(b), ShouldEqual, contactInfoYAML)
		})

		Convey("deserialize from JSON", func() {
			actual := ContactInfo{}
			err := json.Unmarshal([]byte(contactInfoJSON), &actual)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, contactInfo)
		})

		Convey("deserialize from YAML", func() {
			actual := ContactInfo{}
			err := yaml.Unmarshal([]byte(contactInfoYAML), &actual)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, contactInfo)
		})
	})
}
