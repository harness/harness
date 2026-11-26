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

package space

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/harness/gitness/app/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type svcEncrypter struct {
	encErr error
	decErr error
}

func (e svcEncrypter) Encrypt(plaintext string) ([]byte, error) {
	if e.encErr != nil {
		return nil, e.encErr
	}
	return []byte("enc:" + plaintext), nil
}

func (e svcEncrypter) Decrypt(ciphertext []byte) (string, error) {
	if e.decErr != nil {
		return "", e.decErr
	}
	return string(ciphertext[len("enc:"):]), nil
}

func TestSvcJobDefinitionRoundTrip(t *testing.T) {
	input := Input{SourceSpacePath: "root/src", DestinationSpacePath: "root/dst"}

	s := &Service{encrypter: svcEncrypter{}}
	def, err := s.getJobDef(s.JobUIDFromSpacePath(input.SourceSpacePath), input)
	require.NoError(t, err)
	assert.Equal(t, "space-move-root/src", def.UID)
	assert.Equal(t, jobType, def.Type)
	assert.Equal(t, moveJobMaxRetries, def.MaxRetries)
	assert.Equal(t, moveJobMaxDuration, def.Timeout)
	assert.NotEmpty(t, def.Data)

	got, err := s.getJobInput(def.Data)
	require.NoError(t, err)
	assert.Equal(t, input, got)
}

func TestSvcJobDefinitionFailures(t *testing.T) {
	input := Input{SourceSpacePath: "a", DestinationSpacePath: "b"}

	_, err := (&Service{encrypter: svcEncrypter{encErr: errors.New("boom")}}).getJobDef("uid", input)
	require.Error(t, err)

	s := &Service{encrypter: svcEncrypter{}}
	_, err = s.getJobInput("%%% not base64 %%%")
	require.Error(t, err)

	def, err := s.getJobDef("uid", input)
	require.NoError(t, err)
	_, err = (&Service{encrypter: svcEncrypter{decErr: errors.New("boom")}}).getJobInput(def.Data)
	require.Error(t, err)

	// valid base64 and decryption, but the payload is not JSON.
	notJSON, err := s.encrypter.Encrypt("not json")
	require.NoError(t, err)
	_, err = s.getJobInput(base64.StdEncoding.EncodeToString(notJSON))
	require.Error(t, err)
}

type svcRepoStore struct {
	store.RepoStore
	count          int64
	err            error
	gotSrc, gotDst int64
}

func (s *svcRepoStore) UpdateParent(_ context.Context, src, dst int64) (int64, error) {
	s.gotSrc, s.gotDst = src, dst
	return s.count, s.err
}

type svcLabelStore struct {
	store.LabelStore
	count int64
	err   error
}

func (s svcLabelStore) UpdateParentSpace(context.Context, int64, int64) (int64, error) {
	return s.count, s.err
}

type svcRuleStore struct {
	store.RuleStore
	count int64
	err   error
}

func (s svcRuleStore) UpdateParentSpace(context.Context, int64, int64) (int64, error) {
	return s.count, s.err
}

type svcWebhookStore struct {
	store.WebhookStore
	count int64
	err   error
}

func (s svcWebhookStore) UpdateParentSpace(context.Context, int64, int64) (int64, error) {
	return s.count, s.err
}

func TestSvcCodeResourceMover(t *testing.T) {
	ctx := context.Background()
	boom := errors.New("db down")

	t.Run("moves all resource kinds", func(t *testing.T) {
		repos := &svcRepoStore{count: 1}
		mover := ProvideCodeResourceMover(repos, svcLabelStore{count: 2}, svcRuleStore{count: 3}, svcWebhookStore{count: 4})

		out, err := mover.MoveResources(ctx, 10, 20)
		require.NoError(t, err)
		assert.Equal(t, int64(1), out.RepoCount)
		assert.Equal(t, int64(2), out.LabelCount)
		assert.Equal(t, int64(3), out.RuleCount)
		assert.Equal(t, int64(4), out.WebhookCount)
		assert.Equal(t, int64(10), repos.gotSrc)
		assert.Equal(t, int64(20), repos.gotDst)
	})
	t.Run("stops at the first failure", func(t *testing.T) {
		tests := []struct {
			name   string
			repos  *svcRepoStore
			labels svcLabelStore
			rules  svcRuleStore
			hooks  svcWebhookStore
		}{
			{"repos", &svcRepoStore{err: boom}, svcLabelStore{}, svcRuleStore{}, svcWebhookStore{}},
			{"labels", &svcRepoStore{}, svcLabelStore{err: boom}, svcRuleStore{}, svcWebhookStore{}},
			{"rules", &svcRepoStore{}, svcLabelStore{}, svcRuleStore{err: boom}, svcWebhookStore{}},
			{"webhooks", &svcRepoStore{}, svcLabelStore{}, svcRuleStore{}, svcWebhookStore{err: boom}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := NewCodeResourceMover(tt.repos, tt.labels, tt.rules, tt.hooks).MoveResources(ctx, 1, 2)
				require.ErrorIs(t, err, boom)
			})
		}
	})
}

func TestSvcNoopResourceMover(t *testing.T) {
	out, err := ProvideNoopResourceMover().MoveResources(context.Background(), 1, 2)
	require.Error(t, err)
	assert.Equal(t, MoveResourcesOutput{}, out)
}
