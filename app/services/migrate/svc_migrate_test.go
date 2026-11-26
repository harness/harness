// Copyright 2026 Harness, Inc.
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

package migrate

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/app/store"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type svcTx struct{}

func (svcTx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	return fn(ctx)
}

var svcMigrator = types.Principal{ID: 7, UID: "migrator"}

// svcLabelStore is an in-memory store.LabelStore.
type svcLabelStore struct {
	store.LabelStore
	labels     map[string]*types.Label
	nextID     int64
	defineErr  error
	findErr    error
	incrErr    error
	increments []int
}

func (s *svcLabelStore) Find(_ context.Context, _, _ *int64, key string) (*types.Label, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	l, ok := s.labels[key]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return l, nil
}

func (s *svcLabelStore) Define(_ context.Context, l *types.Label) error {
	if s.defineErr != nil {
		return s.defineErr
	}
	s.nextID++
	l.ID = s.nextID
	s.labels[l.Key] = l
	return nil
}

func (s *svcLabelStore) IncrementValueCount(_ context.Context, _ int64, n int) (int64, error) {
	s.increments = append(s.increments, n)
	return 0, s.incrErr
}

type svcValueStore struct {
	store.LabelValueStore
	values    map[int64][]*types.LabelValue
	defineErr error
	listErr   error
}

func (s *svcValueStore) List(_ context.Context, labelID int64, _ types.ListQueryFilter) ([]*types.LabelValue, error) {
	return s.values[labelID], s.listErr
}

func (s *svcValueStore) Define(_ context.Context, v *types.LabelValue) error {
	if s.defineErr != nil {
		return s.defineErr
	}
	s.values[v.LabelID] = append(s.values[v.LabelID], v)
	return nil
}

type svcLabelSpaceStore struct {
	store.SpaceStore
	ancestors []int64
	err       error
}

func (s svcLabelSpaceStore) GetAncestorIDs(context.Context, int64) ([]int64, error) {
	return s.ancestors, s.err
}

func svcNewLabelMigrate() (*Label, *svcLabelStore, *svcValueStore, *svcLabelSpaceStore) {
	labels := &svcLabelStore{labels: map[string]*types.Label{}}
	values := &svcValueStore{values: map[int64][]*types.LabelValue{}}
	spaces := &svcLabelSpaceStore{ancestors: []int64{1, 2}}
	return NewLabel(labels, values, spaces, svcTx{}), labels, values, spaces
}

func TestSvcLabelImport(t *testing.T) {
	ctx := context.Background()
	space := &types.SpaceCore{ID: 5}

	t.Run("defines labels and values", func(t *testing.T) {
		m, labels, values, _ := svcNewLabelMigrate()
		out, err := m.Import(ctx, svcMigrator, space, []*ExternalLabel{
			{Name: "bug", Color: "ff0000", Description: "a bug", Value: "high"},
			{Name: "bug", Color: "ff0000", Value: "low"},
			{Name: "docs", Color: "#0000ff"},
		})
		require.NoError(t, err)
		require.Len(t, out, 3)

		bug := labels.labels["bug"]
		require.NotNil(t, bug)
		assert.Equal(t, enum.LabelColorRed, bug.Color)
		assert.Equal(t, int64(2), bug.Scope, "scope is the depth of the space hierarchy")
		assert.Equal(t, int64(7), bug.CreatedBy)
		assert.Equal(t, enum.LabelTypeStatic, bug.Type)
		require.NotNil(t, bug.SpaceID)
		assert.Equal(t, int64(5), *bug.SpaceID)

		require.Len(t, values.values[bug.ID], 2, "both values are defined exactly once")
		assert.Equal(t, defaultLabelValueColor, values.values[bug.ID][0].Color)
		assert.Contains(t, labels.labels, "docs")
	})
	t.Run("existing labels and values are reused", func(t *testing.T) {
		m, labels, values, _ := svcNewLabelMigrate()
		existing := &types.Label{ID: 99, Key: "bug"}
		labels.labels["bug"] = existing
		values.values[99] = []*types.LabelValue{{LabelID: 99, Value: "high"}}

		_, err := m.Import(ctx, svcMigrator, space, []*ExternalLabel{
			{Name: "bug", Value: "high"}, {Name: "bug", Value: "low"},
		})
		require.NoError(t, err)
		assert.Len(t, values.values[99], 2)
		assert.Equal(t, int64(0), labels.nextID, "no new label is defined")
		assert.Contains(t, labels.increments, 1)
	})
	t.Run("hierarchy failure", func(t *testing.T) {
		m, _, _, spaces := svcNewLabelMigrate()
		spaces.err = errors.New("db down")
		_, err := m.Import(ctx, svcMigrator, space, []*ExternalLabel{{Name: "bug"}})
		require.Error(t, err)
	})
	t.Run("invalid label", func(t *testing.T) {
		m, _, _, _ := svcNewLabelMigrate()
		_, err := m.Import(ctx, svcMigrator, space, []*ExternalLabel{{Name: ""}})
		require.Error(t, err)
	})
	t.Run("invalid value", func(t *testing.T) {
		m, _, _, _ := svcNewLabelMigrate()
		_, err := m.Import(ctx, svcMigrator, space, []*ExternalLabel{
			{Name: "bug", Value: ""}, {Name: "ok", Value: "bad\x01value"},
		})
		require.Error(t, err)
	})
	t.Run("store failures", func(t *testing.T) {
		m, labels, _, _ := svcNewLabelMigrate()
		labels.defineErr = errors.New("db down")
		_, err := m.Import(ctx, svcMigrator, space, []*ExternalLabel{{Name: "bug"}})
		require.Error(t, err)

		m, labels, _, _ = svcNewLabelMigrate()
		labels.findErr = errors.New("db down")
		_, err = m.Import(ctx, svcMigrator, space, []*ExternalLabel{{Name: "bug"}})
		require.Error(t, err)

		m, _, values, _ := svcNewLabelMigrate()
		values.listErr = errors.New("db down")
		_, err = m.Import(ctx, svcMigrator, space, []*ExternalLabel{{Name: "bug"}})
		require.Error(t, err)

		m, _, values, _ = svcNewLabelMigrate()
		values.defineErr = errors.New("db down")
		_, err = m.Import(ctx, svcMigrator, space, []*ExternalLabel{{Name: "bug", Value: "v"}})
		require.Error(t, err)

		m, labels, _, _ = svcNewLabelMigrate()
		labels.incrErr = errors.New("db down")
		_, err = m.Import(ctx, svcMigrator, space, []*ExternalLabel{{Name: "bug"}})
		require.Error(t, err)
	})
}

func TestSvcFindClosestColor(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name  string
		input string
		want  enum.LabelColor
	}{
		{"pure red", "#C7292F", enum.LabelColorRed},
		{"without hash", "16794C", enum.LabelColorGreen},
		{"lower case", "#236e93", enum.LabelColorBlue},
		{"pink", "#C41B87", enum.LabelColorPink},
		{"purple", "#9C2AAD", enum.LabelColorPurple},
		{"violet", "#5645AF", enum.LabelColorViolet},
		{"indigo", "#3250B2", enum.LabelColorIndigo},
		{"cyan", "#0B7792", enum.LabelColorCyan},
		{"orange", "#995137", enum.LabelColorOrange},
		{"brown", "#805C43", enum.LabelColorBrown},
		{"mint", "#247469", enum.LabelColorMint},
		{"lime", "#586729", enum.LabelColorLime},
		{"yellow", "#92582D", enum.LabelColorYellow},
		{"bright red is closest to red", "#FF0000", enum.LabelColorRed},
		{"not a color falls back to default", "not-a-color", enum.LabelColorBlue},
		{"empty falls back to default", "", enum.LabelColorBlue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, findClosestColor(ctx, tt.input))
		})
	}
}

type svcRuleStore struct {
	store.RuleStore
	created   []*types.Rule
	createErr error
}

func (s *svcRuleStore) Create(_ context.Context, r *types.Rule) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.created = append(s.created, r)
	return nil
}

type svcPrincipalStore struct {
	store.PrincipalStore
	byEmail map[string]*types.Principal
	err     error
}

func (s svcPrincipalStore) FindByEmail(_ context.Context, email string) (*types.Principal, error) {
	if s.err != nil {
		return nil, s.err
	}
	p, ok := s.byEmail[email]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return p, nil
}

func svcNewRuleMigrate() (*Rule, *svcRuleStore, *svcPrincipalStore) {
	rules := &svcRuleStore{}
	principals := &svcPrincipalStore{byEmail: map[string]*types.Principal{"a@example.com": {ID: 11}}}
	return NewRule(rules, svcTx{}, principals), rules, principals
}

const svcRuleDefinition = `{
	"bypass": {"user_emails": ["a@example.com", "unknown@example.com"], "repo_owners": true},
	"pullreq": {
		"approvals": {"require_minimum_count": 2, "require_latest_commit": true},
		"comments": {"require_resolve_all": true},
		"merge": {"strategies_allowed": ["squash", "merge"], "delete_branch": true}
	},
	"lifecycle": {"delete_forbidden": true, "update_forbidden": true}
}`

func TestSvcRuleImport(t *testing.T) {
	ctx := context.Background()
	repo := &types.RepositoryCore{ID: 3}
	extRule := func(identifier, definition, pattern string) *ExternalRule {
		return &ExternalRule{
			Identifier: identifier, State: "active", Definition: []byte(definition), Pattern: []byte(pattern),
		}
	}

	t.Run("imports a branch rule", func(t *testing.T) {
		m, store, _ := svcNewRuleMigrate()
		out, err := m.Import(ctx, svcMigrator, repo, ExternalRuleTypeBranch, []*ExternalRule{
			extRule("rule-1", svcRuleDefinition, `{"default":true,"include":["release-*"]}`),
		})
		require.NoError(t, err)
		require.Len(t, out, 1)
		require.Len(t, store.created, 1)

		r := store.created[0]
		assert.Equal(t, "rule-1", r.Identifier)
		assert.Equal(t, enum.RuleStateActive, r.State)
		assert.Equal(t, int64(7), r.CreatedBy)
		require.NotNil(t, r.RepoID)
		assert.Equal(t, int64(3), *r.RepoID)
		assert.Nil(t, r.SpaceID)
		assert.JSONEq(t, `{"default":true,"include":["release-*"]}`, string(r.Pattern))
		assert.Contains(t, string(r.Definition), `"user_ids":[11]`, "known emails are mapped, unknown ones are skipped")
		assert.Contains(t, string(r.Definition), `"repo_owners":true`)
		assert.Contains(t, string(r.Definition), `"delete_forbidden":true`)
	})
	t.Run("invalid input", func(t *testing.T) {
		tests := []struct {
			name string
			rule *ExternalRule
		}{
			{"bad identifier", extRule("not valid!", `{}`, `{}`)},
			{"bad definition json", extRule("r", `{`, `{}`)},
			{"bad definition values", extRule("r", `{"pullreq":{"approvals":{"require_minimum_count":-1}}}`, `{}`)},
			{"bad pattern json", extRule("r", `{}`, `{`)},
			{"bad pattern glob", extRule("r", `{}`, `{"include":["[a-"]}`)},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				m, store, _ := svcNewRuleMigrate()
				_, err := m.Import(ctx, svcMigrator, repo, ExternalRuleTypeBranch, []*ExternalRule{tt.rule})
				require.Error(t, err)
				assert.Empty(t, store.created)
			})
		}
	})
	t.Run("principal lookup failure", func(t *testing.T) {
		m, _, principals := svcNewRuleMigrate()
		principals.err = errors.New("db down")
		_, err := m.Import(ctx, svcMigrator, repo, ExternalRuleTypeBranch, []*ExternalRule{
			extRule("r", `{"bypass":{"user_emails":["a@example.com"]}}`, `{}`),
		})
		require.Error(t, err)
	})
	t.Run("store failure", func(t *testing.T) {
		m, store, _ := svcNewRuleMigrate()
		store.createErr = errors.New("db down")
		_, err := m.Import(ctx, svcMigrator, repo, ExternalRuleTypeBranch, []*ExternalRule{extRule("r", `{}`, `{}`)})
		require.Error(t, err)
	})
}

func TestSvcConvertMergeMethods(t *testing.T) {
	assert.Empty(t, convertMergeMethods(nil))
	assert.Equal(t,
		[]enum.MergeMethod{enum.MergeMethodSquash, enum.MergeMethodRebase},
		convertMergeMethods([]string{"squash", "rebase"}))
}

type svcWebhookStore struct {
	store.WebhookStore
	created   []*types.Webhook
	createErr error
}

func (s *svcWebhookStore) Create(_ context.Context, h *types.Webhook) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.created = append(s.created, h)
	return nil
}

func TestSvcWebhookImport(t *testing.T) {
	ctx := context.Background()
	repo := &types.RepositoryCore{ID: 3}
	cfg := webhook.Config{}

	t.Run("imports webhooks", func(t *testing.T) {
		ws := &svcWebhookStore{}
		m := NewWebhook(cfg, svcTx{}, ws)
		out, err := m.Import(ctx, svcMigrator, repo, []*ExternalWebhook{{
			Identifier: "hook-1", Target: "https://example.com/hook", Active: true, SkipVerify: true,
			Events: []string{"branch_created", "branch_created", "tag_created"},
		}})
		require.NoError(t, err)
		require.Len(t, out, 1)
		require.Len(t, ws.created, 1)

		h := ws.created[0]
		assert.Equal(t, "hook-1", h.Identifier)
		assert.Equal(t, "hook-1", h.DisplayName)
		assert.Equal(t, "https://example.com/hook", h.URL)
		assert.True(t, h.Enabled)
		assert.True(t, h.Insecure)
		assert.Equal(t, int64(3), h.ParentID)
		assert.Equal(t, enum.WebhookParentRepo, h.ParentType)
		assert.Equal(t, int64(7), h.CreatedBy)
		assert.Equal(t,
			[]enum.WebhookTrigger{enum.WebhookTriggerBranchCreated, enum.WebhookTriggerTagCreated}, h.Triggers)
	})
	t.Run("invalid webhooks are rejected", func(t *testing.T) {
		tests := []struct {
			name string
			hook *ExternalWebhook
		}{
			{"bad identifier", &ExternalWebhook{Identifier: "bad id!", Target: "https://example.com"}},
			{"loopback target", &ExternalWebhook{Identifier: "h", Target: "http://127.0.0.1/hook"}},
			{"private target", &ExternalWebhook{Identifier: "h", Target: "http://10.0.0.1/hook"}},
			{"bad scheme", &ExternalWebhook{Identifier: "h", Target: "ftp://example.com"}},
			{"bad trigger", &ExternalWebhook{Identifier: "h", Target: "https://example.com", Events: []string{"nope"}}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ws := &svcWebhookStore{}
				_, err := NewWebhook(cfg, svcTx{}, ws).Import(ctx, svcMigrator, repo, []*ExternalWebhook{tt.hook})
				require.Error(t, err)
				assert.Empty(t, ws.created)
			})
		}
	})
	t.Run("loopback and private targets can be allowed", func(t *testing.T) {
		ws := &svcWebhookStore{}
		m := NewWebhook(webhook.Config{AllowLoopback: true, AllowPrivateNetwork: true}, svcTx{}, ws)
		_, err := m.Import(ctx, svcMigrator, repo, []*ExternalWebhook{
			{Identifier: "a", Target: "http://127.0.0.1/hook"}, {Identifier: "b", Target: "http://10.0.0.1/hook"},
		})
		require.NoError(t, err)
		assert.Len(t, ws.created, 2)
	})
	t.Run("store failure", func(t *testing.T) {
		ws := &svcWebhookStore{createErr: errors.New("db down")}
		_, err := NewWebhook(cfg, svcTx{}, ws).Import(ctx, svcMigrator, repo, []*ExternalWebhook{
			{Identifier: "a", Target: "https://example.com"},
		})
		require.Error(t, err)
	})
}
