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
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAuthSerialization(t *testing.T) {
	Convey("Auth should", t, func() {
		Convey("serialize", func() {
			Convey("basic auth security scheme", func() {
				auth := BasicAuth()
				So(auth, ShouldSerializeJSON, `{"type":"basic"}`)
			})

			Convey("header key model", func() {
				auth := APIKeyAuth("api-key", "header")
				So(auth, ShouldSerializeJSON, `{"type":"apiKey","name":"api-key","in":"header"}`)
			})

			Convey("oauth2 implicit flow model", func() {
				auth := OAuth2Implicit("http://foo.com/authorization")
				So(auth, ShouldSerializeJSON, `{"type":"oauth2","flow":"implicit","authorizationUrl":"http://foo.com/authorization"}`)
			})

			Convey("oauth2 password flow model", func() {
				auth := OAuth2Password("http://foo.com/token")
				So(auth, ShouldSerializeJSON, `{"type":"oauth2","flow":"password","tokenUrl":"http://foo.com/token"}`)
			})

			Convey("oauth2 application flow model", func() {
				auth := OAuth2Application("http://foo.com/token")
				So(auth, ShouldSerializeJSON, `{"type":"oauth2","flow":"application","tokenUrl":"http://foo.com/token"}`)
			})

			Convey("oauth2 access code flow model", func() {
				auth := OAuth2AccessToken("http://foo.com/authorization", "http://foo.com/token")
				So(auth, ShouldSerializeJSON, `{"type":"oauth2","flow":"accessCode","authorizationUrl":"http://foo.com/authorization","tokenUrl":"http://foo.com/token"}`)
			})

			Convey("oauth2 implicit flow model with scopes", func() {
				auth := OAuth2Implicit("http://foo.com/authorization")
				auth.AddScope("email", "read your email")
				So(auth, ShouldSerializeJSON, `{"type":"oauth2","flow":"implicit","authorizationUrl":"http://foo.com/authorization","scopes":{"email":"read your email"}}`)
			})

			Convey("oauth2 password flow model with scopes", func() {
				auth := OAuth2Password("http://foo.com/token")
				auth.AddScope("email", "read your email")
				So(auth, ShouldSerializeJSON, `{"type":"oauth2","flow":"password","tokenUrl":"http://foo.com/token","scopes":{"email":"read your email"}}`)
			})

			Convey("oauth2 application flow model with scopes", func() {
				auth := OAuth2Application("http://foo.com/token")
				auth.AddScope("email", "read your email")
				So(auth, ShouldSerializeJSON, `{"type":"oauth2","flow":"application","tokenUrl":"http://foo.com/token","scopes":{"email":"read your email"}}`)
			})

			Convey("oauth2 access code flow model with scopes", func() {
				auth := OAuth2AccessToken("http://foo.com/authorization", "http://foo.com/token")
				auth.AddScope("email", "read your email")
				So(auth, ShouldSerializeJSON, `{"type":"oauth2","flow":"accessCode","authorizationUrl":"http://foo.com/authorization","tokenUrl":"http://foo.com/token","scopes":{"email":"read your email"}}`)
			})
		})

		Convey("deserialize", func() {
			Convey("basic auth security scheme", func() {
				auth := BasicAuth()
				So(`{"type":"basic"}`, ShouldParseJSON, auth)
			})

			Convey("header key model", func() {
				auth := APIKeyAuth("api-key", "header")
				So(`{"in":"header","name":"api-key","type":"apiKey"}`, ShouldParseJSON, auth)
			})

			Convey("oauth2 implicit flow model", func() {
				auth := OAuth2Implicit("http://foo.com/authorization")
				So(`{"authorizationUrl":"http://foo.com/authorization","flow":"implicit","type":"oauth2"}`, ShouldParseJSON, auth)
			})

			Convey("oauth2 password flow model", func() {
				auth := OAuth2Password("http://foo.com/token")
				So(`{"flow":"password","tokenUrl":"http://foo.com/token","type":"oauth2"}`, ShouldParseJSON, auth)
			})

			Convey("oauth2 application flow model", func() {
				auth := OAuth2Application("http://foo.com/token")
				So(`{"flow":"application","tokenUrl":"http://foo.com/token","type":"oauth2"}`, ShouldParseJSON, auth)
			})

			Convey("oauth2 access code flow model", func() {
				auth := OAuth2AccessToken("http://foo.com/authorization", "http://foo.com/token")
				So(`{"authorizationUrl":"http://foo.com/authorization","flow":"accessCode","tokenUrl":"http://foo.com/token","type":"oauth2"}`, ShouldParseJSON, auth)
			})

			Convey("oauth2 implicit flow model with scopes", func() {
				auth := OAuth2Implicit("http://foo.com/authorization")
				auth.AddScope("email", "read your email")
				So(`{"authorizationUrl":"http://foo.com/authorization","flow":"implicit","scopes":{"email":"read your email"},"type":"oauth2"}`, ShouldParseJSON, auth)
			})

			Convey("oauth2 password flow model with scopes", func() {
				auth := OAuth2Password("http://foo.com/token")
				auth.AddScope("email", "read your email")
				So(`{"flow":"password","scopes":{"email":"read your email"},"tokenUrl":"http://foo.com/token","type":"oauth2"}`, ShouldParseJSON, auth)
			})

			Convey("oauth2 application flow model with scopes", func() {
				auth := OAuth2Application("http://foo.com/token")
				auth.AddScope("email", "read your email")
				So(`{"flow":"application","scopes":{"email":"read your email"},"tokenUrl":"http://foo.com/token","type":"oauth2"}`, ShouldParseJSON, auth)
			})

			Convey("oauth2 access code flow model with scopes", func() {
				auth := OAuth2AccessToken("http://foo.com/authorization", "http://foo.com/token")
				auth.AddScope("email", "read your email")
				So(`{"authorizationUrl":"http://foo.com/authorization","flow":"accessCode","scopes":{"email":"read your email"},"tokenUrl":"http://foo.com/token","type":"oauth2"}`, ShouldParseJSON, auth)
			})
		})
	})
}
