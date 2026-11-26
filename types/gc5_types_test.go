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

package types

import (
	"encoding/json"
	"testing"

	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestGc5LegacyUIDMarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"pipeline", Pipeline{Identifier: "gc5-id"}},
		{"template", Template{Identifier: "gc5-id"}},
		{"token", Token{Identifier: "gc5-id"}},
		{"trigger", Trigger{Identifier: "gc5-id"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(tt.value)
			require.NoError(t, err)

			var out map[string]any
			require.NoError(t, json.Unmarshal(raw, &out))
			require.Equal(t, "gc5-id", out["uid"])
			require.Equal(t, "gc5-id", out["identifier"])
		})
	}
}

func TestGc5StepString(t *testing.T) {
	s := Step{Name: "gc5-step", Image: "alpine"}

	out := s.String()
	require.Contains(t, out, `"name": "gc5-step"`)
	require.Contains(t, out, `"image": "alpine"`)
}

func TestGc5SpaceCore(t *testing.T) {
	s := &Space{ID: 3, ParentID: 2, Path: "a/b", Identifier: "b", Description: "ignored"}

	require.Equal(t, &SpaceCore{ID: 3, ParentID: 2, Path: "a/b", Identifier: "b"}, s.Core())
}

func TestGc5UserToPrincipal(t *testing.T) {
	u := &User{
		ID: 7, UID: "gc5-user", Email: "gc5@example.com", DisplayName: "Gc5 User", Admin: true, Blocked: true,
		Salt: "salt", Created: 11, Updated: 12,
	}

	p := u.ToPrincipal()
	require.Equal(t, enum.PrincipalTypeUser, p.Type)
	require.Equal(t, int64(7), p.ID)
	require.Equal(t, "gc5-user", p.UID)
	require.True(t, p.Admin)
	require.True(t, p.Blocked)
	require.Equal(t, "salt", p.Salt)

	info := u.ToPrincipalInfo()
	require.Equal(t, int64(7), info.ID)
	require.Equal(t, "Gc5 User", info.DisplayName)
	require.Equal(t, enum.PrincipalTypeUser, info.Type)
}

func TestGc5UserGroupInfo(t *testing.T) {
	g := &UserGroup{ID: 4, Identifier: "gc5-group", Name: "Gc5", Description: "d", Scope: 2, Users: []string{"x"}}

	require.Equal(t, &UserGroupInfo{ID: 4, Identifier: "gc5-group", Name: "Gc5", Description: "d", Scope: 2},
		g.ToUserGroupInfo())
}

func TestGc5WebhookClone(t *testing.T) {
	result := enum.WebhookExecutionResultSuccess
	orig := Webhook{
		Triggers:              []enum.WebhookTrigger{enum.WebhookTriggerBranchCreated},
		LatestExecutionResult: &result,
		ExtraHeaders:          []ExtraHeader{{Key: "k", Value: "v"}},
	}

	clone := orig.Clone()
	require.Equal(t, orig, clone)

	other := enum.WebhookExecutionResultFatalError
	*clone.LatestExecutionResult = other
	clone.Triggers[0] = enum.WebhookTriggerBranchDeleted
	clone.ExtraHeaders[0].Value = "changed"

	require.Equal(t, enum.WebhookExecutionResultSuccess, *orig.LatestExecutionResult)
	require.Equal(t, enum.WebhookTriggerBranchCreated, orig.Triggers[0])
	require.Equal(t, "v", orig.ExtraHeaders[0].Value)

	empty := Webhook{}.Clone()
	require.Nil(t, empty.LatestExecutionResult)
	require.Nil(t, empty.Triggers)
}

func TestGc5MergeQueueEntry(t *testing.T) {
	started, deadline := int64(10), int64(20)
	merge := sha.Must("1111111111111111111111111111111111111111")
	checks := sha.Must("2222222222222222222222222222222222222222")
	e := &MergeQueueEntry{
		PullReqID: 5, MergeQueueID: 6, OrderIndex: 7, MergeMethod: enum.MergeMethodSquash,
		State: enum.MergeQueueEntryStateChecksInProgress, MergeCommitSHA: merge, ChecksCommitSHA: checks,
		CommitCount: 3, ChangedFileCount: 4, Additions: 5, Deletions: 6,
		ChecksStarted: &started, ChecksDeadline: &deadline,
	}

	require.Contains(t, e.String(), "id=5 q=6 index=7 method=squash")
	require.Contains(t, e.String(), merge.String())

	e.CancelMerge()
	require.Equal(t, enum.MergeQueueEntryStateMergePending, e.State)
	require.Equal(t, sha.None, e.MergeCommitSHA)
	require.Equal(t, sha.None, e.ChecksCommitSHA)
	require.Zero(t, e.CommitCount+e.ChangedFileCount+e.Additions+e.Deletions)
	require.Nil(t, e.ChecksStarted)
	require.Nil(t, e.ChecksDeadline)
}

func TestGc5PlatformConnectorMappings(t *testing.T) {
	require.Equal(t, ArtifactoryPlatformConnectorType, ToPlatformConnectorType("Artifactory"))
	require.Equal(t, DockerRegistryPlatformConnectorType, ToPlatformConnectorType("DockerRegistry"))
	require.Equal(t, AWSPlatformConnectorType, ToPlatformConnectorType("Aws"))
	require.Equal(t, UnknownPlatformConnectorType, ToPlatformConnectorType("nope"))

	require.Equal(t, UserNamePasswordPlatformConnectorAuthType, ToPlatformConnectorAuthType("UsernamePassword"))
	require.Equal(t, AnonymousPlatformConnectorAuthType, ToPlatformConnectorAuthType("Anonymous"))
	require.Equal(t, UnknownPlatformConnectorAuthType, ToPlatformConnectorAuthType("nope"))

	require.Equal(t, ManualConfigAWSCredentialsType, ToAwsCredentialsType("ManualConfig"))
	require.Equal(t, UnknownAWSCredentialsType, ToAwsCredentialsType("nope"))
}

func TestGc5PlatformConnectorExtract(t *testing.T) {
	user, pass := NewMaskSecret("gc5-user"), NewMaskSecret("gc5-pass")
	basicAuth := PlatformConnectorAuthSpec{
		AuthType: UserNamePasswordPlatformConnectorAuthType, UserName: &user, Password: &pass,
		UserNameRef: "userRef", PasswordRef: "passRef",
	}

	docker := PlatformConnectorSpec{Type: DockerRegistryPlatformConnectorType, DockerRegistryURL: "docker.example.com",
		AuthSpec: basicAuth}
	artifactory := PlatformConnectorSpec{Type: ArtifactoryPlatformConnectorType, ArtifactoryURL: "art.example.com",
		AuthSpec: basicAuth}
	aws := PlatformConnectorSpec{Type: AWSPlatformConnectorType, AwsECRRegistryURL: "ecr.example.com",
		AwsCredentials: AwsCredentials{AccessToken: NewMaskSecret("gc5-aws-access")}}
	anonymous := PlatformConnectorSpec{Type: DockerRegistryPlatformConnectorType,
		AuthSpec: PlatformConnectorAuthSpec{AuthType: AnonymousPlatformConnectorAuthType}}

	require.Equal(t, "docker.example.com", docker.ExtractRegistryURL())
	require.Equal(t, "art.example.com", artifactory.ExtractRegistryURL())
	require.Equal(t, "ecr.example.com", aws.ExtractRegistryURL())
	require.Empty(t, PlatformConnectorSpec{Type: UnknownPlatformConnectorType}.ExtractRegistryURL())
	require.Empty(t, PlatformConnectorSpec{Type: "other"}.ExtractRegistryURL())

	require.Equal(t, "gc5-user", docker.ExtractRegistryUserName())
	require.Equal(t, "gc5-user", artifactory.ExtractRegistryUserName())
	require.Equal(t, "AWS", aws.ExtractRegistryUserName())
	require.Empty(t, anonymous.ExtractRegistryUserName())

	require.Equal(t, "gc5-pass", docker.ExtractRegistryAuth())
	require.Equal(t, "gc5-aws-access", aws.ExtractRegistryAuth())
	require.Empty(t, anonymous.ExtractRegistryAuth())
	require.Empty(t, PlatformConnectorSpec{Type: AWSPlatformConnectorType}.ExtractRegistryAuth())

	require.Equal(t, "userRef", basicAuth.ExtractUserNameRef())
	require.Equal(t, "passRef", basicAuth.ExtractPasswordRef())
	require.Empty(t, anonymous.AuthSpec.ExtractUserNameRef())
	require.Empty(t, anonymous.AuthSpec.ExtractPasswordRef())
}

func TestGc5PullReqActivityPredicates(t *testing.T) {
	resolved := int64(1)

	code := &PullReqActivity{
		Type: enum.PullReqActivityTypeCodeComment, Kind: enum.PullReqActivityKindChangeComment,
		CodeComment: &CodeCommentFields{Path: "a.go", LineNew: 3, SpanNew: 1, SourceSHA: "src", MergeBaseSHA: "mb"},
		ID:          9, Version: 2, Updated: 5,
	}
	require.True(t, code.IsValidCodeComment())
	cc := code.AsCodeComment()
	require.Equal(t, int64(9), cc.ID)
	require.Equal(t, "a.go", cc.Path)
	require.Equal(t, 3, cc.LineNew)

	notCode := &PullReqActivity{Type: enum.PullReqActivityTypeComment}
	require.False(t, notCode.IsValidCodeComment())
	require.Equal(t, &CodeComment{}, notCode.AsCodeComment())

	require.True(t, (&PullReqActivity{Type: enum.PullReqActivityTypeComment}).IsReplyable())
	require.False(t, (&PullReqActivity{Type: enum.PullReqActivityTypeComment, SubOrder: 1}).IsReplyable())
	require.False(t, (&PullReqActivity{Type: enum.PullReqActivityTypeMerge}).IsReplyable())
	require.False(t, (&PullReqActivity{
		Type: enum.PullReqActivityTypeCodeComment, Kind: enum.PullReqActivityKindReviewComment,
	}).IsReplyable())

	require.True(t, (&PullReqActivity{SubOrder: 2}).IsReply())
	require.False(t, (&PullReqActivity{}).IsReply())

	require.True(t, (&PullReqActivity{Kind: enum.PullReqActivityKindComment}).IsBlocking())
	require.False(t, (&PullReqActivity{Kind: enum.PullReqActivityKindComment, Resolved: &resolved}).IsBlocking())
	require.False(t, (&PullReqActivity{Kind: enum.PullReqActivityKindComment, Deleted: &resolved}).IsBlocking())
	require.False(t, (&PullReqActivity{Kind: enum.PullReqActivityKindComment, SubOrder: 1}).IsBlocking())
	require.False(t, (&PullReqActivity{Kind: enum.PullReqActivityKindSystem}).IsBlocking())
}

func TestGc5PullReqActivityPayloadRoundTrip(t *testing.T) {
	require.NotEmpty(t, allPullReqActivityPayloads)

	for activityType, factory := range allPullReqActivityPayloads {
		t.Run(string(activityType), func(t *testing.T) {
			a := &PullReqActivity{Type: activityType}

			_, err := a.GetPayload()
			require.ErrorIs(t, err, ErrNoPayload)

			require.NoError(t, a.SetPayload(factory()))
			got, err := a.GetPayload()
			if activityType == enum.PullReqActivityTypeComment {
				// the comment payload factory returns a non-pointer, which json cannot unmarshal into.
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, activityType, got.ActivityType())
			require.IsType(t, factory(), got)
		})
	}
}

func TestGc5PullReqActivityPayloadErrors(t *testing.T) {
	a := &PullReqActivity{Type: enum.PullReqActivityTypeMerge, PayloadRaw: json.RawMessage(`"x"`)}

	// wrong payload type.
	require.Error(t, a.SetPayload(&PullRequestActivityPayloadTitleChange{}))

	// nil payload resets the raw value.
	require.NoError(t, a.SetPayload(nil))
	require.Nil(t, a.PayloadRaw)

	// a JSON null is no payload either.
	a.PayloadRaw = json.RawMessage("null")
	_, err := a.GetPayload()
	require.ErrorIs(t, err, ErrNoPayload)

	// malformed payloads fail to unmarshal.
	a.PayloadRaw = json.RawMessage(`"not an object"`)
	_, err = a.GetPayload()
	require.Error(t, err)

	// activity types without payload.
	a = &PullReqActivity{Type: enum.PullReqActivityType("unknown"), PayloadRaw: json.RawMessage(`{}`)}
	_, err = a.GetPayload()
	require.Error(t, err)
	_, err = newPayloadForActivity(enum.PullReqActivityType("unknown"))
	require.Error(t, err)
}

func TestGc5PullReqActivityMetadata(t *testing.T) {
	var nilMeta *PullReqActivityMetadata
	require.True(t, nilMeta.IsEmpty())
	require.True(t, (&PullReqActivityMetadata{}).IsEmpty())
	require.True(t, (&PullReqActivityMetadata{Reactions: &PullReqActivityReactionsMetadata{}}).IsEmpty())
	require.False(t, (&PullReqActivityMetadata{Mentions: &PullReqActivityMentionsMetadata{}}).IsEmpty())

	a := &PullReqActivity{}

	a.UpdateMetadata(WithPullReqActivityMetadataUpdate(func(m *PullReqActivityMetadata) {
		m.Reactions = &PullReqActivityReactionsMetadata{Counts: map[string][]int64{"+1": {1}}}
	}))
	require.NotNil(t, a.Metadata)
	require.Equal(t, []int64{1}, a.Metadata.Reactions.Counts["+1"])

	a.UpdateMetadata(
		WithPullReqActivitySuggestionsMetadataUpdate(func(m *PullReqActivitySuggestionsMetadata) {
			m.CheckSums = []string{"abc"}
		}),
		WithPullReqActivityMentionsMetadataUpdate(func(m *PullReqActivityMentionsMetadata) {
			m.IDs = []int64{4}
			m.UserGroupIDs = []int64{5}
		}),
	)
	require.Equal(t, []string{"abc"}, a.Metadata.Suggestions.CheckSums)
	require.Equal(t, []int64{4}, a.Metadata.Mentions.IDs)

	// clearing every part drops the metadata altogether.
	a.UpdateMetadata(
		WithPullReqActivitySuggestionsMetadataUpdate(func(m *PullReqActivitySuggestionsMetadata) { m.CheckSums = nil }),
		WithPullReqActivityMentionsMetadataUpdate(func(m *PullReqActivityMentionsMetadata) {
			m.IDs, m.UserGroupIDs = nil, nil
		}),
		WithPullReqActivityMetadataUpdate(func(m *PullReqActivityMetadata) { m.Reactions = nil }),
	)
	require.Nil(t, a.Metadata)

	require.False(t, (&PullReqActivitySuggestionsMetadata{AppliedCheckSum: "x"}).IsEmpty())
	require.False(t, (&PullReqActivitySuggestionsMetadata{AppliedCommitSHA: "x"}).IsEmpty())
}

func TestGc5RepoActivityPayload(t *testing.T) {
	tests := []struct {
		name    string
		typ     enum.RepoActivityType
		payload RepoActivityPayload
	}{
		{"created", enum.RepoActivityTypeBranchCreated, &RepoActivityPayloadBranchCreated{Name: "main", New: "abc"}},
		{"updated", enum.RepoActivityTypeBranchUpdated,
			&RepoActivityPayloadBranchUpdated{Name: "main", Old: "a", New: "b", Forced: true}},
		{"deleted", enum.RepoActivityTypeBranchDeleted, &RepoActivityPayloadBranchDeleted{Name: "main", Old: "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.typ, tt.payload.ActivityType())

			raw, err := MarshalRepoActivityPayload(tt.payload)
			require.NoError(t, err)

			got, err := UnmarshalRepoActivityPayload(tt.typ, raw)
			require.NoError(t, err)
			require.Equal(t, tt.payload, got)

			// empty raw payloads yield a zero value of the right type.
			empty, err := UnmarshalRepoActivityPayload(tt.typ, nil)
			require.NoError(t, err)
			require.IsType(t, tt.payload, empty)
		})
	}

	raw, err := MarshalRepoActivityPayload(nil)
	require.NoError(t, err)
	require.JSONEq(t, "{}", string(raw))

	_, err = UnmarshalRepoActivityPayload(enum.RepoActivityType("unknown"), json.RawMessage(`{}`))
	require.Error(t, err)
	_, err = UnmarshalRepoActivityPayload(enum.RepoActivityTypeBranchCreated, json.RawMessage(`[1]`))
	require.Error(t, err)
	_, err = UnmarshalRepoActivityPayload(enum.RepoActivityType("unknown"), nil)
	require.Error(t, err)
}
