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

package triage

import (
	"reflect"
	"testing"

	"github.com/harness/gitness/types"
)

func TestInAppPaths(t *testing.T) {
	ev := &types.InsightsEvent{Payload: `{"exception":{"values":[{"type":"E","value":"v","stacktrace":{"frames":[` +
		`{"filename":"vendor/lib.go","in_app":false},` +
		`{"filename":"./main.go","in_app":true},` +
		`{"filename":"pay/charge.go","in_app":true},` +
		`{"filename":"pay/charge.go","in_app":true},` +
		`{"filename":"/home/ci/src/x.go","in_app":true},` +
		`{"filename":"https://cdn.example.com/app.js","in_app":true}]}}]}}`}
	got := InAppPaths(ev)
	want := []string{"pay/charge.go", "main.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %v, want %v", got, want)
	}
	if InAppPaths(nil) != nil || InAppPaths(&types.InsightsEvent{Payload: "not json"}) != nil {
		t.Fatal("expected no paths")
	}
}
