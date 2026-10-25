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

// Package insights receives error events from Sentry compatible SDKs,
// groups them into issues and stores them.
package insights

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/harness/gitness/app/services/insights/ingest"
	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	// DefaultMaxPayloadBytes caps the size of the stored sanitized event JSON.
	DefaultMaxPayloadBytes = 64 << 10

	maxTitleLength   = 200
	maxCulpritLength = 200
	maxUpsertRetries = 5
)

var (
	// ErrInvalidKey is returned for unknown projects, wrong keys and disabled projects.
	// The causes are not distinguished so callers cannot probe for projects.
	ErrInvalidKey = errors.New("insights: invalid project or key")
)

// RateLimitedError is returned when the project exceeded its event rate.
type RateLimitedError struct {
	RetryAfter time.Duration
}

func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("insights: rate limit exceeded, retry after %s", e.RetryAfter)
}

// Config configures the service.
type Config struct {
	// Limits bound the size of requests, see ingest.Limits.
	Limits ingest.Limits
	// MaxPayloadBytes caps the stored event JSON (default DefaultMaxPayloadBytes).
	MaxPayloadBytes int
}

// IngestInput is one SDK request.
type IngestInput struct {
	// ProjectID is the project id from the request path.
	ProjectID string
	// SentryAuth is the X-Sentry-Auth header value.
	SentryAuth string
	// Query holds the URL query parameters (sentry_key).
	Query url.Values
	// Body is the raw request body.
	Body io.Reader
	// ContentEncoding is the Content-Encoding header.
	ContentEncoding string
	// Legacy is true for the store endpoint (single JSON event instead of an envelope).
	Legacy bool
}

// IngestResult is the outcome of an accepted request.
type IngestResult struct {
	// EventID is the id of the first event of the request (or the envelope header id).
	EventID string
	// Stored is the number of events that were stored.
	Stored int
	// Duplicates is the number of events that were already known.
	Duplicates int
	// Sampled is the number of events dropped by sampling.
	Sampled int
}

// Service ingests events.
type Service struct {
	cfg      Config
	projects store.InsightsProjectStore
	issues   store.InsightsIssueStore
	events   store.InsightsEventStore

	limiter *rateLimiter

	mu       sync.RWMutex
	observer IssueObserver

	now  func() time.Time
	rand func() float64
}

// NewService returns a new Service.
func NewService(
	cfg Config,
	projects store.InsightsProjectStore,
	issues store.InsightsIssueStore,
	events store.InsightsEventStore,
) *Service {
	if cfg.MaxPayloadBytes <= 0 {
		cfg.MaxPayloadBytes = DefaultMaxPayloadBytes
	}
	return &Service{
		cfg:      cfg,
		projects: projects,
		issues:   issues,
		events:   events,
		limiter:  newRateLimiter(),
		observer: NoopObserver{},
		now:      time.Now,
		rand:     secureFloat,
	}
}

// SetObserver replaces the issue observer (nil resets to the no-op observer).
func (s *Service) SetObserver(o IssueObserver) {
	if o == nil {
		o = NoopObserver{}
	}
	s.mu.Lock()
	s.observer = o
	s.mu.Unlock()
}

func (s *Service) getObserver() IssueObserver {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.observer
}

// Ingest authenticates, rate limits, samples and stores the events of one SDK request.
// Errors: ErrInvalidKey (403), *RateLimitedError (429), ingest.ErrTooLarge (413),
// other ingest client errors (400); anything else is an internal error.
//
//nolint:gocognit,funlen // linear pipeline, splitting it would not help readability
func (s *Service) Ingest(ctx context.Context, in IngestInput) (*IngestResult, error) {
	lim := s.cfg.Limits

	body, err := ingest.ReadBody(in.Body, in.ContentEncoding, lim)
	if err != nil {
		return nil, err
	}

	var (
		events    []*ingest.Event
		headerDSN string
		headerID  string
	)
	if in.Legacy {
		ev, perr := ingest.ParseStore(bytes.NewReader(body), "identity", lim)
		if perr != nil {
			return nil, perr
		}
		events = []*ingest.Event{ev}
	} else {
		env, perr := ingest.ParseEnvelopeBytes(body, lim)
		if perr != nil {
			return nil, perr
		}
		if len(env.Events) == 0 && len(env.ItemErrors) > 0 {
			return nil, fmt.Errorf("%w: %v", ingest.ErrInvalidEvent, env.ItemErrors[0].Err)
		}
		events, headerDSN, headerID = env.Events, env.Header.DSN, env.Header.EventID
	}

	project, err := s.authenticate(ctx, in, headerDSN)
	if err != nil {
		return nil, err
	}

	res := &IngestResult{EventID: normalizeEventID(headerID)}
	if len(events) == 0 {
		return res, nil // nothing we store (transactions, sessions, ...)
	}

	if ok, wait := s.limiter.allow(project.ID, project.RateLimitPerMinute, len(events), s.now()); !ok {
		return nil, &RateLimitedError{RetryAfter: wait}
	}

	for _, ev := range events {
		ev.EventID = normalizeEventID(ev.EventID)
		if ev.EventID == "" {
			ev.EventID = newEventID()
		}
		if res.EventID == "" {
			res.EventID = ev.EventID
		}

		if project.SampleRate < 1 && s.rand() >= project.SampleRate {
			res.Sampled++
			continue
		}

		stored, serr := s.store(ctx, project, ev)
		if serr != nil {
			return nil, serr
		}
		if stored {
			res.Stored++
		} else {
			res.Duplicates++
		}
	}

	return res, nil
}

func (s *Service) authenticate(ctx context.Context, in IngestInput, envelopeDSN string) (*types.InsightsProject, error) {
	creds, err := ingest.Authenticate(ingest.AuthInput{
		ProjectID:   in.ProjectID,
		SentryAuth:  in.SentryAuth,
		Query:       in.Query,
		EnvelopeDSN: envelopeDSN,
	})
	if err != nil {
		return nil, err
	}

	id, perr := strconv.ParseInt(creds.ProjectID, 10, 64)
	if perr != nil || id <= 0 {
		_ = ingest.ConstantTimeEqual(creds.PublicKey, creds.PublicKey)
		return nil, ErrInvalidKey
	}

	project, err := s.projects.Find(ctx, id)
	if err != nil {
		if errors.Is(err, gitness_store.ErrResourceNotFound) {
			// compare anyway to keep timing independent of project existence.
			_ = ingest.ConstantTimeEqual(creds.PublicKey, creds.PublicKey)
			return nil, ErrInvalidKey
		}
		return nil, fmt.Errorf("failed to find insights project: %w", err)
	}

	if !ingest.ConstantTimeEqual(creds.PublicKey, project.PublicKey) || !project.Enabled {
		return nil, ErrInvalidKey
	}

	return project, nil
}

// store sanitizes and persists one event, updating its issue. It returns false
// if the event id was stored before.
func (s *Service) store(ctx context.Context, project *types.InsightsProject, ev *ingest.Event) (bool, error) {
	exists, err := s.events.Exists(ctx, project.ID, ev.EventID)
	if err != nil {
		return false, fmt.Errorf("failed to check insights event: %w", err)
	}
	if exists {
		return false, nil
	}

	// the fingerprint must be computed on the original event; sanitizing only drops sensitive data.
	fp := ingest.ComputeFingerprint(ev).Hash
	ingest.Sanitize(ev, ingest.DefaultSanitizeOptions())

	now := s.now().UnixMilli()
	ts := now
	if !ev.Timestamp.IsZero() {
		ts = ev.Timestamp.UnixMilli()
	}

	rec := &types.InsightsEvent{
		ProjectID:   project.ID,
		EventID:     ev.EventID,
		Timestamp:   ts,
		Release:     ev.Release,
		Environment: ev.Environment,
		Level:       ev.Level,
		Message:     truncate(eventMessage(ev), 1000),
		Payload:     s.payload(ev),
		Created:     now,
	}

	issue, transition, err := s.upsertIssue(ctx, project, ev, fp, ts, now)
	if err != nil {
		return false, err
	}
	rec.IssueID = issue.ID

	if err = s.events.Create(ctx, rec); err != nil {
		if errors.Is(err, gitness_store.ErrDuplicate) {
			return false, nil // concurrent delivery of the same event
		}
		return false, fmt.Errorf("failed to store insights event: %w", err)
	}

	obs := s.getObserver()
	switch transition {
	case transitionCreated:
		if oerr := obs.OnIssueCreated(ctx, project, issue, rec); oerr != nil {
			log.Ctx(ctx).Warn().Err(oerr).Int64("issue_id", issue.ID).Msg("insights: issue observer failed")
		}
	case transitionRegressed:
		if oerr := obs.OnIssueRegressed(ctx, project, issue, rec); oerr != nil {
			log.Ctx(ctx).Warn().Err(oerr).Int64("issue_id", issue.ID).Msg("insights: issue observer failed")
		}
	case transitionNone:
	}

	return true, nil
}

type transition int

const (
	transitionNone transition = iota
	transitionCreated
	transitionRegressed
)

// upsertIssue creates the issue of the fingerprint or updates it. Concurrent
// writers are handled with retries (unique fingerprint, optimistic version).
func (s *Service) upsertIssue(
	ctx context.Context,
	project *types.InsightsProject,
	ev *ingest.Event,
	fingerprint string,
	ts, now int64,
) (*types.InsightsIssue, transition, error) {
	userKey := userKey(ev)

	for range maxUpsertRetries {
		issue, err := s.issues.FindByFingerprint(ctx, project.ID, fingerprint)
		if errors.Is(err, gitness_store.ErrResourceNotFound) {
			issue = &types.InsightsIssue{
				ProjectID:    project.ID,
				RepoID:       project.RepoID,
				Fingerprint:  fingerprint,
				Title:        truncate(issueTitle(ev), maxTitleLength),
				Culprit:      truncate(culprit(ev), maxCulpritLength),
				Level:        ev.Level,
				Platform:     ev.Platform,
				FirstSeen:    ts,
				LastSeen:     ts,
				TimesSeen:    1,
				LastUser:     userKey,
				FirstRelease: ev.Release,
				LastRelease:  ev.Release,
				Status:       enum.InsightsIssueStatusUnresolved,
				Created:      now,
				Updated:      now,
			}
			if userKey != "" {
				issue.UsersSeen = 1
			}
			if err = s.issues.Create(ctx, issue); err == nil {
				return issue, transitionCreated, nil
			}
			if errors.Is(err, gitness_store.ErrDuplicate) {
				continue // lost the race, update the winner's issue
			}
			return nil, transitionNone, fmt.Errorf("failed to create insights issue: %w", err)
		}
		if err != nil {
			return nil, transitionNone, fmt.Errorf("failed to find insights issue: %w", err)
		}

		tr := transitionNone
		issue.TimesSeen++
		if ts > issue.LastSeen {
			issue.LastSeen = ts
		}
		if ev.Release != "" {
			issue.LastRelease = ev.Release
		}
		if userKey != "" && userKey != issue.LastUser {
			issue.UsersSeen++
			issue.LastUser = userKey
		}
		if ev.Level != "" {
			issue.Level = ev.Level
		}
		if isRegression(issue, ev.Release) {
			issue.Status = enum.InsightsIssueStatusRegressed
			tr = transitionRegressed
		}
		issue.Updated = now

		if err = s.issues.Update(ctx, issue); err != nil {
			if errors.Is(err, gitness_store.ErrVersionConflict) {
				continue
			}
			return nil, transitionNone, fmt.Errorf("failed to update insights issue: %w", err)
		}
		return issue, tr, nil
	}

	return nil, transitionNone, errors.New("insights: too many concurrent updates of the issue")
}

// payload serializes the sanitized event, dropping bulky parts until it fits the cap.
func (s *Service) payload(ev *ingest.Event) string {
	b, err := json.Marshal(ev)
	if err == nil && len(b) <= s.cfg.MaxPayloadBytes {
		return string(b)
	}

	slim := *ev
	slim.Breadcrumbs, slim.Extra, slim.Contexts, slim.Modules, slim.Request = nil, nil, nil, nil, nil
	if b, err = json.Marshal(&slim); err == nil && len(b) <= s.cfg.MaxPayloadBytes {
		return string(b)
	}

	// keep the identifying parts only.
	min := map[string]any{
		"event_id": ev.EventID, "platform": ev.Platform, "level": ev.Level, "release": ev.Release,
		"environment": ev.Environment, "message": truncate(eventMessage(ev), 1000), "truncated": true,
	}
	b, _ = json.Marshal(min)
	return string(b)
}

// isRegression reports whether an event of the release reopens a resolved issue.
// Resolved without a release regresses on any event. With a release, an event
// of a different release regresses unless both are semantic versions and the
// event's release is older than the fix.
func isRegression(i *types.InsightsIssue, eventRelease string) bool {
	if i.Status != enum.InsightsIssueStatusResolved {
		return false
	}
	if i.ResolvedInRelease == "" {
		return true
	}
	if eventRelease == "" || eventRelease == i.ResolvedInRelease {
		return false
	}
	if c, ok := compareSemver(eventRelease, i.ResolvedInRelease); ok && c < 0 {
		return false // an older release still running in the field
	}
	return true
}

func eventMessage(ev *ingest.Event) string {
	if m := ev.FormattedMessage(); m != "" {
		return m
	}
	if vals := ev.ExceptionValues(); len(vals) > 0 {
		last := vals[len(vals)-1]
		return strings.TrimSpace(last.Type + ": " + last.Value)
	}
	return ""
}

func issueTitle(ev *ingest.Event) string {
	if vals := ev.ExceptionValues(); len(vals) > 0 {
		last := vals[len(vals)-1]
		switch {
		case last.Type != "" && last.Value != "":
			return last.Type + ": " + firstLine(last.Value)
		case last.Type != "":
			return last.Type
		case last.Value != "":
			return firstLine(last.Value)
		}
	}
	if m := ev.MessageTemplate(); m != "" {
		return firstLine(m)
	}
	if m := ev.FormattedMessage(); m != "" {
		return firstLine(m)
	}
	return "<unlabeled event>"
}

// culprit is the transaction, else the innermost in-app frame (or the innermost frame).
func culprit(ev *ingest.Event) string {
	if ev.Transaction != "" {
		return ev.Transaction
	}
	vals := ev.ExceptionValues()
	for i := len(vals) - 1; i >= 0; i-- {
		if vals[i].Stacktrace == nil {
			continue
		}
		frames := vals[i].Stacktrace.Frames
		best := -1
		for j := len(frames) - 1; j >= 0; j-- {
			if frames[j].InApp != nil && *frames[j].InApp {
				best = j
				break
			}
		}
		if best < 0 && len(frames) > 0 {
			best = len(frames) - 1
		}
		if best >= 0 {
			f := frames[best]
			loc := f.Module
			if loc == "" {
				loc = f.Filename
			}
			if f.Function != "" {
				return loc + " in " + f.Function
			}
			return loc
		}
	}
	return ""
}

// userKey is a short hash of the user identity (no PII is stored on the issue).
func userKey(ev *ingest.Event) string {
	if ev.User == nil {
		return ""
	}
	id := ev.User.ID
	if id == "" {
		id = ev.User.Email
	}
	if id == "" {
		id = ev.User.Username
	}
	if id == "" {
		id = ev.User.IPAddress
	}
	if id == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:8])
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// do not cut inside a UTF-8 sequence.
	for n > 0 && s[n]&0xC0 == 0x80 {
		n--
	}
	return s[:n]
}

// normalizeEventID lower-cases the id and removes dashes (Sentry uses 32 hex chars).
func normalizeEventID(id string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(id), "-", ""))
}

func newEventID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func secureFloat() float64 {
	n, err := rand.Int(rand.Reader, big.NewInt(1<<53))
	if err != nil {
		return 0
	}
	return float64(n.Int64()) / float64(1<<53)
}
