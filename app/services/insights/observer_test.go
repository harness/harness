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

package insights

import (
	"context"
	"testing"

	"github.com/harness/gitness/types"
)

func TestService_AddObserverComposes(t *testing.T) {
	a, b, c := &recordingObserver{}, &recordingObserver{}, &recordingObserver{}
	s := NewService(Config{}, nil, nil, nil)
	s.AddObserver(a)
	s.AddObserver(b)
	s.AddObserver(c)

	issue := &types.InsightsIssue{ID: 4}
	if err := s.getObserver().OnIssueRegressed(context.Background(), nil, issue, nil); err != nil {
		t.Fatal(err)
	}
	for i, o := range []*recordingObserver{a, b, c} {
		if len(o.regressed) != 1 {
			t.Errorf("observer %d regressed calls = %d, want 1", i, len(o.regressed))
		}
	}

	s.SetObserver(a)
	s.AddObserver(nil)
	if _, ok := s.getObserver().(*recordingObserver); !ok {
		t.Errorf("SetObserver should replace the composite")
	}
}
