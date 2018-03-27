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

package web

import (
	"testing"
)

func Test_injectPartials(t *testing.T) {
	got, want := injectPartials(before), after
	if got != want {
		t.Errorf("Want html %q, got %q", want, got)
	}
}

var before = `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<!-- drone:version -->
	<!-- drone:user -->
	<!-- drone:csrf -->
<link rel="shortcut icon" href="/favicon.png"></head>
<body>
</html>`

var after = `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	{{ template "version" . }}
	{{ template "user" . }}
	{{ template "csrf" . }}
<link rel="shortcut icon" href="/favicon.png"></head>
<body>
</html>`
