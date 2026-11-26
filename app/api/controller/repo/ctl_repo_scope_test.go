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

package repo

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/harness/gitness/app/services/autolink"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/rules"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestCtlRepoController_Labels(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	label, err := e.ctrl.DefineLabel(ctx, s, "root/repo1", &types.DefineLabelInput{
		Key: "priority", Type: enum.LabelTypeStatic, Color: enum.LabelColorRed, Description: "  prio ",
	})
	require.NoError(t, err)
	require.Equal(t, "priority", label.Key)

	_, err = e.ctrl.DefineLabel(ctx, s, "root/repo1", &types.DefineLabelInput{Key: ""})
	require.Error(t, err)
	_, err = e.ctrl.DefineLabel(ctx, s, "root/repo1", &types.DefineLabelInput{
		Key: "priority", Type: enum.LabelTypeStatic, Color: enum.LabelColorRed,
	})
	require.Error(t, err, "duplicate label key")

	value, err := e.ctrl.DefineLabelValue(ctx, s, "root/repo1", "priority", &types.DefineValueInput{
		Value: "high", Color: enum.LabelColorGreen,
	})
	require.NoError(t, err)
	require.Equal(t, "high", value.Value)

	_, err = e.ctrl.DefineLabelValue(ctx, s, "root/repo1", "priority", &types.DefineValueInput{Value: ""})
	require.Error(t, err)
	_, err = e.ctrl.DefineLabelValue(ctx, s, "root/repo1", "missing", &types.DefineValueInput{
		Value: "x", Color: enum.LabelColorGreen,
	})
	require.Error(t, err)

	values, count, err := e.ctrl.ListLabelValues(ctx, s, "root/repo1", "priority", types.ListQueryFilter{
		Pagination: types.Pagination{Page: 1, Size: 10},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, values, 1)
	_, _, err = e.ctrl.ListLabelValues(ctx, s, "root/repo1", "missing", types.ListQueryFilter{})
	require.Error(t, err)

	newValue := "higher"
	updatedValue, err := e.ctrl.UpdateLabelValue(
		ctx, s, "root/repo1", "priority", "high", &types.UpdateValueInput{Value: &newValue})
	require.NoError(t, err)
	require.Equal(t, newValue, updatedValue.Value)
	_, err = e.ctrl.UpdateLabelValue(ctx, s, "root/repo1", "missing", "high", &types.UpdateValueInput{})
	require.Error(t, err)
	badValue := ""
	_, err = e.ctrl.UpdateLabelValue(ctx, s, "root/repo1", "priority", "higher", &types.UpdateValueInput{Value: &badValue})
	require.Error(t, err)

	found, err := e.ctrl.FindLabel(ctx, s, "root/repo1", "priority", false)
	require.NoError(t, err)
	require.Equal(t, "priority", found.Key)
	withValues, err := e.ctrl.FindLabel(ctx, s, "root/repo1", "priority", true)
	require.NoError(t, err)
	require.Len(t, withValues.Values, 1)
	_, err = e.ctrl.FindLabel(ctx, s, "root/repo1", "missing", false)
	require.Error(t, err)
	_, err = e.ctrl.FindLabel(ctx, s, "root/repo1", "missing", true)
	require.Error(t, err)

	labels, total, err := e.ctrl.ListLabels(ctx, s, "root/repo1", &types.LabelFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, labels, 1)

	desc := "updated"
	updated, err := e.ctrl.UpdateLabel(ctx, s, "root/repo1", "priority", &types.UpdateLabelInput{Description: &desc})
	require.NoError(t, err)
	require.Equal(t, desc, updated.Description)
	badKey := ""
	_, err = e.ctrl.UpdateLabel(ctx, s, "root/repo1", "priority", &types.UpdateLabelInput{Key: &badKey})
	require.Error(t, err)
	_, err = e.ctrl.UpdateLabel(ctx, s, "root/repo1", "missing", &types.UpdateLabelInput{})
	require.Error(t, err)

	saved, err := e.ctrl.SaveLabel(ctx, s, "root/repo1", &types.SaveInput{
		Label: types.SaveLabelInput{DefineLabelInput: types.DefineLabelInput{
			Key: "area", Type: enum.LabelTypeDynamic, Color: enum.LabelColorYellow,
		}},
		Values: []*types.SaveLabelValueInput{
			{DefineValueInput: types.DefineValueInput{Value: "ui", Color: enum.LabelColorGreen}},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "area", saved.Key)
	_, err = e.ctrl.SaveLabel(ctx, s, "root/repo1", &types.SaveInput{})
	require.Error(t, err)

	require.NoError(t, e.ctrl.DeleteLabelValue(ctx, s, "root/repo1", "priority", newValue))
	require.NoError(t, e.ctrl.DeleteLabel(ctx, s, "root/repo1", "priority"))
}

func TestCtlRepoController_Rules(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	in := func(identifier string) *rules.CreateInput {
		return &rules.CreateInput{
			Type:       enum.RuleTypeBranch,
			State:      enum.RuleStateActive,
			Identifier: identifier,
			Pattern:    protection.Pattern{Default: true},
			Definition: json.RawMessage(`{"lifecycle":{"delete_forbidden":true}}`),
		}
	}

	created, err := e.ctrl.RuleCreate(ctx, s, "root/repo1", in("rule1"))
	require.NoError(t, err)
	require.Equal(t, "rule1", created.Identifier)
	require.NotNil(t, created.RepoID)

	// Invalid inputs.
	noDefinition := in("rule2")
	noDefinition.Definition = nil
	_, err = e.ctrl.RuleCreate(ctx, s, "root/repo1", noDefinition)
	require.Error(t, err)
	badState := in("rule3")
	badState.State = "nope"
	_, err = e.ctrl.RuleCreate(ctx, s, "root/repo1", badState)
	require.Error(t, err)
	_, err = e.ctrl.RuleCreate(ctx, s, "root/repo1", in(""))
	require.Error(t, err)
	_, err = e.ctrl.RuleCreate(ctx, s, "root/repo1", in("rule1"))
	require.Error(t, err, "duplicate identifier")

	found, err := e.ctrl.RuleFind(ctx, s, "root/repo1", "rule1")
	require.NoError(t, err)
	require.Equal(t, created.ID, found.ID)
	_, err = e.ctrl.RuleFind(ctx, s, "root/repo1", "missing")
	require.Error(t, err)

	list, count, err := e.ctrl.RuleList(ctx, s, "root/repo1", false, &types.RuleFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
		Order:           enum.OrderAsc,
		Sort:            enum.RuleSortCreated,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, list, 1)

	desc := "described"
	updated, err := e.ctrl.RuleUpdate(ctx, s, "root/repo1", "rule1", &rules.UpdateInput{Description: &desc})
	require.NoError(t, err)
	require.Equal(t, desc, updated.Description)
	_, err = e.ctrl.RuleUpdate(ctx, s, "root/repo1", "missing", &rules.UpdateInput{Description: &desc})
	require.Error(t, err)

	require.NoError(t, e.ctrl.RuleDelete(ctx, s, "root/repo1", "rule1"))
	require.Error(t, e.ctrl.RuleDelete(ctx, s, "root/repo1", "rule1"))
}

func TestCtlRepoController_Autolinks(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	created, err := e.ctrl.AutolinkCreate(ctx, s, "root/repo1", &autolink.AutoLinkInput{
		Type: enum.AutoLinkTypeRegex, Pattern: `JIRA-(\d+)`, TargetURL: "https://jira.example.com/<num>",
	})
	require.NoError(t, err)

	found, err := e.ctrl.AutolinkFind(ctx, s, "root/repo1", created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, found.ID)

	list, count, err := e.ctrl.AutolinkList(ctx, s, "root/repo1", &types.AutoLinkFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, list, 1)

	pattern := `BUG-(\d+)`
	updated, err := e.ctrl.AutolinkUpdate(
		ctx, s, "root/repo1", created.ID, &autolink.AutoLinkUpdateInput{Pattern: &pattern})
	require.NoError(t, err)
	require.Equal(t, pattern, updated.Pattern)

	_, err = e.ctrl.AutolinkCreate(ctx, s, "root/repo1", &autolink.AutoLinkInput{})
	require.Error(t, err)
	_, err = e.ctrl.AutolinkUpdate(ctx, s, "root/repo1", created.ID+100, &autolink.AutoLinkUpdateInput{Pattern: &pattern})
	require.Error(t, err)
	_, err = e.ctrl.AutolinkFind(ctx, s, "root/repo1", created.ID+100)
	require.Error(t, err)

	require.NoError(t, e.ctrl.AutolinkDelete(ctx, s, "root/repo1", created.ID))
	require.Error(t, e.ctrl.AutolinkDelete(ctx, s, "root/repo1", created.ID))
}
