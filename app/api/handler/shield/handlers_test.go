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

package shield

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/harness/gitness/types/enum"
)

func TestParseFindingFilter(t *testing.T) {
	r := httptest.NewRequest("GET", "/?state=open,accepted_risk&kind=dependency&min_risk=70&page=2&limit=5", nil)
	f, err := ParseFindingFilter(r)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.States, []enum.ShieldFindingState{
		enum.ShieldFindingStateOpen, enum.ShieldFindingStateAcceptedRisk,
	}) || !reflect.DeepEqual(f.Kinds, []enum.ShieldFindingKind{enum.ShieldFindingKindDependency}) ||
		f.MinRiskScore != 70 || f.Page != 2 || f.Size != 5 {
		t.Fatalf("unexpected filter: %+v", f)
	}

	for _, q := range []string{"state=nope", "kind=nope", "min_risk=abc", "min_risk=101", "min_risk=-1"} {
		if _, err = ParseFindingFilter(httptest.NewRequest("GET", "/?"+q, nil)); err == nil {
			t.Errorf("%s: expected an error", q)
		}
	}

	f, err = ParseFindingFilter(httptest.NewRequest("GET", "/", nil))
	if err != nil || len(f.States) != 0 || f.MinRiskScore != 0 || f.Page != 1 {
		t.Fatalf("defaults: %+v %v", f, err)
	}
}
