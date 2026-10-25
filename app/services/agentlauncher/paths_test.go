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

package agentlauncher

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type fakeLinks struct{ links []*types.TaskLink }

func (f fakeLinks) ListByTask(context.Context, int64) ([]*types.TaskLink, error) { return f.links, nil }

type fakeEvents struct{ ev *types.InsightsEvent }

func (f fakeEvents) ListByIssue(context.Context, int64, int) ([]*types.InsightsEvent, error) {
	return []*types.InsightsEvent{f.ev}, nil
}

type fakeFinding struct{ f *types.ShieldFinding }

func (f fakeFinding) Find(context.Context, int64) (*types.ShieldFinding, error) { return f.f, nil }

type fakePulls struct{}

func (fakePulls) Find(_ context.Context, id int64) (*types.PullReq, error) {
	return &types.PullReq{ID: id, SourceSHA: "head", MergeBaseSHA: "base"}, nil
}

type fakePRFiles struct{ err error }

func (f fakePRFiles) ChangedFiles(context.Context, *types.PullReq) ([]string, error) {
	return []string{"api/handler.go", "go.mod"}, f.err
}

func TestStorePathResolver(t *testing.T) {
	payload := `{"exception":{"values":[{"type":"E","value":"v","stacktrace":{"frames":[` +
		`{"filename":"vendor/x.go","in_app":false},` +
		`{"filename":"main.go","in_app":true},` +
		`{"filename":"pay/charge.go","in_app":true},` +
		`{"filename":"/abs/machine/path.go","in_app":true}]}}]}}`
	r := StorePathResolver{
		Links: fakeLinks{links: []*types.TaskLink{
			{Kind: enum.TaskLinkKindInsightsIssue, RefID: "5"},
			{Kind: enum.TaskLinkKindShieldFinding, RefID: "6"},
			{Kind: enum.TaskLinkKindPullReq, RefID: "7"},
			{Kind: enum.TaskLinkKindAITask, RefID: "8"},
			{Kind: enum.TaskLinkKindPullReq, RefID: "not-a-number"},
		}},
		Events:   fakeEvents{ev: &types.InsightsEvent{Payload: payload}},
		Findings: fakeFinding{f: &types.ShieldFinding{ManifestPath: "go.mod"}},
		PullReqs: fakePulls{},
		PRFiles:  fakePRFiles{},
	}
	got, err := r.TaskPaths(context.Background(), &types.Task{ID: 1})
	if err != nil {
		t.Fatal(err)
	}
	// innermost in-app frame first, the manifest and the PR files, without duplicates
	want := []string{"pay/charge.go", "main.go", "go.mod", "api/handler.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %v, want %v", got, want)
	}
}

func TestStorePathResolverSkipsFailingPRFiles(t *testing.T) {
	r := StorePathResolver{
		Links:    fakeLinks{links: []*types.TaskLink{{Kind: enum.TaskLinkKindPullReq, RefID: "7"}}},
		PullReqs: fakePulls{},
		PRFiles:  fakePRFiles{err: errors.New("boom")},
	}
	got, err := r.TaskPaths(context.Background(), &types.Task{ID: 1})
	if err != nil || len(got) != 0 {
		t.Fatalf("paths=%v err=%v", got, err)
	}
}
