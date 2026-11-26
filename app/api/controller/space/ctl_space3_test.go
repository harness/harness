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
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/services/importer"

	"github.com/stretchr/testify/require"
)

func TestCtlSpaceController_Events(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)

	_, _, _, err := e.ctrl.Events(ctx, e.session(), "root")
	require.NoError(t, err)
	_, _, _, err = e.ctrl.Events(ctx, e.session(), "missing")
	require.Error(t, err)

	deny := *e.ctrl
	deny.authorizer = ctlAuthorizer{mode: ctlDeny}
	_, _, _, err = deny.Events(ctx, e.session(), "root")
	require.ErrorIs(t, err, apiauth.ErrForbidden)
}

func TestCtlSpaceController_ExportInput(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)
	valid := func() *ExportInput {
		return &ExportInput{AccountID: "acc", OrgIdentifier: "org", ProjectIdentifier: "proj", Token: "tok"}
	}

	require.NoError(t, e.ctrl.sanitizeExportInput(valid()))

	for name, mutate := range map[string]func(in *ExportInput){
		"account": func(in *ExportInput) { in.AccountID = "" },
		"org":     func(in *ExportInput) { in.OrgIdentifier = "" },
		"project": func(in *ExportInput) { in.ProjectIdentifier = "" },
		"token":   func(in *ExportInput) { in.Token = "" },
	} {
		in := valid()
		mutate(in)
		require.Error(t, e.ctrl.sanitizeExportInput(in), name)
		require.Error(t, e.ctrl.Export(ctx, e.session(), "root", in), name)
	}

	require.Error(t, e.ctrl.Export(ctx, e.session(), "missing", valid()))
	_, err := e.ctrl.ExportProgress(ctx, e.session(), "missing")
	require.Error(t, err)

	deny := *e.ctrl
	deny.authorizer = ctlAuthorizer{mode: ctlDeny}
	require.ErrorIs(t, deny.Export(ctx, e.session(), "root", valid()), apiauth.ErrForbidden)
	_, err = deny.ExportProgress(ctx, e.session(), "root")
	require.ErrorIs(t, err, apiauth.ErrForbidden)
}

func TestCtlSpaceController_ImportInput(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)

	in := &ImportInput{CreateInput: CreateInput{Identifier: "imported"}}
	require.NoError(t, e.ctrl.sanitizeImportInput(in))
	require.Equal(t, importer.PipelineOptionConvert, in.Pipelines)

	require.Error(t, e.ctrl.sanitizeImportInput(&ImportInput{CreateInput: CreateInput{Identifier: ""}}))

	// Import with an unknown parent space fails before anything is created.
	_, err := e.ctrl.Import(ctx, e.session(), &ImportInput{
		CreateInput: CreateInput{ParentRef: "missing", Identifier: "x"},
	})
	require.Error(t, err)

	// Invalid identifier derived from the provider space.
	_, err = e.ctrl.Import(ctx, e.session(), &ImportInput{
		ProviderInput: ProviderInput{ProviderSpace: ""},
	})
	require.Error(t, err)

	_, err = e.ctrl.ImportRepositories(ctx, e.session(), "missing", &ImportRepositoriesInput{})
	require.Error(t, err)
}
