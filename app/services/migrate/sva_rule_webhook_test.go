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

package migrate

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type svaRuleStore struct {
	store.RuleStore

	created []*types.Rule
	err     error
}

func (s *svaRuleStore) Create(_ context.Context, rule *types.Rule) error {
	if s.err != nil {
		return s.err
	}
	s.created = append(s.created, rule)
	return nil
}

type svaWebhookStore struct {
	store.WebhookStore

	created []*types.Webhook
	err     error
}

func (s *svaWebhookStore) Create(_ context.Context, hook *types.Webhook) error {
	if s.err != nil {
		return s.err
	}
	s.created = append(s.created, hook)
	return nil
}

func svaBranchRule(identifier, definition, pattern string) *ExternalRule {
	return &ExternalRule{
		Identifier: identifier,
		State:      string(enum.RuleStateActive),
		Definition: json.RawMessage(definition),
		Pattern:    json.RawMessage(pattern),
	}
}

func TestSvaRuleImport(t *testing.T) {
	t.Parallel()

	migrator := types.Principal{ID: 7}
	repo := &types.RepositoryCore{ID: 11}
	principals := &svaPrincipalStore{byEmail: map[string]*types.Principal{"owner@example.com": {ID: 21}}}

	const definition = `{
		"bypass": {"user_emails": ["owner@example.com", "missing@example.com"], "repo_owners": true},
		"pullreq": {
			"approvals": {"require_minimum_count": 2, "require_latest_commit": true, "require_code_owners": true},
			"comments": {"require_resolve_all": true},
			"merge": {"strategies_allowed": ["squash", "merge"], "delete_branch": true}
		},
		"lifecycle": {"delete_forbidden": true, "update_force_forbidden": true}
	}`
	const pattern = `{"default": true, "include": ["release/*"], "exclude": ["tmp/*"]}`

	t.Run("converts and stores rules", func(t *testing.T) {
		t.Parallel()

		rules := &svaRuleStore{}
		got, err := NewRule(rules, svaTx{}, principals).Import(
			context.Background(), migrator, repo, ExternalRuleTypeBranch,
			[]*ExternalRule{svaBranchRule("main-rule", definition, pattern)},
		)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Len(t, rules.created, 1)

		rule := rules.created[0]
		require.Equal(t, "main-rule", rule.Identifier)
		require.Equal(t, int64(7), rule.CreatedBy)
		require.Equal(t, int64(11), *rule.RepoID)
		require.Nil(t, rule.SpaceID)
		require.Equal(t, protection.TypeBranch, rule.Type)
		require.Equal(t, enum.RuleStateActive, rule.State)

		var def protection.Branch
		require.NoError(t, json.Unmarshal(rule.Definition, &def))
		require.Equal(t, []int64{21}, def.Bypass.UserIDs, "unknown bypass users must be dropped")
		require.True(t, def.Bypass.RepoOwners)
		require.Equal(t, 2, def.PullReq.Approvals.RequireMinimumCount)
		require.True(t, def.PullReq.Approvals.RequireCodeOwners)
		require.True(t, def.PullReq.Comments.RequireResolveAll)
		require.True(t, def.PullReq.Merge.DeleteBranch)
		require.ElementsMatch(t, []enum.MergeMethod{enum.MergeMethodSquash, enum.MergeMethodMerge},
			def.PullReq.Merge.StrategiesAllowed)
		require.True(t, def.Lifecycle.DeleteForbidden)
		require.True(t, def.Lifecycle.UpdateForceForbidden)

		var pat protection.Pattern
		require.NoError(t, json.Unmarshal(rule.Pattern, &pat))
		require.True(t, pat.Default)
		require.Equal(t, []string{"release/*"}, pat.Include)
		require.Equal(t, []string{"tmp/*"}, pat.Exclude)
	})

	t.Run("no rules", func(t *testing.T) {
		t.Parallel()

		got, err := NewRule(&svaRuleStore{}, svaTx{}, principals).Import(
			context.Background(), migrator, repo, ExternalRuleTypeBranch, nil)
		require.NoError(t, err)
		require.Empty(t, got)
	})

	errTests := []struct {
		name  string
		rule  *ExternalRule
		store *svaRuleStore
		tx    svaTx
		pstor *svaPrincipalStore
	}{
		{name: "invalid identifier", rule: svaBranchRule("not valid!", "{}", "{}")},
		{name: "malformed definition", rule: svaBranchRule("rule", "{", "{}")},
		{
			name: "invalid definition",
			rule: svaBranchRule("rule", `{"pullreq":{"approvals":{"require_minimum_count":-1}}}`, "{}"),
		},
		{
			name: "invalid merge strategy",
			rule: svaBranchRule("rule", `{"pullreq":{"merge":{"strategies_allowed":["bogus"]}}}`, "{}"),
		},
		{
			name:  "principal lookup",
			rule:  svaBranchRule("rule", `{"bypass":{"user_emails":["a@example.com"]}}`, "{}"),
			pstor: &svaPrincipalStore{err: errSvaBoom},
		},
		{name: "malformed pattern", rule: svaBranchRule("rule", "{}", "[")},
		{name: "invalid pattern", rule: svaBranchRule("rule", "{}", `{"include":[""]}`)},
		{name: "store failure", rule: svaBranchRule("rule", "{}", "{}"), store: &svaRuleStore{err: errSvaBoom}},
		{name: "transaction failure", rule: svaBranchRule("rule", "{}", "{}"), tx: svaTx{err: errSvaBoom}},
	}

	for _, test := range errTests {
		t.Run("error "+test.name, func(t *testing.T) {
			t.Parallel()

			rules := test.store
			if rules == nil {
				rules = &svaRuleStore{}
			}
			pstor := test.pstor
			if pstor == nil {
				pstor = principals
			}

			_, err := NewRule(rules, test.tx, pstor).Import(
				context.Background(), migrator, repo, ExternalRuleTypeBranch, []*ExternalRule{test.rule})
			require.Error(t, err)
			require.Empty(t, rules.created)
		})
	}
}

func TestSvaConvertMergeMethods(t *testing.T) {
	t.Parallel()

	require.Empty(t, convertMergeMethods(nil))
	require.Equal(t,
		[]enum.MergeMethod{enum.MergeMethodRebase, enum.MergeMethodFastForward},
		convertMergeMethods([]string{"rebase", "fast-forward"}))
}

func TestSvaWebhookImport(t *testing.T) {
	t.Parallel()

	migrator := types.Principal{ID: 7}
	repo := &types.RepositoryCore{ID: 11}

	newHook := func(identifier, target string, events ...string) *ExternalWebhook {
		return &ExternalWebhook{
			Identifier: identifier, Target: target, Events: events, Active: true, SkipVerify: true,
		}
	}

	t.Run("converts and stores webhooks", func(t *testing.T) {
		t.Parallel()

		hooks := &svaWebhookStore{}
		got, err := NewWebhook(webhook.Config{}, svaTx{}, hooks).Import(
			context.Background(), migrator, repo, []*ExternalWebhook{
				newHook("hook-one", "https://example.com/hook", "branch_created", "branch_created", "tag_created"),
				newHook("hook-two", "http://example.org/other"),
			})
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Len(t, hooks.created, 2)

		hook := hooks.created[0]
		require.Equal(t, "hook-one", hook.Identifier)
		require.Equal(t, "hook-one", hook.DisplayName)
		require.Equal(t, "https://example.com/hook", hook.URL)
		require.Equal(t, int64(11), hook.ParentID)
		require.Equal(t, enum.WebhookParentRepo, hook.ParentType)
		require.Equal(t, int64(7), hook.CreatedBy)
		require.True(t, hook.Enabled)
		require.True(t, hook.Insecure)
		require.Equal(t,
			[]enum.WebhookTrigger{enum.WebhookTriggerBranchCreated, enum.WebhookTriggerTagCreated}, hook.Triggers)
		require.Empty(t, hooks.created[1].Triggers)
	})

	t.Run("loopback addresses follow the configuration", func(t *testing.T) {
		t.Parallel()

		in := []*ExternalWebhook{newHook("hook-one", "http://127.0.0.1:8080/hook")}

		_, err := NewWebhook(webhook.Config{}, svaTx{}, &svaWebhookStore{}).Import(
			context.Background(), migrator, repo, in)
		require.Error(t, err)

		hooks := &svaWebhookStore{}
		_, err = NewWebhook(webhook.Config{AllowLoopback: true}, svaTx{}, hooks).Import(
			context.Background(), migrator, repo, in)
		require.NoError(t, err)
		require.Len(t, hooks.created, 1)
	})

	errTests := []struct {
		name  string
		hook  *ExternalWebhook
		store *svaWebhookStore
		tx    svaTx
	}{
		{name: "invalid identifier", hook: newHook("bad id!", "https://example.com")},
		{name: "invalid url", hook: newHook("hook-one", "ftp://example.com")},
		{name: "localhost", hook: newHook("hook-one", "http://localhost/hook")},
		{name: "invalid trigger", hook: newHook("hook-one", "https://example.com", "not_a_trigger")},
		{name: "store failure", hook: newHook("hook-one", "https://example.com"), store: &svaWebhookStore{err: errSvaBoom}},
		{name: "transaction failure", hook: newHook("hook-one", "https://example.com"), tx: svaTx{err: errSvaBoom}},
	}

	for _, test := range errTests {
		t.Run("error "+test.name, func(t *testing.T) {
			t.Parallel()

			hooks := test.store
			if hooks == nil {
				hooks = &svaWebhookStore{}
			}

			_, err := NewWebhook(webhook.Config{}, test.tx, hooks).Import(
				context.Background(), migrator, repo, []*ExternalWebhook{test.hook})
			require.Error(t, err)
			require.Empty(t, hooks.created)
		})
	}
}
