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

package template

import (
	"strings"
	"testing"

	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const (
	cxrStepYAML  = "version: 1\nkind: template\ntype: step\nspec:\n  run: echo hello\n"
	cxrStageYAML = "version: 1\nkind: template\ntype: stage\nspec:\n  steps: []\n"
)

func cxrNewController(b *cxrBase, mode cxrAuthMode) *Controller {
	return NewController(cxrAuthorizer{mode: mode}, database.NewTemplateStore(b.db), b.spaceFinder)
}

func TestCxrTemplateParseResolverType(t *testing.T) {
	got, err := parseResolverType(cxrStepYAML)
	require.NoError(t, err)
	require.Equal(t, enum.ResolverTypeStep, got)

	got, err = parseResolverType(cxrStageYAML)
	require.NoError(t, err)
	require.Equal(t, enum.ResolverTypeStage, got)

	_, err = parseResolverType("type: [unterminated")
	require.ErrorContains(t, err, "could not parse template data")

	_, err = parseResolverType("version: 1\nkind: template\ntype: bogus\n")
	require.ErrorContains(t, err, "could not parse template data")
}

func TestCxrTemplateLifecycle(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	b.createSpace(ctx, t)
	c := cxrNewController(b, cxrAllow)

	created, err := c.Create(ctx, b.session(), &CreateInput{
		SpaceRef: "root", UID: "tpl", Description: " d ", Data: cxrStepYAML,
	})
	require.NoError(t, err)
	require.Equal(t, enum.ResolverTypeStep, created.Type)
	require.Equal(t, "tpl", created.Identifier)

	found, err := c.Find(ctx, b.session(), "root", "tpl", enum.ResolverTypeStep)
	require.NoError(t, err)
	require.Equal(t, cxrStepYAML, found.Data)

	newID, newDesc := "tpl2", "updated"
	updated, err := c.Update(ctx, b.session(), "root", "tpl", enum.ResolverTypeStep, &UpdateInput{
		Identifier: &newID, Description: &newDesc,
	})
	require.NoError(t, err)
	require.Equal(t, newID, updated.Identifier)
	require.Equal(t, newDesc, updated.Description)

	uid, data := "tpl3", cxrStageYAML
	updated, err = c.Update(ctx, b.session(), "root", newID, enum.ResolverTypeStep, &UpdateInput{
		UID: &uid, Data: &data,
	})
	require.NoError(t, err)
	require.Equal(t, uid, updated.Identifier)
	require.Equal(t, enum.ResolverTypeStage, updated.Type)
	require.Equal(t, cxrStageYAML, updated.Data)

	require.NoError(t, c.Delete(ctx, b.session(), "root", uid, enum.ResolverTypeStage))
	_, err = c.Find(ctx, b.session(), "root", uid, enum.ResolverTypeStage)
	require.ErrorContains(t, err, "failed to find template")
}

func TestCxrTemplateCreateErrors(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	b.createSpace(ctx, t)

	tests := []struct {
		name string
		mode cxrAuthMode
		in   *CreateInput
		want string
	}{
		{"empty space ref", cxrAllow, &CreateInput{Identifier: "abc", Data: cxrStepYAML}, "Parent space required"},
		{"zero space id", cxrAllow, &CreateInput{SpaceRef: "0", Identifier: "abc", Data: cxrStepYAML},
			"Parent space required"},
		{"bad identifier", cxrAllow, &CreateInput{SpaceRef: "root", Identifier: "!!", Data: cxrStepYAML}, ""},
		{"bad data", cxrAllow, &CreateInput{SpaceRef: "root", Identifier: "abc", Data: "type: [x"},
			"could not parse template data"},
		{"bad description", cxrAllow, &CreateInput{SpaceRef: "root", Identifier: "abc", Data: cxrStepYAML,
			Description: strings.Repeat("x", 1100)}, ""},
		{"unknown space", cxrAllow, &CreateInput{SpaceRef: "nope", Identifier: "abc", Data: cxrStepYAML},
			"failed to find parent"},
		{"forbidden", cxrDeny, &CreateInput{SpaceRef: "root", Identifier: "abc", Data: cxrStepYAML}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cxrNewController(b, tt.mode).Create(ctx, b.session(), tt.in)
			require.Error(t, err)
			require.Nil(t, got)
			require.Contains(t, err.Error(), tt.want)
		})
	}

	c := cxrNewController(b, cxrAllow)
	in := CreateInput{SpaceRef: "root", Identifier: "dup", Data: cxrStepYAML}
	dup := in
	_, err := c.Create(ctx, b.session(), &in)
	require.NoError(t, err)
	_, err = c.Create(ctx, b.session(), &dup)
	require.ErrorContains(t, err, "template creation failed")
}

func TestCxrTemplateFindUpdateDeleteErrors(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	b.createSpace(ctx, t)
	allow := cxrNewController(b, cxrAllow)
	deny := cxrNewController(b, cxrDeny)
	_, err := allow.Create(ctx, b.session(), &CreateInput{SpaceRef: "root", Identifier: "t1", Data: cxrStepYAML})
	require.NoError(t, err)

	step := enum.ResolverTypeStep
	badID, badData := "!!", "type: [x"
	longDesc := strings.Repeat("x", 1100)

	_, err = allow.Find(ctx, b.session(), "nope", "t1", step)
	require.ErrorContains(t, err, "failed to find space")
	_, err = deny.Find(ctx, b.session(), "root", "t1", step)
	require.ErrorContains(t, err, "failed to authorize")
	_, err = allow.Find(ctx, b.session(), "root", "missing", step)
	require.ErrorContains(t, err, "failed to find template")

	_, err = allow.Update(ctx, b.session(), "root", "t1", step, &UpdateInput{Identifier: &badID})
	require.Error(t, err)
	_, err = allow.Update(ctx, b.session(), "root", "t1", step, &UpdateInput{Description: &longDesc})
	require.Error(t, err)
	_, err = allow.Update(ctx, b.session(), "root", "t1", step, &UpdateInput{Data: &badData})
	require.Error(t, err)
	_, err = allow.Update(ctx, b.session(), "nope", "t1", step, &UpdateInput{})
	require.ErrorContains(t, err, "failed to find space")
	_, err = deny.Update(ctx, b.session(), "root", "t1", step, &UpdateInput{})
	require.ErrorContains(t, err, "failed to authorize")
	_, err = allow.Update(ctx, b.session(), "root", "missing", step, &UpdateInput{})
	require.ErrorContains(t, err, "failed to find template")

	require.ErrorContains(t, allow.Delete(ctx, b.session(), "nope", "t1", step), "failed to find space")
	require.ErrorContains(t, deny.Delete(ctx, b.session(), "root", "t1", step), "failed to authorize")
}
