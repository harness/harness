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

package notification

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/services/notification/mailer"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type mailerStub struct {
	calls []mailer.Payload
	err   error
}

func (s *mailerStub) Send(_ context.Context, payload mailer.Payload) error {
	s.calls = append(s.calls, payload)
	return s.err
}

func buildBasePayload() *BasePullReqPayload {
	return &BasePullReqPayload{
		Repo:       &types.Repository{ID: 1, Identifier: "my-repo", Path: "space/my-repo"},
		PullReq:    &types.PullReq{ID: 10, Number: 42, Title: "My PR"},
		Author:     &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"},
		PullReqURL: "https://example.com/pr/42",
	}
}

func TestSendUserGroupReviewerAdded_SendsEmailToRecipients(t *testing.T) {
	t.Parallel()

	stub := &mailerStub{}
	client := NewMailClient(stub)

	recipients := []*types.PrincipalInfo{
		{ID: 2, DisplayName: "Alice", Email: "alice@example.com"},
		{ID: 3, DisplayName: "Bob", Email: "bob@example.com"},
	}

	payload := &UserGroupReviewerAddedPayload{
		Base:          buildBasePayload(),
		ReviewerCount: 2,
		ReviewerNames: "Alice, Bob",
	}

	err := client.SendUserGroupReviewerAdded(context.Background(), recipients, payload)

	require.NoError(t, err)
	require.Len(t, stub.calls, 1)
	sent := stub.calls[0]
	require.ElementsMatch(t, []string{"alice@example.com", "bob@example.com"}, sent.ToRecipients)
	require.Contains(t, sent.Subject, "My PR")
	require.Contains(t, sent.Subject, "#42")
	require.Equal(t, "space/my-repo", sent.RepoRef)
}

func TestSendUserGroupReviewerAdded_SubjectContainsRepoIdentifier(t *testing.T) {
	t.Parallel()

	stub := &mailerStub{}
	client := NewMailClient(stub)

	recipients := []*types.PrincipalInfo{
		{ID: 2, DisplayName: "Alice", Email: "alice@example.com"},
	}

	payload := &UserGroupReviewerAddedPayload{
		Base:          buildBasePayload(),
		ReviewerCount: 1,
		ReviewerNames: "Alice",
	}

	err := client.SendUserGroupReviewerAdded(context.Background(), recipients, payload)

	require.NoError(t, err)
	require.Len(t, stub.calls, 1)
	require.Contains(t, stub.calls[0].Subject, "my-repo")
}

func TestSendUserGroupReviewerAdded_PropagatesMailerError(t *testing.T) {
	t.Parallel()

	sendErr := errors.New("smtp unreachable")
	stub := &mailerStub{err: sendErr}
	client := NewMailClient(stub)

	recipients := []*types.PrincipalInfo{
		{ID: 2, DisplayName: "Alice", Email: "alice@example.com"},
	}

	payload := &UserGroupReviewerAddedPayload{
		Base:          buildBasePayload(),
		ReviewerCount: 1,
		ReviewerNames: "Alice",
	}

	err := client.SendUserGroupReviewerAdded(context.Background(), recipients, payload)

	require.Error(t, err)
	require.ErrorIs(t, err, sendErr)
}

func TestSendUserGroupReviewerAdded_SingleRecipient(t *testing.T) {
	t.Parallel()

	stub := &mailerStub{}
	client := NewMailClient(stub)

	recipient := &types.PrincipalInfo{ID: 5, DisplayName: "Carol", Email: "carol@example.com"}

	payload := &UserGroupReviewerAddedPayload{
		Base:          buildBasePayload(),
		ReviewerCount: 1,
		ReviewerNames: "Carol",
	}

	err := client.SendUserGroupReviewerAdded(context.Background(), []*types.PrincipalInfo{recipient}, payload)

	require.NoError(t, err)
	require.Len(t, stub.calls, 1)
	require.Equal(t, []string{"carol@example.com"}, stub.calls[0].ToRecipients)
}
