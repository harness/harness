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
	"context"
	"encoding/json"
	"testing"

	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/encrypt"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// The fakes below embed the interfaces and implement only the methods that Create/Update call;
// any other call panics, which flags unexpected use.

type headersFakeWebhookStore struct {
	store.WebhookStore
	hooks   map[int64]*types.Webhook
	created []*types.Webhook
	updated []*types.Webhook
}

func (f *headersFakeWebhookStore) Find(_ context.Context, id int64) (*types.Webhook, error) {
	if h, ok := f.hooks[id]; ok {
		c := h.Clone()
		return &c, nil
	}
	return nil, gitness_store.ErrResourceNotFound
}

func (f *headersFakeWebhookStore) FindByIdentifier(
	_ context.Context,
	parentType enum.WebhookParent,
	parentID int64,
	identifier string,
) (*types.Webhook, error) {
	for _, h := range f.hooks {
		if h.ParentType == parentType && h.ParentID == parentID && h.Identifier == identifier {
			c := h.Clone()
			return &c, nil
		}
	}
	return nil, gitness_store.ErrResourceNotFound
}

func (f *headersFakeWebhookStore) Create(_ context.Context, hook *types.Webhook) error {
	f.created = append(f.created, hook)
	return nil
}

func (f *headersFakeWebhookStore) Update(_ context.Context, hook *types.Webhook) error {
	f.updated = append(f.updated, hook)
	return nil
}

type headersFakeAuditService struct {
	events []audit.Event
}

func (f *headersFakeAuditService) Log(
	_ context.Context,
	user types.Principal,
	resource audit.Resource,
	action audit.Action,
	spacePath string,
	options ...audit.Option,
) error {
	e := audit.Event{User: user, Resource: resource, Action: action, SpacePath: spacePath}
	for _, o := range options {
		o.Apply(&e)
	}
	f.events = append(f.events, e)
	return nil
}

type headersFakeStreamer struct {
	sse.Streamer
	published []any
}

func (f *headersFakeStreamer) Publish(_ context.Context, _ int64, _ enum.SSEType, data any) {
	f.published = append(f.published, data)
}

const (
	headersTestRepoID   = int64(7)
	headersTestHookID   = int64(42)
	headersStoredToken  = "Bearer stored-super-secret" //nolint:gosec // test fixture, not a real credential
	headersStoredPlain  = "plain-value"
	headersTestHookName = "hook"
)

type headersFixture struct {
	svc       *Service
	store     *headersFakeWebhookStore
	audit     *headersFakeAuditService
	streamer  *headersFakeStreamer
	encrypter encrypt.Encrypter
	parent    ParentResource
	principal *types.Principal
}

func newHeadersFixture(t *testing.T) *headersFixture {
	t.Helper()

	encrypter, err := encrypt.New("fb4f3d6b1ec2fe0c5bf3d5bd8f1ac3a9", false)
	require.NoError(t, err)

	encryptedSecret, err := encrypter.Encrypt("stored-secret")
	require.NoError(t, err)

	whStore := &headersFakeWebhookStore{hooks: map[int64]*types.Webhook{
		headersTestHookID: {
			ID:          headersTestHookID,
			Identifier:  headersTestHookName,
			DisplayName: headersTestHookName,
			ParentID:    headersTestRepoID,
			ParentType:  enum.WebhookParentRepo,
			Type:        enum.WebhookTypeExternal,
			URL:         "https://example.com/hook",
			Secret:      string(encryptedSecret),
			ExtraHeaders: []types.ExtraHeader{
				{Key: "Authorization", Value: headersStoredToken, Masked: true},
				{Key: "X-Plain", Value: headersStoredPlain},
			},
		},
	}}
	auditSvc := &headersFakeAuditService{}
	streamer := &headersFakeStreamer{}

	svc := &Service{
		webhookStore: whStore,
		repoStore: &draftFakeRepoStore{repos: map[int64]*types.Repository{
			headersTestRepoID: {ID: headersTestRepoID, ParentID: 1},
		}},
		encrypter:    encrypter,
		auditService: auditSvc,
		sseStreamer:  streamer,
	}

	return &headersFixture{
		svc:       svc,
		store:     whStore,
		audit:     auditSvc,
		streamer:  streamer,
		encrypter: encrypter,
		parent: ParentResource{
			ID:         headersTestRepoID,
			Identifier: "repo",
			Type:       enum.WebhookParentRepo,
			Path:       "space/repo",
		},
		principal: &types.Principal{ID: 1, UID: "admin"},
	}
}

func (f *headersFixture) update(in *types.WebhookUpdateInput) (*types.Webhook, error) {
	return f.svc.Update(context.Background(), f.principal, headersTestHookName,
		enum.WebhookTypeExternal, f.parent, in)
}

// clientRoundTrip simulates a client that fetched the webhook (getting redacted headers) and sends them back.
func clientRoundTrip(t *testing.T, hook *types.Webhook) []types.ExtraHeader {
	t.Helper()

	raw, err := json.Marshal(hook)
	require.NoError(t, err)

	var out struct {
		ExtraHeaders []types.ExtraHeader `json:"extra_headers"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	return out.ExtraHeaders
}

func TestUpdate_RoundTripOfRedactedHeadersKeepsStoredValues(t *testing.T) {
	f := newHeadersFixture(t)

	stored, err := f.store.Find(context.Background(), headersTestHookID)
	require.NoError(t, err)

	fromClient := clientRoundTrip(t, stored)
	require.Equal(t, "", fromClient[0].Value, "precondition: masked value must be redacted for the client")

	desc := "new description"
	hook, err := f.update(&types.WebhookUpdateInput{Description: &desc, ExtraHeaders: fromClient})
	require.NoError(t, err)

	require.Len(t, f.store.updated, 1)
	assert.Equal(t, []types.ExtraHeader{
		{Key: "Authorization", Value: headersStoredToken, Masked: true},
		{Key: "X-Plain", Value: headersStoredPlain},
	}, f.store.updated[0].ExtraHeaders)
	assert.Equal(t, desc, hook.Description)
}

func TestUpdate_NilExtraHeadersLeavesHeadersUntouched(t *testing.T) {
	f := newHeadersFixture(t)

	desc := "only description"
	_, err := f.update(&types.WebhookUpdateInput{Description: &desc})
	require.NoError(t, err)

	require.Len(t, f.store.updated, 1)
	assert.Equal(t, f.store.hooks[headersTestHookID].ExtraHeaders, f.store.updated[0].ExtraHeaders)
}

func TestUpdate_ExtraHeaders(t *testing.T) {
	tests := []struct {
		name    string
		in      []types.ExtraHeader
		want    []types.ExtraHeader
		wantErr bool
	}{
		{
			name: "empty_list_removes_all_headers",
			in:   []types.ExtraHeader{},
			want: []types.ExtraHeader{},
		},
		{
			name: "new_masked_value_replaces_stored_value",
			in:   []types.ExtraHeader{{Key: "Authorization", Value: "Bearer rotated", Masked: true}},
			want: []types.ExtraHeader{{Key: "Authorization", Value: "Bearer rotated", Masked: true}},
		},
		{
			name: "key_rename_case_only_keeps_stored_value",
			in:   []types.ExtraHeader{{Key: "AUTHORIZATION", Masked: true}},
			want: []types.ExtraHeader{{Key: "AUTHORIZATION", Value: headersStoredToken, Masked: true}},
		},
		{
			name: "masking_plain_header_keeps_value",
			in:   []types.ExtraHeader{{Key: "X-Plain", Masked: true}},
			want: []types.ExtraHeader{{Key: "X-Plain", Value: headersStoredPlain, Masked: true}},
		},
		{
			name: "unmasking_without_value_clears_value",
			in:   []types.ExtraHeader{{Key: "Authorization"}},
			want: []types.ExtraHeader{{Key: "Authorization"}},
		},
		{
			name:    "new_masked_header_without_value_rejected",
			in:      []types.ExtraHeader{{Key: "X-New", Masked: true}},
			wantErr: true,
		},
		{
			name: "duplicate_keys_rejected",
			in: []types.ExtraHeader{
				{Key: "Authorization", Masked: true},
				{Key: "authorization", Value: "other"},
			},
			wantErr: true,
		},
		{
			name:    "empty_key_rejected",
			in:      []types.ExtraHeader{{Key: "", Value: "v"}},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newHeadersFixture(t)

			_, err := f.update(&types.WebhookUpdateInput{ExtraHeaders: tc.in})
			if tc.wantErr {
				require.Error(t, err)
				assert.Empty(t, f.store.updated, "invalid input must not be persisted")
				assert.Empty(t, f.audit.events)
				assert.Empty(t, f.streamer.published)
				return
			}

			require.NoError(t, err)
			require.Len(t, f.store.updated, 1)
			assert.Equal(t, tc.want, f.store.updated[0].ExtraHeaders)
		})
	}
}

func TestUpdate_ResponseAndSSERedactMaskedValues(t *testing.T) {
	f := newHeadersFixture(t)

	hook, err := f.update(&types.WebhookUpdateInput{ExtraHeaders: []types.ExtraHeader{
		{Key: "Authorization", Masked: true},
	}})
	require.NoError(t, err)

	// the returned hook carries the real value (needed internally), but its JSON must not.
	raw, err := json.Marshal(hook)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), headersStoredToken)

	require.Len(t, f.streamer.published, 1)
	raw, err = json.Marshal(f.streamer.published[0])
	require.NoError(t, err)
	assert.NotContains(t, string(raw), headersStoredToken, "SSE payload leaked masked value")
}

// TestUpdate_AuditDoesNotLeakMaskedValues checks the audit old/new objects as serialized by the enterprise
// audit.Service, which uses yaml.Marshal (MarshalJSON is not involved; types.Webhook excludes the extra headers
// and the secret via `yaml:"-"`).
func TestUpdate_AuditDoesNotLeakMaskedValues(t *testing.T) {
	f := newHeadersFixture(t)

	_, err := f.update(&types.WebhookUpdateInput{ExtraHeaders: []types.ExtraHeader{
		{Key: "Authorization", Masked: true},
	}})
	require.NoError(t, err)
	require.Len(t, f.audit.events, 1)

	diff := f.audit.events[0].DiffObject
	for name, obj := range map[string]any{"old": diff.OldObject, "new": diff.NewObject} {
		raw, err := yaml.Marshal(obj)
		require.NoError(t, err)
		assert.NotContains(t, string(raw), headersStoredToken, "%s audit object leaked masked value", name)
		assert.NotContains(t, string(raw), f.store.hooks[headersTestHookID].Secret, "%s audit object leaked secret", name)
	}
}

func TestUpdate_Secret(t *testing.T) {
	t.Run("empty_secret_clears_secret", func(t *testing.T) {
		f := newHeadersFixture(t)

		empty := ""
		hook, err := f.update(&types.WebhookUpdateInput{Secret: &empty})
		require.NoError(t, err)

		assert.Empty(t, f.store.updated[0].Secret)
		raw, err := json.Marshal(hook)
		require.NoError(t, err)
		assert.Contains(t, string(raw), `"has_secret":false`)
	})

	t.Run("new_secret_is_encrypted", func(t *testing.T) {
		f := newHeadersFixture(t)

		secret := "rotated-secret"
		_, err := f.update(&types.WebhookUpdateInput{Secret: &secret})
		require.NoError(t, err)

		stored := f.store.updated[0].Secret
		assert.NotEqual(t, secret, stored)
		decrypted, err := f.encrypter.Decrypt([]byte(stored))
		require.NoError(t, err)
		assert.Equal(t, secret, decrypted)
	})

	t.Run("nil_secret_keeps_stored_secret", func(t *testing.T) {
		f := newHeadersFixture(t)

		desc := "x"
		_, err := f.update(&types.WebhookUpdateInput{Description: &desc})
		require.NoError(t, err)
		assert.Equal(t, f.store.hooks[headersTestHookID].Secret, f.store.updated[0].Secret)
	})
}

func TestCreate_ExtraHeadersAndSecret(t *testing.T) {
	newInput := func(secret string, headers []types.ExtraHeader) *types.WebhookCreateInput {
		return &types.WebhookCreateInput{
			Identifier:   "new-hook",
			DisplayName:  "new-hook",
			URL:          "https://93.184.216.34/hook", // public IP literal: no DNS lookup needed
			Secret:       secret,
			ExtraHeaders: headers,
		}
	}
	create := func(f *headersFixture, in *types.WebhookCreateInput) (*types.Webhook, error) {
		return f.svc.Create(context.Background(), f.principal, enum.WebhookTypeExternal, f.parent, in)
	}

	t.Run("without_secret_has_no_secret", func(t *testing.T) {
		f := newHeadersFixture(t)

		hook, err := create(f, newInput("", nil))
		require.NoError(t, err)

		require.Len(t, f.store.created, 1)
		assert.Empty(t, f.store.created[0].Secret)
		raw, err := json.Marshal(hook)
		require.NoError(t, err)
		assert.Contains(t, string(raw), `"has_secret":false`)
	})

	t.Run("with_secret_is_encrypted", func(t *testing.T) {
		f := newHeadersFixture(t)

		hook, err := create(f, newInput("create-secret", nil))
		require.NoError(t, err)

		decrypted, err := f.encrypter.Decrypt([]byte(f.store.created[0].Secret))
		require.NoError(t, err)
		assert.Equal(t, "create-secret", decrypted)
		raw, err := json.Marshal(hook)
		require.NoError(t, err)
		assert.Contains(t, string(raw), `"has_secret":true`)
	})

	t.Run("masked_header_stored_but_redacted_in_response_and_sse", func(t *testing.T) {
		f := newHeadersFixture(t)

		hook, err := create(f, newInput("", []types.ExtraHeader{
			{Key: "Authorization", Value: "Bearer created-token", Masked: true},
		}))
		require.NoError(t, err)

		assert.Equal(t, "Bearer created-token", f.store.created[0].ExtraHeaders[0].Value)

		raw, err := json.Marshal(hook)
		require.NoError(t, err)
		assert.NotContains(t, string(raw), "created-token")

		require.Len(t, f.streamer.published, 1)
		raw, err = json.Marshal(f.streamer.published[0])
		require.NoError(t, err)
		assert.NotContains(t, string(raw), "created-token")
	})

	t.Run("masked_header_without_value_rejected", func(t *testing.T) {
		f := newHeadersFixture(t)

		_, err := create(f, newInput("", []types.ExtraHeader{{Key: "Authorization", Masked: true}}))
		require.Error(t, err)
		assert.Empty(t, f.store.created)
	})
}
