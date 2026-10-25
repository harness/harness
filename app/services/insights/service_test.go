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
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/insights/ingest"
	"github.com/harness/gitness/app/services/insights/insightstest"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	testProjectID = 42
	testKey       = "abc123publickey" // the key and project id of the ingest fixtures
)

type recordingObserver struct {
	created, regressed []int64
}

func (o *recordingObserver) OnIssueCreated(_ context.Context, _ *types.InsightsProject, i *types.InsightsIssue,
	_ *types.InsightsEvent) error {
	o.created = append(o.created, i.ID)
	return nil
}

func (o *recordingObserver) OnIssueRegressed(_ context.Context, _ *types.InsightsProject, i *types.InsightsIssue,
	_ *types.InsightsEvent) error {
	o.regressed = append(o.regressed, i.ID)
	return nil
}

type env struct {
	svc   *Service
	st    *insightstest.Store
	obs   *recordingObserver
	clock time.Time
}

func newEnv(t *testing.T, mutate func(*types.InsightsProject)) *env {
	t.Helper()
	st := insightstest.New()
	p := &types.InsightsProject{
		ID: testProjectID, RepoID: 1, Identifier: "api", PublicKey: testKey, Enabled: true,
		RateLimitPerMinute: 600, SampleRate: 1,
	}
	if mutate != nil {
		mutate(p)
	}
	st.Projects[p.ID] = p

	e := &env{st: st, obs: &recordingObserver{}, clock: time.Unix(1_800_000_000, 0)}
	e.svc = NewService(Config{}, st.ProjectStore(), st.IssueStore(), st.EventStore())
	e.svc.SetObserver(e.obs)
	e.svc.now = func() time.Time { return e.clock }
	return e
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("ingest/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// envelopeFor builds an envelope with a single error event.
func envelopeFor(eventID, release, errType, value string) []byte {
	ev := fmt.Sprintf(`{"event_id":%q,"level":"error","platform":"python","release":%q,`+
		`"exception":{"values":[{"type":%q,"value":%q,"stacktrace":{"frames":[`+
		`{"module":"app.main","function":"run","in_app":true}]}}]}}`, eventID, release, errType, value)
	return []byte(fmt.Sprintf("{\"event_id\":%q}\n{\"type\":\"event\",\"length\":%d}\n%s\n", eventID, len(ev), ev))
}

func (e *env) ingestKey(body []byte, key string) (*IngestResult, error) {
	return e.svc.Ingest(context.Background(), IngestInput{
		ProjectID:  fmt.Sprint(testProjectID),
		SentryAuth: "Sentry sentry_version=7, sentry_key=" + key,
		Body:       bytes.NewReader(body),
	})
}

func (e *env) send(body []byte) (*IngestResult, error) { return e.ingestKey(body, testKey) }

func (e *env) onlyIssue(t *testing.T) *types.InsightsIssue {
	t.Helper()
	if len(e.st.Issues) != 1 {
		t.Fatalf("issues = %d, want 1", len(e.st.Issues))
	}
	for _, i := range e.st.Issues {
		return i
	}
	return nil
}

func eid(n int) string { return fmt.Sprintf("%032x", n) }

func TestIngest_FixturesCreateIssues(t *testing.T) {
	for _, name := range []string{"sentry_go_error.envelope", "sentry_javascript_error.envelope", "sentry_python_error.envelope"} {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t, nil)
			// the fixtures authenticate through the envelope header dsn.
			res, err := e.svc.Ingest(context.Background(), IngestInput{
				ProjectID: "42", Body: bytes.NewReader(fixture(t, name)),
			})
			if err != nil {
				t.Fatalf("Ingest() error = %v", err)
			}
			if res.Stored != 1 || res.EventID == "" {
				t.Fatalf("result = %+v", res)
			}
			i := e.onlyIssue(t)
			if i.TimesSeen != 1 || i.Title == "" || i.Status != enum.InsightsIssueStatusUnresolved {
				t.Errorf("issue = %+v", i)
			}
			if len(e.obs.created) != 1 {
				t.Errorf("created notifications = %d, want 1", len(e.obs.created))
			}
		})
	}
}

func TestIngest_SanitizesStoredPayload(t *testing.T) {
	e := newEnv(t, nil)
	if _, err := e.svc.Ingest(context.Background(), IngestInput{
		ProjectID: "42", Body: bytes.NewReader(fixture(t, "sentry_go_error.envelope")),
	}); err != nil {
		t.Fatal(err)
	}
	for _, ev := range e.st.Events {
		for _, secret := range []string{"Bearer secret", "sid=abc", "o:\\u0026{ID:987}"} {
			if strings.Contains(ev.Payload, secret) {
				t.Errorf("payload leaks %q", secret)
			}
		}
	}
}

func TestIngest_LegacyStore(t *testing.T) {
	e := newEnv(t, nil)
	res, err := e.svc.Ingest(context.Background(), IngestInput{
		ProjectID: "42", Legacy: true, Query: url.Values{"sentry_key": {testKey}},
		Body: bytes.NewReader(fixture(t, "store_legacy.json")),
	})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if res.Stored != 1 || res.EventID != "fc6d8c0c43fc4630ad850ee518f1b9d0" {
		t.Errorf("result = %+v", res)
	}
}

func TestIngest_DedupeAndCounting(t *testing.T) {
	e := newEnv(t, nil)
	if _, err := e.send(envelopeFor(eid(1), "1.0.0", "ValueError", "bad value 17")); err != nil {
		t.Fatal(err)
	}
	// same event id again: dropped.
	res, err := e.send(envelopeFor(eid(1), "1.0.0", "ValueError", "bad value 17"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Duplicates != 1 || res.Stored != 0 {
		t.Errorf("duplicate result = %+v", res)
	}
	// same error, different event id and value digits: same issue.
	if _, err = e.send(envelopeFor(eid(2), "1.0.1", "ValueError", "bad value 99")); err != nil {
		t.Fatal(err)
	}
	// different error type: new issue.
	if _, err = e.send(envelopeFor(eid(3), "1.0.1", "KeyError", "x")); err != nil {
		t.Fatal(err)
	}

	if len(e.st.Issues) != 2 || len(e.st.Events) != 3 {
		t.Fatalf("issues = %d events = %d, want 2 and 3", len(e.st.Issues), len(e.st.Events))
	}
	var value *types.InsightsIssue
	for _, i := range e.st.Issues {
		if strings.HasPrefix(i.Title, "ValueError") {
			value = i
		}
	}
	if value == nil || value.TimesSeen != 2 || value.FirstRelease != "1.0.0" || value.LastRelease != "1.0.1" {
		t.Errorf("value issue = %+v", value)
	}
	if len(e.obs.created) != 2 || len(e.obs.regressed) != 0 {
		t.Errorf("created = %v regressed = %v", e.obs.created, e.obs.regressed)
	}
}

func TestIngest_Regression(t *testing.T) {
	tests := []struct {
		name         string
		resolvedIn   string
		eventRelease string
		wantStatus   enum.InsightsIssueStatus
	}{
		{"later release", "1.1.0", "1.2.0", enum.InsightsIssueStatusRegressed},
		{"different non-semver release", "build-5", "build-6", enum.InsightsIssueStatusRegressed},
		{"same release stays resolved", "1.1.0", "1.1.0", enum.InsightsIssueStatusResolved},
		{"older release stays resolved", "1.1.0", "1.0.0", enum.InsightsIssueStatusResolved},
		{"no release on event stays resolved", "1.1.0", "", enum.InsightsIssueStatusResolved},
		{"resolved without release", "", "1.0.0", enum.InsightsIssueStatusRegressed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t, nil)
			if _, err := e.send(envelopeFor(eid(1), "1.0.0", "ValueError", "x")); err != nil {
				t.Fatal(err)
			}
			i := e.onlyIssue(t)
			i.Status, i.ResolvedInRelease = enum.InsightsIssueStatusResolved, tt.resolvedIn

			if _, err := e.send(envelopeFor(eid(2), tt.eventRelease, "ValueError", "x")); err != nil {
				t.Fatal(err)
			}
			if got := e.onlyIssue(t).Status; got != tt.wantStatus {
				t.Errorf("status = %s, want %s", got, tt.wantStatus)
			}
			wantNotified := tt.wantStatus == enum.InsightsIssueStatusRegressed
			if got := len(e.obs.regressed) == 1; got != wantNotified {
				t.Errorf("regressed notifications = %v, want notified=%v", e.obs.regressed, wantNotified)
			}

			// a further event of a regressed issue does not notify again.
			if _, err := e.send(envelopeFor(eid(3), "9.9.9", "ValueError", "x")); err != nil {
				t.Fatal(err)
			}
			if len(e.obs.regressed) > 1 {
				t.Errorf("regressed notified %d times", len(e.obs.regressed))
			}
		})
	}
}

func TestIngest_IgnoredIssueStaysIgnored(t *testing.T) {
	e := newEnv(t, nil)
	if _, err := e.send(envelopeFor(eid(1), "1.0.0", "ValueError", "x")); err != nil {
		t.Fatal(err)
	}
	e.onlyIssue(t).Status = enum.InsightsIssueStatusIgnored
	if _, err := e.send(envelopeFor(eid(2), "2.0.0", "ValueError", "x")); err != nil {
		t.Fatal(err)
	}
	if got := e.onlyIssue(t).Status; got != enum.InsightsIssueStatusIgnored {
		t.Errorf("status = %s", got)
	}
}

func TestIngest_BadKey(t *testing.T) {
	body := envelopeFor(eid(1), "1", "E", "v")

	t.Run("wrong key", func(t *testing.T) {
		e := newEnv(t, nil)
		if _, err := e.ingestKey(body, "nope"); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("error = %v, want ErrInvalidKey", err)
		}
		if len(e.st.Events) != 0 {
			t.Error("event stored")
		}
	})
	t.Run("unknown project", func(t *testing.T) {
		e := newEnv(t, nil)
		_, err := e.svc.Ingest(context.Background(), IngestInput{
			ProjectID: "7", SentryAuth: "Sentry sentry_key=" + testKey, Body: bytes.NewReader(body),
		})
		if !errors.Is(err, ErrInvalidKey) {
			t.Errorf("error = %v, want ErrInvalidKey", err)
		}
	})
	t.Run("non numeric project", func(t *testing.T) {
		e := newEnv(t, nil)
		_, err := e.svc.Ingest(context.Background(), IngestInput{
			ProjectID: "api", SentryAuth: "Sentry sentry_key=" + testKey, Body: bytes.NewReader(body),
		})
		if !errors.Is(err, ErrInvalidKey) {
			t.Errorf("error = %v, want ErrInvalidKey", err)
		}
	})
	t.Run("disabled project", func(t *testing.T) {
		e := newEnv(t, func(p *types.InsightsProject) { p.Enabled = false })
		if _, err := e.send(body); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("error = %v, want ErrInvalidKey", err)
		}
	})
	t.Run("missing key", func(t *testing.T) {
		e := newEnv(t, nil)
		_, err := e.svc.Ingest(context.Background(), IngestInput{ProjectID: "42", Body: bytes.NewReader(body)})
		if !errors.Is(err, ingest.ErrMissingKey) {
			t.Errorf("error = %v, want ErrMissingKey", err)
		}
	})
	t.Run("dsn of another project", func(t *testing.T) {
		e := newEnv(t, nil)
		_, err := e.svc.Ingest(context.Background(), IngestInput{
			ProjectID: "43", Body: bytes.NewReader(fixture(t, "sentry_go_error.envelope")),
		})
		if !errors.Is(err, ingest.ErrProjectMismatch) {
			t.Errorf("error = %v, want ErrProjectMismatch", err)
		}
	})
}

func TestIngest_RateLimit(t *testing.T) {
	e := newEnv(t, func(p *types.InsightsProject) { p.RateLimitPerMinute = 3 })

	for n := 1; n <= 3; n++ {
		if _, err := e.send(envelopeFor(eid(n), "1", "E", "v")); err != nil {
			t.Fatalf("event %d: %v", n, err)
		}
	}
	_, err := e.send(envelopeFor(eid(4), "1", "E", "v"))
	var rl *RateLimitedError
	if !errors.As(err, &rl) {
		t.Fatalf("error = %v, want RateLimitedError", err)
	}
	if rl.RetryAfter < time.Second || rl.RetryAfter > 30*time.Second {
		t.Errorf("RetryAfter = %v", rl.RetryAfter)
	}
	if len(e.st.Events) != 3 {
		t.Errorf("events = %d, want 3", len(e.st.Events))
	}

	// after the retry period the project can send again.
	e.clock = e.clock.Add(rl.RetryAfter)
	if _, err = e.send(envelopeFor(eid(5), "1", "E", "v")); err != nil {
		t.Errorf("after refill: %v", err)
	}
}

func TestIngest_RateLimitIsPerProject(t *testing.T) {
	e := newEnv(t, func(p *types.InsightsProject) { p.RateLimitPerMinute = 1 })
	e.st.Projects[43] = &types.InsightsProject{
		ID: 43, RepoID: 1, Identifier: "web", PublicKey: "other", Enabled: true, RateLimitPerMinute: 1, SampleRate: 1,
	}

	if _, err := e.send(envelopeFor(eid(1), "1", "E", "v")); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Ingest(context.Background(), IngestInput{
		ProjectID: "43", SentryAuth: "Sentry sentry_key=other", Body: bytes.NewReader(envelopeFor(eid(2), "1", "E", "v")),
	}); err != nil {
		t.Errorf("second project is limited by the first: %v", err)
	}
}

func TestIngest_Sampling(t *testing.T) {
	e := newEnv(t, func(p *types.InsightsProject) { p.SampleRate = 0.5 })

	e.svc.rand = func() float64 { return 0.9 } // >= rate: dropped
	res, err := e.send(envelopeFor(eid(1), "1", "E", "v"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Sampled != 1 || res.Stored != 0 || res.EventID != eid(1) || len(e.st.Events) != 0 {
		t.Errorf("sampled out result = %+v events = %d", res, len(e.st.Events))
	}

	e.svc.rand = func() float64 { return 0.1 } // < rate: kept
	res, err = e.send(envelopeFor(eid(2), "1", "E", "v"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Stored != 1 || len(e.st.Events) != 1 {
		t.Errorf("kept result = %+v", res)
	}
}

func TestIngest_SampleRateZeroDropsEverything(t *testing.T) {
	e := newEnv(t, func(p *types.InsightsProject) { p.SampleRate = 0 })
	e.svc.rand = func() float64 { return 0 }
	res, err := e.send(envelopeFor(eid(1), "1", "E", "v"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Stored != 0 || res.Sampled != 1 {
		t.Errorf("result = %+v", res)
	}
}

func TestIngest_PayloadCap(t *testing.T) {
	e := newEnv(t, nil)
	e.svc.cfg.MaxPayloadBytes = 600

	big := strings.Repeat("x", 5000)
	ev := fmt.Sprintf(`{"event_id":%q,"message":"boom","level":"error","extra":{"blob":%q}}`, eid(1), big)
	body := fmt.Sprintf("{}\n{\"type\":\"event\",\"length\":%d}\n%s\n", len(ev), ev)
	if _, err := e.send([]byte(body)); err != nil {
		t.Fatal(err)
	}
	for _, rec := range e.st.Events {
		if len(rec.Payload) > 600 || !strings.Contains(rec.Payload, `"message":"boom"`) {
			t.Errorf("payload len = %d: %.80s", len(rec.Payload), rec.Payload)
		}
	}
}

func TestIngest_NonEventEnvelope(t *testing.T) {
	e := newEnv(t, nil)
	body := "{\"event_id\":\"" + eid(9) + "\"}\n{\"type\":\"session\"}\n{\"sid\":\"x\"}\n"
	res, err := e.send([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if res.Stored != 0 || res.EventID != eid(9) || len(e.st.Events) != 0 {
		t.Errorf("result = %+v", res)
	}
}

func TestIngest_TooLarge(t *testing.T) {
	e := newEnv(t, nil)
	e.svc.cfg.Limits = ingest.Limits{MaxCompressedSize: 100}
	_, err := e.send(envelopeFor(eid(1), "1", "E", strings.Repeat("v", 500)))
	if !errors.Is(err, ingest.ErrTooLarge) {
		t.Errorf("error = %v, want ErrTooLarge", err)
	}
}

func TestIngest_UsersSeen(t *testing.T) {
	e := newEnv(t, nil)
	mk := func(n int, user string) []byte {
		ev := fmt.Sprintf(`{"event_id":%q,"exception":{"values":[{"type":"E","value":"v"}]},"user":{"id":%q}}`, eid(n), user)
		return []byte(fmt.Sprintf("{}\n{\"type\":\"event\",\"length\":%d}\n%s\n", len(ev), ev))
	}
	for n, u := range []string{"a", "a", "b"} {
		if _, err := e.send(mk(n+1, u)); err != nil {
			t.Fatal(err)
		}
	}
	if i := e.onlyIssue(t); i.UsersSeen != 2 || i.TimesSeen != 3 {
		t.Errorf("users_seen = %d times_seen = %d, want 2 and 3", i.UsersSeen, i.TimesSeen)
	}
}

func TestCreateProject(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()

	p, err := e.svc.CreateProject(ctx, 5, 1, &types.InsightsProjectCreateInput{Identifier: "web"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.PublicKey) != 32 || !p.Enabled || p.SampleRate != 1 || p.RateLimitPerMinute != DefaultRateLimitPerMinute {
		t.Errorf("project = %+v", p)
	}
	if _, err = e.svc.CreateProject(ctx, 5, 1, &types.InsightsProjectCreateInput{Identifier: "web"}); err == nil {
		t.Error("duplicate identifier accepted")
	}
	bad := 2.0
	for _, in := range []*types.InsightsProjectCreateInput{
		{Identifier: ""}, {Identifier: "bad name"}, {Identifier: "ok", SampleRate: &bad},
	} {
		if _, err = e.svc.CreateProject(ctx, 5, 1, in); err == nil {
			t.Errorf("input %+v accepted", in)
		}
	}
	list, err := e.svc.ListProjects(ctx, 5)
	if err != nil || len(list) != 1 {
		t.Errorf("list = %v err = %v", list, err)
	}
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		a, b   string
		want   int
		wantOK bool
	}{
		{"1.2.3", "1.2.4", -1, true},
		{"v2.0", "1.9.9", 1, true},
		{"app@1.2.3+build5", "1.2.3", 0, true},
		{"1.0.0-rc1", "1.0.0", 0, true},
		{"web@2026.03.01", "web@2026.02.01", 1, true},
		{"build-5", "1.0.0", 0, false},
		{"abc", "def", 0, false},
	}
	for _, tt := range tests {
		got, ok := compareSemver(tt.a, tt.b)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("compareSemver(%q,%q) = %d,%v want %d,%v", tt.a, tt.b, got, ok, tt.want, tt.wantOK)
		}
	}
}
