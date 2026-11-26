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

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestMaskSecret(t *testing.T) {
	s := NewMaskSecret("hunter2")
	require.Equal(t, "hunter2", s.Value())
	require.Len(t, s.String(), maxTruncatedLen)
	require.NotContains(t, s.String(), "hunter2")
	require.Equal(t, s.String(), NewMaskSecret("hunter2").String())
	require.NotEqual(t, s.String(), NewMaskSecret("other").String())

	// a secret that was not built by the constructor still masks its value
	zero := MaskSecret{value: "hunter2"}
	require.Equal(t, s.String(), zero.String())
	require.Empty(t, MaskSecret{}.String())

	data, err := json.Marshal(s)
	require.NoError(t, err)
	require.JSONEq(t, `"hunter2"`, string(data))

	var decoded MaskSecret
	require.NoError(t, json.Unmarshal([]byte(`"abc"`), &decoded))
	require.Equal(t, "abc", decoded.Value())
	require.Error(t, json.Unmarshal([]byte(`123`), &decoded))
}

func TestSanitizeTag(t *testing.T) {
	require.NoError(t, SanitizeTag(nil, TagPartTypeKey, true))

	text := "  trimmed  "
	require.NoError(t, SanitizeTag(&text, TagPartTypeKey, true))
	require.Equal(t, "trimmed", text)

	empty := "   "
	err := SanitizeTag(&empty, TagPartTypeValue, true)
	require.True(t, errors.IsInvalidArgument(err))
	require.Contains(t, err.Error(), "value")
	empty = ""
	require.NoError(t, SanitizeTag(&empty, TagPartTypeValue, false))

	long := make([]rune, 51)
	for i := range long {
		long[i] = 'é'
	}
	longText := string(long)
	require.True(t, errors.IsInvalidArgument(SanitizeTag(&longText, TagPartTypeKey, false)))
	longText = string(long[:50])
	require.NoError(t, SanitizeTag(&longText, TagPartTypeKey, false))

	control := "bad\tvalue"
	require.True(t, errors.IsInvalidArgument(SanitizeTag(&control, TagPartTypeKey, false)))
}

func TestPrincipalConversions(t *testing.T) {
	p := &Principal{ID: 1, UID: "jane", Email: "j@x.io", DisplayName: "Jane", Type: enum.PrincipalTypeUser,
		Created: 10, Updated: 20}
	require.False(t, p.IsAnonymous())
	require.True(t, (&Principal{UID: AnonymousPrincipalUID}).IsAnonymous())

	info := p.ToPrincipalInfo()
	require.Equal(t, &PrincipalInfo{ID: 1, UID: "jane", DisplayName: "Jane", Email: "j@x.io",
		Type: enum.PrincipalTypeUser, Created: 10, Updated: 20}, info)
	require.Equal(t, int64(1), info.Identifier())

	svc := &Service{ID: 2, UID: "svc", Email: "s@x.io", DisplayName: "Svc", Admin: true, Blocked: true,
		Salt: "salt", Created: 1, Updated: 2}
	principal := svc.ToPrincipal()
	require.Equal(t, enum.PrincipalTypeService, principal.Type)
	require.Equal(t, "salt", principal.Salt)
	require.True(t, principal.Admin)
	require.True(t, principal.Blocked)
	require.Equal(t, int64(2), svc.ToPrincipalInfo().ID)
	require.Equal(t, enum.PrincipalTypeService, svc.ToPrincipalInfo().Type)

	sa := &ServiceAccount{ID: 3, UID: "sa", Email: "sa@x.io", DisplayName: "SA", ParentType: enum.ParentResourceTypeSpace,
		ParentID: 9}
	require.Equal(t, enum.PrincipalTypeServiceAccount, sa.ToPrincipal().Type)
	require.Equal(t, enum.PrincipalTypeServiceAccount, sa.ToPrincipalInfo().Type)
	saInfo := sa.ToServiceAccountInfo()
	require.Equal(t, int64(3), saInfo.ID)
	require.Equal(t, enum.ParentResourceTypeSpace, saInfo.ParentType)
	require.Equal(t, int64(9), saInfo.ParentID)
}

func TestSecret(t *testing.T) {
	s := Secret{ID: 1, Description: "d", SpaceID: 2, Identifier: "my-secret", Data: "top-secret", Version: 3,
		Created: 4, Updated: 5}

	data, err := json.Marshal(s)
	require.NoError(t, err)
	require.JSONEq(t, `{"description":"d","space_id":2,"created_by":0,"identifier":"my-secret",`+
		`"created":4,"updated":5,"uid":"my-secret"}`, string(data))
	require.NotContains(t, string(data), "top-secret")

	cp := s.CopyWithoutData()
	require.Empty(t, cp.Data)
	require.Equal(t, s.Identifier, cp.Identifier)
	require.Equal(t, s.Version, cp.Version)
	require.Equal(t, s.SpaceID, cp.SpaceID)
	require.Equal(t, "top-secret", s.Data)
}

func TestPlugin(t *testing.T) {
	p := &Plugin{Identifier: "id", Description: "d", Type: "t", Version: "1", Logo: "l", Spec: "s"}
	same := *p
	require.True(t, p.Matches(&same))

	for name, mutate := range map[string]func(*Plugin){
		"identifier":  func(x *Plugin) { x.Identifier = "x" },
		"description": func(x *Plugin) { x.Description = "x" },
		"spec":        func(x *Plugin) { x.Spec = "x" },
		"version":     func(x *Plugin) { x.Version = "x" },
		"logo":        func(x *Plugin) { x.Logo = "x" },
	} {
		other := *p
		mutate(&other)
		require.False(t, p.Matches(&other), name)
	}

	data, err := json.Marshal(*p)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, "id", decoded["uid"])
	require.Equal(t, "id", decoded["identifier"])
}

func TestLabelInputSanitize(t *testing.T) {
	t.Run("define label", func(t *testing.T) {
		in := DefineLabelInput{Key: "  bug ", Description: "  d  ", Type: " STATIC ", Color: " Blue "}
		require.NoError(t, in.Sanitize())
		require.Equal(t, "bug", in.Key)
		require.Equal(t, "d", in.Description)
		require.Equal(t, enum.LabelTypeStatic, in.Type)
		require.Equal(t, enum.LabelColorBlue, in.Color)

		require.Error(t, (&DefineLabelInput{Key: " ", Type: enum.LabelTypeStatic, Color: enum.LabelColorBlue}).Sanitize())
		require.Error(t, (&DefineLabelInput{Key: "k", Type: "nope", Color: enum.LabelColorBlue}).Sanitize())
		require.Error(t, (&DefineLabelInput{Key: "k", Type: enum.LabelTypeStatic, Color: "nope"}).Sanitize())
	})

	t.Run("update label", func(t *testing.T) {
		key, desc := "  key ", "  desc "
		typ, color := enum.LabelType(" Dynamic "), enum.LabelColor(" RED ")
		in := UpdateLabelInput{Key: &key, Description: &desc, Type: &typ, Color: &color}
		require.NoError(t, in.Sanitize())
		require.Equal(t, "key", key)
		require.Equal(t, "desc", desc)
		require.Equal(t, enum.LabelTypeDynamic, typ)
		require.Equal(t, enum.LabelColorRed, color)

		require.NoError(t, (&UpdateLabelInput{}).Sanitize())

		blank := " "
		require.Error(t, (&UpdateLabelInput{Key: &blank}).Sanitize())
		badType := enum.LabelType("x")
		require.Error(t, (&UpdateLabelInput{Type: &badType}).Sanitize())
		badColor := enum.LabelColor("x")
		require.Error(t, (&UpdateLabelInput{Color: &badColor}).Sanitize())
	})

	t.Run("define value", func(t *testing.T) {
		in := DefineValueInput{Value: " v ", Color: "GREEN"}
		require.NoError(t, in.Sanitize())
		require.Equal(t, "v", in.Value)
		require.Equal(t, enum.LabelColorGreen, in.Color)

		require.Error(t, (&DefineValueInput{Value: "", Color: enum.LabelColorGreen}).Sanitize())
		require.Error(t, (&DefineValueInput{Value: "v", Color: "x"}).Sanitize())
	})

	t.Run("update value", func(t *testing.T) {
		v := " v "
		color := enum.LabelColor("Blue")
		require.NoError(t, (&UpdateValueInput{Value: &v, Color: &color}).Sanitize())
		require.Equal(t, "v", v)
		require.NoError(t, (&UpdateValueInput{}).Sanitize())

		blank := ""
		require.Error(t, (&UpdateValueInput{Value: &blank}).Sanitize())
		badColor := enum.LabelColor("x")
		require.Error(t, (&UpdateValueInput{Color: &badColor}).Sanitize())
	})

	t.Run("save input", func(t *testing.T) {
		valid := func() *SaveInput {
			return &SaveInput{
				Label: SaveLabelInput{DefineLabelInput: DefineLabelInput{
					Key: " k ", Type: enum.LabelTypeStatic, Color: enum.LabelColorBlue,
				}},
				Values: []*SaveLabelValueInput{{DefineValueInput: DefineValueInput{Value: " v ", Color: "red"}}},
			}
		}
		in := valid()
		require.NoError(t, in.Sanitize())
		require.Equal(t, "k", in.Label.Key)
		require.Equal(t, "v", in.Values[0].Value)

		badLabel := valid()
		badLabel.Label.Key = ""
		require.Error(t, badLabel.Sanitize())

		badValue := valid()
		badValue.Values[0].Color = "x"
		require.Error(t, badValue.Sanitize())
	})

	t.Run("assign input validate", func(t *testing.T) {
		one := int64(1)
		zero := int64(0)
		require.NoError(t, PullReqLabelAssignInput{LabelID: 1}.Validate())
		require.NoError(t, PullReqLabelAssignInput{ValueID: &one}.Validate())
		require.NoError(t, PullReqLabelAssignInput{ValueID: &zero, Value: "v"}.Validate())
		require.NoError(t, PullReqLabelAssignInput{Value: "v"}.Validate())
		require.True(t, errors.IsInvalidArgument(PullReqLabelAssignInput{ValueID: &one, Value: "v"}.Validate()))
	})
}

func TestRepoTagsSanitize(t *testing.T) {
	require.NoError(t, RepoTags(nil).Sanitize())
	require.NoError(t, RepoTags{}.Sanitize())

	tags := RepoTags{"  key ": " value "}
	require.NoError(t, tags.Sanitize())
	require.Equal(t, RepoTags{"key": "value"}, tags)

	require.True(t, errors.IsInvalidArgument(RepoTags{"a:b": "v"}.Sanitize()))
	require.True(t, errors.IsInvalidArgument(RepoTags{"k": "v:w"}.Sanitize()))
}

func TestRepositoryHelpers(t *testing.T) {
	deleted := int64(5)
	repo := Repository{
		ID: 1, ParentID: 2, Identifier: "r", Path: "s/r", GitUID: "uid", DefaultBranch: "main", ForkID: 3,
		State: enum.RepoStateArchived, Type: enum.RepoTypeLinked, Deleted: &deleted,
	}

	core := repo.Core()
	require.Equal(t, &RepositoryCore{
		ID: 1, ParentID: 2, Identifier: "r", Path: "s/r", GitUID: "uid", DefaultBranch: "main", ForkID: 3,
		State: enum.RepoStateArchived, Type: enum.RepoTypeLinked,
	}, core)
	require.Equal(t, "uid", core.GetGitUID())
	require.Equal(t, "uid", repo.GetGitUID())

	clone := repo.Clone()
	require.Equal(t, repo.Deleted, clone.Deleted)
	require.NotSame(t, repo.Deleted, clone.Deleted)
	*clone.Deleted = 99
	require.Equal(t, int64(5), *repo.Deleted)

	require.Nil(t, (Repository{}).Clone().Deleted)
}

func TestPullReqHelpers(t *testing.T) {
	t.Run("update merge outcome", func(t *testing.T) {
		pr := &PullReq{}

		pr.UpdateMergeOutcome(enum.MergeMethodMerge, []string{"a.txt"})
		require.Equal(t, enum.MergeCheckStatusConflict, pr.MergeCheckStatus)
		require.Equal(t, []string{"a.txt"}, pr.MergeConflicts)

		pr.UpdateMergeOutcome(enum.MergeMethodSquash, nil)
		require.Equal(t, enum.MergeCheckStatusMergeable, pr.MergeCheckStatus)
		require.Nil(t, pr.MergeConflicts)

		pr.UpdateMergeOutcome(enum.MergeMethodRebase, []string{"b.txt"})
		require.Equal(t, enum.MergeCheckStatusConflict, pr.RebaseCheckStatus)
		require.Equal(t, []string{"b.txt"}, pr.RebaseConflicts)

		pr.UpdateMergeOutcome(enum.MergeMethodRebase, nil)
		require.Equal(t, enum.MergeCheckStatusMergeable, pr.RebaseCheckStatus)
		require.Nil(t, pr.RebaseConflicts)

		before := *pr
		pr.UpdateMergeOutcome(enum.MergeMethodFastForward, []string{"c.txt"})
		require.Equal(t, before, *pr)
	})

	t.Run("state markers", func(t *testing.T) {
		pr := &PullReq{
			MergeConflicts: []string{"a"}, RebaseConflicts: []string{"b"},
			MergeCheckStatus: enum.MergeCheckStatusConflict, RebaseCheckStatus: enum.MergeCheckStatusConflict,
		}
		pr.MarkAsMergeUnchecked()
		require.Equal(t, enum.MergeCheckStatusUnchecked, pr.MergeCheckStatus)
		require.Equal(t, enum.MergeCheckStatusUnchecked, pr.RebaseCheckStatus)
		require.Nil(t, pr.MergeConflicts)
		require.Nil(t, pr.RebaseConflicts)

		pr.MergeConflicts = []string{"a"}
		pr.MarkAsMergeable()
		require.Equal(t, enum.MergeCheckStatusMergeable, pr.MergeCheckStatus)
		require.Nil(t, pr.MergeConflicts)
		require.Equal(t, enum.MergeCheckStatusUnchecked, pr.RebaseCheckStatus)

		pr.RebaseConflicts = []string{"b"}
		pr.MarkAsRebaseable()
		require.Equal(t, enum.MergeCheckStatusMergeable, pr.RebaseCheckStatus)
		require.Nil(t, pr.RebaseConflicts)

		other := &PullReq{MergeConflicts: []string{"x"}, RebaseConflicts: []string{"y"}}
		other.MarkAsMerged()
		require.Equal(t, enum.MergeCheckStatusMergeable, other.MergeCheckStatus)
		require.Equal(t, enum.MergeCheckStatusMergeable, other.RebaseCheckStatus)
		require.Nil(t, other.MergeConflicts)
		require.Nil(t, other.RebaseConflicts)
	})

	t.Run("linked", func(t *testing.T) {
		require.False(t, (&PullReq{}).IsLinked())
		normal := enum.PullReqTypeNormal
		require.False(t, (&PullReq{Type: &normal}).IsLinked())
		linked := enum.PullReqTypeLinked
		require.True(t, (&PullReq{Type: &linked}).IsLinked())
	})

	t.Run("diff stats", func(t *testing.T) {
		stats := NewDiffStats(1, 2, 3, 4)
		require.Equal(t, int64(1), *stats.Commits)
		require.Equal(t, int64(2), *stats.FilesChanged)
		require.Equal(t, int64(3), *stats.Additions)
		require.Equal(t, int64(4), *stats.Deletions)
	})
}

func TestRuleHelpers(t *testing.T) {
	repoID := int64(7)
	rule := Rule{
		ID: 1, Identifier: "rule", Description: "d", Type: enum.RuleTypeBranch, State: enum.RuleStateActive,
		RepoID:     &repoID,
		Pattern:    json.RawMessage(`{"default":true}`),
		RepoTarget: json.RawMessage(`{}`),
		Definition: json.RawMessage(`{"bypass":{}}`),
		Users:      map[int64]*PrincipalInfo{1: {ID: 1, UID: "u"}},
	}

	t.Run("json", func(t *testing.T) {
		data, err := json.Marshal(rule)
		require.NoError(t, err)
		var m map[string]any
		require.NoError(t, json.Unmarshal(data, &m))
		require.Equal(t, "rule", m["uid"])
		require.Equal(t, "rule", m["identifier"])
	})

	t.Run("yaml", func(t *testing.T) {
		out, err := rule.MarshalYAML()
		require.NoError(t, err)
		m, ok := out.(map[string]any)
		require.True(t, ok)
		require.Equal(t, "rule", m["identifier"])
		require.Equal(t, enum.RuleTypeBranch, m["type"])
		require.NotNil(t, m["pattern"])
		require.NotNil(t, m["definition"])

		bad := rule
		bad.Pattern = json.RawMessage(`: not yaml :`)
		_, err = bad.MarshalYAML()
		require.Error(t, err)

		badDef := rule
		badDef.Definition = json.RawMessage(`: not yaml :`)
		_, err = badDef.MarshalYAML()
		require.Error(t, err)
	})

	t.Run("clone is deep", func(t *testing.T) {
		spaceID := int64(8)
		withSpace := rule
		withSpace.SpaceID = &spaceID

		clone := withSpace.Clone()
		require.Equal(t, *withSpace.RepoID, *clone.RepoID)
		require.NotSame(t, withSpace.RepoID, clone.RepoID)
		require.NotSame(t, withSpace.SpaceID, clone.SpaceID)

		clone.Pattern[0] = 'X'
		clone.Definition[0] = 'X'
		clone.Users[1].UID = "changed"
		require.Equal(t, byte('{'), withSpace.Pattern[0])
		require.Equal(t, byte('{'), withSpace.Definition[0])
		require.Equal(t, "u", withSpace.Users[1].UID)

		empty := Rule{}.Clone()
		require.Nil(t, empty.RepoID)
		require.Nil(t, empty.SpaceID)
	})

	t.Run("is equal", func(t *testing.T) {
		same := rule.Clone()
		require.True(t, rule.IsEqual(&same))

		for name, mutate := range map[string]func(*Rule){
			"identifier":  func(r *Rule) { r.Identifier = "x" },
			"state":       func(r *Rule) { r.State = enum.RuleStateDisabled },
			"description": func(r *Rule) { r.Description = "x" },
			"pattern":     func(r *Rule) { r.Pattern = json.RawMessage(`{}`) },
			"repo target": func(r *Rule) { r.RepoTarget = json.RawMessage(`{"a":1}`) },
			"definition":  func(r *Rule) { r.Definition = json.RawMessage(`{}`) },
		} {
			other := rule.Clone()
			mutate(&other)
			require.False(t, rule.IsEqual(&other), name)
		}
	})

	t.Run("rule info json", func(t *testing.T) {
		data, err := json.Marshal(RuleInfo{Identifier: "info", Type: enum.RuleTypePush, State: enum.RuleStateMonitor})
		require.NoError(t, err)
		var m map[string]any
		require.NoError(t, json.Unmarshal(data, &m))
		require.Equal(t, "info", m["uid"])
	})
}

func TestRuleViolations(t *testing.T) {
	v := &RuleViolations{Rule: RuleInfo{State: enum.RuleStateActive}}
	require.False(t, v.IsCritical())
	require.False(t, v.IsBypassed())

	v.Add("code1", "message")
	v.Addf("code2", "value %d is %s", 5, "bad")
	require.Len(t, v.Violations, 2)
	require.Equal(t, Violation{Code: "code1", Message: "message"}, v.Violations[0])
	require.Equal(t, "value 5 is bad", v.Violations[1].Message)
	require.Equal(t, []any{5, "bad"}, v.Violations[1].Params)

	require.True(t, v.IsCritical())
	require.False(t, v.IsBypassed())

	v.Bypassed = true
	require.False(t, v.IsCritical())
	require.True(t, v.IsBypassed())

	v.Rule.State = enum.RuleStateMonitor
	require.False(t, v.IsCritical())
	require.False(t, v.IsBypassed())
}
