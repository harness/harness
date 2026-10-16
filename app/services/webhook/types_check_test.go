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

package webhook

import (
	"encoding/json"
	"testing"

	"github.com/harness/gitness/types/enum"
)

func TestCheckInfoJSON(t *testing.T) {
	raw, err := json.Marshal(CheckInfo{
		Identifier: "ci",
		Name:       "ci",
		Status:     enum.CheckStatusSuccess,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(raw); got != `{"identifier":"ci","name":"ci","status":"success"}` {
		t.Fatalf("unexpected json: %s", got)
	}
}
