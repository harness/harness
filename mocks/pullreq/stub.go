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

package pullreq

import (
	"testing"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/events"
)

// NewStubReporter returns a real *pullreqevents.Reporter backed by an in-memory
// events system. Use in tests that inject the full reporter type into handlers.
func NewStubReporter(t *testing.T) *pullreqevents.Reporter {
	t.Helper()

	eventSystem, err := events.ProvideSystem(events.Config{
		Mode:            events.ModeInMemory,
		MaxStreamLength: 1000,
	}, nil, events.NewNoopCollector())
	if err != nil {
		t.Fatalf("create in-memory event system: %v", err)
	}

	reporter, err := pullreqevents.NewReporter(eventSystem)
	if err != nil {
		t.Fatalf("create pull request event reporter: %v", err)
	}

	return reporter
}
