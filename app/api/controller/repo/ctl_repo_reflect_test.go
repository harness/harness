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
	"fmt"
	"reflect"
	"sort"
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"

	"github.com/stretchr/testify/require"
)

var (
	ctlCtxType     = reflect.TypeOf((*context.Context)(nil)).Elem()
	ctlSessionType = reflect.TypeOf((*auth.Session)(nil))
	ctlErrorType   = reflect.TypeOf((*error)(nil)).Elem()
)

// ctlRepoRefMethods returns the names of all exported controller methods with the
// (ctx, session, repoRef, ...) shape, which all start with a repository lookup and access check.
func ctlRepoRefMethods(c *Controller) []string {
	typ := reflect.TypeOf(c)
	var names []string
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		mt := m.Type // receiver is the first parameter.
		if mt.NumIn() < 4 || mt.In(1) != ctlCtxType || mt.In(2) != ctlSessionType || mt.In(3).Kind() != reflect.String {
			continue
		}
		names = append(names, m.Name)
	}
	sort.Strings(names)
	return names
}

// ctlInvoke calls the named method with a valid session, the given repo ref and zero valued remaining
// arguments. It returns the returned error (nil if none) and recovers panics.
func ctlInvoke(t *testing.T, c *Controller, name string, session *auth.Session, repoRef string) (err error) {
	t.Helper()
	m := reflect.ValueOf(c).MethodByName(name)
	mt := m.Type()

	args := make([]reflect.Value, mt.NumIn())
	seenString := false
	for i := range args {
		pt := mt.In(i)
		switch {
		case pt == ctlCtxType:
			args[i] = reflect.ValueOf(t.Context())
		case pt == ctlSessionType:
			args[i] = reflect.ValueOf(session)
		case pt.Kind() == reflect.String && !seenString:
			seenString = true
			args[i] = reflect.ValueOf(repoRef)
		case pt.Kind() == reflect.Ptr:
			args[i] = reflect.New(pt.Elem())
		default:
			args[i] = reflect.Zero(pt)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	var out []reflect.Value
	if mt.IsVariadic() {
		out = m.CallSlice(args)
	} else {
		out = m.Call(args)
	}
	for _, v := range out {
		if v.Type() == ctlErrorType && !v.IsNil() {
			errVal, _ := v.Interface().(error)
			return errVal
		}
	}
	return nil
}

func TestCtlRepoController_UnknownRepo(t *testing.T) {
	e := ctlNewEnv(t)
	names := ctlRepoMethods(t, e)

	for _, name := range names {
		err := ctlInvoke(t, e.ctrl, name, e.session(), "root/unknown")
		require.Error(t, err, name)
		require.NotContains(t, err.Error(), "panic:", name)
	}
}

func TestCtlRepoController_AccessDenied(t *testing.T) {
	allow := ctlNewEnv(t)
	names := ctlRepoMethods(t, allow)

	for _, mode := range []ctlAuthMode{ctlDeny, ctlError} {
		c := *allow.ctrl
		c.authorizer = ctlAuthorizer{mode: mode}

		for _, name := range names {
			err := ctlInvoke(t, &c, name, allow.session(), "root/repo1")
			require.Error(t, err, name)
			require.NotContains(t, err.Error(), "panic:", name)
			switch {
			case ctlValidatesInputFirst[name]:
				continue
			case mode == ctlError:
				require.ErrorIs(t, err, errCtlAuthz, name)
			default:
				require.ErrorIs(t, err, apiauth.ErrForbidden, name)
			}
		}
	}
}

// ctlRepoMethods lists the methods that are driven generically. DeleteGitRepository takes the git UID instead
// of a repository reference and is tested separately.
func ctlRepoMethods(t *testing.T, e *ctlEnv) []string {
	t.Helper()
	var names []string
	for _, name := range ctlRepoRefMethods(e.ctrl) {
		if name != "DeleteGitRepository" {
			names = append(names, name)
		}
	}
	require.NotEmpty(t, names)
	return names
}

// ctlValidatesInputFirst lists methods that validate their (zero valued) input or look up a deleted repository
// before checking access, so they return a validation error instead of a forbidden error for the generic call.
var ctlValidatesInputFirst = map[string]bool{
	"Blame":                 true,
	"LinkedSync":            true,
	"MergeCommit":           true,
	"MergeQueueClear":       true,
	"MergeQueueListEntries": true,
	"Purge":                 true,
	"Rebase":                true,
	"Restore":               true,
	"Squash":                true,
}
