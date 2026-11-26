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
	"encoding/json"
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/rules"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestCtlSpaceController_Labels(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)
	s := e.session()

	label, err := e.ctrl.DefineLabel(ctx, s, "root", &types.DefineLabelInput{
		Key: "priority", Type: enum.LabelTypeStatic, Color: enum.LabelColorRed, Description: "  prio ",
	})
	require.NoError(t, err)
	require.Equal(t, "priority", label.Key)

	_, err = e.ctrl.DefineLabel(ctx, s, "root", &types.DefineLabelInput{Key: ""})
	require.Error(t, err)
	_, err = e.ctrl.DefineLabel(ctx, s, "root", &types.DefineLabelInput{
		Key: "priority", Type: enum.LabelTypeStatic, Color: enum.LabelColorRed,
	})
	require.Error(t, err, "duplicate label key")

	value, err := e.ctrl.DefineLabelValue(ctx, s, "root", "priority", &types.DefineValueInput{
		Value: "high", Color: enum.LabelColorGreen,
	})
	require.NoError(t, err)
	require.Equal(t, "high", value.Value)

	_, err = e.ctrl.DefineLabelValue(ctx, s, "root", "priority", &types.DefineValueInput{Value: ""})
	require.Error(t, err)
	_, err = e.ctrl.DefineLabelValue(ctx, s, "root", "missing", &types.DefineValueInput{
		Value: "x", Color: enum.LabelColorGreen,
	})
	require.Error(t, err)

	values, count, err := e.ctrl.ListLabelValues(ctx, s, "root", "priority", types.ListQueryFilter{
		Pagination: types.Pagination{Page: 1, Size: 10},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, values, 1)
	_, _, err = e.ctrl.ListLabelValues(ctx, s, "root", "missing", types.ListQueryFilter{})
	require.Error(t, err)

	newValue := "higher"
	updatedValue, err := e.ctrl.UpdateLabelValue(
		ctx, s, "root", "priority", "high", &types.UpdateValueInput{Value: &newValue})
	require.NoError(t, err)
	require.Equal(t, newValue, updatedValue.Value)
	_, err = e.ctrl.UpdateLabelValue(ctx, s, "root", "missing", "high", &types.UpdateValueInput{})
	require.Error(t, err)
	badValue := ""
	_, err = e.ctrl.UpdateLabelValue(ctx, s, "root", "priority", "higher", &types.UpdateValueInput{Value: &badValue})
	require.Error(t, err)

	found, err := e.ctrl.FindLabel(ctx, s, "root", "priority", false)
	require.NoError(t, err)
	require.Equal(t, "priority", found.Key)
	withValues, err := e.ctrl.FindLabel(ctx, s, "root", "priority", true)
	require.NoError(t, err)
	require.Len(t, withValues.Values, 1)
	_, err = e.ctrl.FindLabel(ctx, s, "root", "missing", false)
	require.Error(t, err)
	_, err = e.ctrl.FindLabel(ctx, s, "root", "missing", true)
	require.Error(t, err)

	labels, total, err := e.ctrl.ListLabels(ctx, s, "root", &types.LabelFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, labels, 1)

	desc := "updated"
	updated, err := e.ctrl.UpdateLabel(ctx, s, "root", "priority", &types.UpdateLabelInput{Description: &desc})
	require.NoError(t, err)
	require.Equal(t, desc, updated.Description)
	badKey := ""
	_, err = e.ctrl.UpdateLabel(ctx, s, "root", "priority", &types.UpdateLabelInput{Key: &badKey})
	require.Error(t, err)
	_, err = e.ctrl.UpdateLabel(ctx, s, "root", "missing", &types.UpdateLabelInput{})
	require.Error(t, err)

	saved, err := e.ctrl.SaveLabel(ctx, s, "root", &types.SaveInput{
		Label: types.SaveLabelInput{DefineLabelInput: types.DefineLabelInput{
			Key: "area", Type: enum.LabelTypeDynamic, Color: enum.LabelColorYellow,
		}},
		Values: []*types.SaveLabelValueInput{
			{DefineValueInput: types.DefineValueInput{Value: "ui", Color: enum.LabelColorGreen}},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "area", saved.Key)
	_, err = e.ctrl.SaveLabel(ctx, s, "root", &types.SaveInput{})
	require.Error(t, err)

	require.NoError(t, e.ctrl.DeleteLabelValue(ctx, s, "root", "priority", newValue))
	require.NoError(t, e.ctrl.DeleteLabel(ctx, s, "root", "priority"))
}

func TestCtlSpaceController_LabelAccess(t *testing.T) {
	ctx := context.Background()
	allow := ctlNewEnv(t, true)
	s := allow.session()

	for _, mode := range []ctlAuthMode{ctlDeny, ctlError} {
		c := *allow.ctrl
		c.authorizer = ctlAuthorizer{mode: mode}

		errs := map[string]error{}
		_, errs["define"] = c.DefineLabel(ctx, s, "root", &types.DefineLabelInput{})
		_, errs["save"] = c.SaveLabel(ctx, s, "root", &types.SaveInput{})
		_, errs["find"] = c.FindLabel(ctx, s, "root", "k", false)
		_, _, errs["list"] = c.ListLabels(ctx, s, "root", &types.LabelFilter{})
		_, errs["update"] = c.UpdateLabel(ctx, s, "root", "k", &types.UpdateLabelInput{})
		errs["delete"] = c.DeleteLabel(ctx, s, "root", "k")
		_, errs["define value"] = c.DefineLabelValue(ctx, s, "root", "k", &types.DefineValueInput{})
		_, _, errs["list values"] = c.ListLabelValues(ctx, s, "root", "k", types.ListQueryFilter{})
		_, errs["update value"] = c.UpdateLabelValue(ctx, s, "root", "k", "v", &types.UpdateValueInput{})
		errs["delete value"] = c.DeleteLabelValue(ctx, s, "root", "k", "v")
		_, errs["rule create"] = c.RuleCreate(ctx, s, "root", &rules.CreateInput{})
		_, _, errs["rule list"] = c.RuleList(ctx, s, "root", false, &types.RuleFilter{})
		_, errs["rule find"] = c.RuleFind(ctx, s, "root", "r")
		_, errs["rule update"] = c.RuleUpdate(ctx, s, "root", "r", &rules.UpdateInput{})
		errs["rule delete"] = c.RuleDelete(ctx, s, "root", "r")
		for name, err := range errs {
			require.Error(t, err, name)
			if mode == ctlError {
				require.ErrorIs(t, err, errCtlAuthz, name)
			} else {
				require.ErrorIs(t, err, apiauth.ErrForbidden, name)
			}
		}
	}
}

func TestCtlSpaceController_Rules(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)
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

	created, err := e.ctrl.RuleCreate(ctx, s, "root", in("rule1"))
	require.NoError(t, err)
	require.Equal(t, "rule1", created.Identifier)
	require.NotNil(t, created.SpaceID)

	// Invalid inputs.
	noDefinition := in("rule2")
	noDefinition.Definition = nil
	_, err = e.ctrl.RuleCreate(ctx, s, "root", noDefinition)
	require.Error(t, err)
	badState := in("rule3")
	badState.State = "nope"
	_, err = e.ctrl.RuleCreate(ctx, s, "root", badState)
	require.Error(t, err)
	_, err = e.ctrl.RuleCreate(ctx, s, "root", in(""))
	require.Error(t, err)
	_, err = e.ctrl.RuleCreate(ctx, s, "root", in("rule1"))
	require.Error(t, err, "duplicate identifier")

	found, err := e.ctrl.RuleFind(ctx, s, "root", "rule1")
	require.NoError(t, err)
	require.Equal(t, created.ID, found.ID)
	_, err = e.ctrl.RuleFind(ctx, s, "root", "missing")
	require.Error(t, err)

	list, count, err := e.ctrl.RuleList(ctx, s, "root", false, &types.RuleFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
		Order:           enum.OrderAsc,
		Sort:            enum.RuleSortCreated,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, list, 1)

	desc := "described"
	updated, err := e.ctrl.RuleUpdate(ctx, s, "root", "rule1", &rules.UpdateInput{Description: &desc})
	require.NoError(t, err)
	require.Equal(t, desc, updated.Description)
	_, err = e.ctrl.RuleUpdate(ctx, s, "root", "missing", &rules.UpdateInput{Description: &desc})
	require.Error(t, err)

	require.NoError(t, e.ctrl.RuleDelete(ctx, s, "root", "rule1"))
	require.Error(t, e.ctrl.RuleDelete(ctx, s, "root", "rule1"))
}

func TestCtlSpaceController_Restore(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)
	s := e.session()

	parent := e.createSpace(t, "parent", 0)
	e.createSpace(t, "kid", parent.ID)

	del, err := e.ctrl.SoftDelete(ctx, s, "parent")
	require.NoError(t, err)

	// Not allowed to pass parent refs without nested spaces being enabled.
	flat := *e.ctrl
	flat.nestedSpacesEnabled = false
	ref := "root"
	_, err = flat.Restore(ctx, s, "parent", del.DeletedAt, &RestoreInput{NewParentRef: &ref})
	require.ErrorIs(t, err, errNestedSpacesNotSupported)

	missing := "missing"
	_, err = e.ctrl.Restore(ctx, s, "parent", del.DeletedAt, &RestoreInput{NewParentRef: &missing})
	require.Error(t, err)

	// Unknown deletion timestamp.
	_, err = e.ctrl.Restore(ctx, s, "parent", del.DeletedAt+1000, &RestoreInput{})
	require.Error(t, err)

	newName := "reborn"
	restored, err := e.ctrl.Restore(ctx, s, "parent", del.DeletedAt, &RestoreInput{NewIdentifier: &newName})
	require.NoError(t, err)
	require.Equal(t, "reborn", restored.Identifier)
	require.False(t, restored.IsPublic)

	found, err := e.ctrl.Find(ctx, s, "reborn/kid")
	require.NoError(t, err)
	require.Equal(t, "kid", found.Identifier)

	// Denied restore.
	del2, err := e.ctrl.SoftDelete(ctx, s, "reborn")
	require.NoError(t, err)
	deny := *e.ctrl
	deny.authorizer = ctlAuthorizer{mode: ctlDeny}
	_, err = deny.Restore(ctx, s, "reborn", del2.DeletedAt, &RestoreInput{})
	require.ErrorIs(t, err, apiauth.ErrForbidden)
	require.ErrorIs(t, deny.Purge(ctx, s, "reborn", del2.DeletedAt), apiauth.ErrForbidden)
}

func TestCtlSpaceController_UsageMetricsNoData(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)

	metric, err := e.ctrl.GetUsageMetrics(ctx, e.session(), "root", 0, 1)
	require.NoError(t, err)
	require.NotNil(t, metric)
}

func TestCtlSpaceOutput_MarshalJSON(t *testing.T) {
	out := SpaceOutput{Space: types.Space{Identifier: "abc"}, IsPublic: true}
	data, err := json.Marshal(out)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, "abc", decoded["uid"])
	require.Equal(t, "abc", decoded["identifier"])
	require.Equal(t, true, decoded["is_public"])
}

func TestCtlGetSpaceCheckAuthRepoCreation(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)

	space, err := e.ctrl.getSpaceCheckAuthRepoCreation(ctx, e.session(), "root")
	require.NoError(t, err)
	require.Equal(t, e.root.ID, space.ID)

	deny := *e.ctrl
	deny.authorizer = ctlAuthorizer{mode: ctlDeny}
	_, err = deny.getSpaceCheckAuthRepoCreation(ctx, e.session(), "root")
	require.Error(t, err)
}
