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
	"strings"
	"testing"

	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func gc5Ptr[T any](v T) *T { return &v }

func gc5Resource(cpu, memory, disk, region string) InfraProviderResource {
	return InfraProviderResource{CPU: &cpu, Memory: &memory, Disk: &disk, Region: region}
}

func TestGc5ValidateInfraProviderResource(t *testing.T) {
	require.NoError(t, validateInfraProviderResource(gc5Resource("2", "4 GB", "10gb", "r")))

	tests := []struct {
		name string
		res  InfraProviderResource
		msg  string
	}{
		{"missing cpu", InfraProviderResource{Memory: gc5Ptr("1gb"), Disk: gc5Ptr("1gb")}, "cpu is required"},
		{"bad cpu", gc5Resource("two", "1gb", "1gb", ""), "invalid syntax"},
		{"negative cpu", gc5Resource("-1", "1gb", "1gb", ""), "cpu must be positive"},
		{"missing memory", InfraProviderResource{CPU: gc5Ptr("1"), Disk: gc5Ptr("1gb")}, "bytes is required"},
		{"bad memory", gc5Resource("1", "lots", "1gb", ""), "invalid size"},
		{"negative memory", gc5Resource("1", "-1gb", "1gb", ""), "invalid size"},
		{"bad disk", gc5Resource("1", "1gb", "huge", ""), "invalid size"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.ErrorContains(t, validateInfraProviderResource(tt.res), tt.msg)
		})
	}

	require.Equal(t, "ab", withoutSpace(" a b "))
	require.Equal(t, int64(5), (&InfraProviderResource{ID: 5}).Identifier())
}

func TestGc5CompareInfraProviderResource(t *testing.T) {
	base := gc5Resource("2", "4gb", "10gb", "a")

	tests := []struct {
		name string
		a, b InfraProviderResource
		sign int
	}{
		{"equal", base, gc5Resource("2", "4 gb", "10gb", "a"), 0},
		{"cpu smaller", gc5Resource("1", "4gb", "10gb", "a"), base, -1},
		{"cpu larger", gc5Resource("4", "4gb", "10gb", "a"), base, 1},
		{"memory smaller", gc5Resource("2", "2gb", "10gb", "a"), base, -1},
		{"memory larger", gc5Resource("2", "8gb", "10gb", "a"), base, 1},
		{"disk smaller", gc5Resource("2", "4gb", "5gb", "a"), base, -1},
		{"disk larger", gc5Resource("2", "4gb", "20gb", "a"), base, 1},
		{"region smaller", base, gc5Resource("2", "4gb", "10gb", "b"), -1},
		{"region larger", gc5Resource("2", "4gb", "10gb", "c"), base, 1},
		{"invalid first is not comparable", InfraProviderResource{}, base, 0},
		{"invalid second is not comparable", base, InfraProviderResource{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareInfraProviderResource(tt.a, tt.b)
			switch {
			case tt.sign < 0:
				require.Negative(t, got)
			case tt.sign > 0:
				require.Positive(t, got)
			default:
				require.Zero(t, got)
			}
		})
	}
}

func gc5Feature(sourceWithoutTag, tag, canonical string, overridden map[string]any) *ResolvedFeature {
	return &ResolvedFeature{
		OverriddenOptions: overridden,
		DownloadedFeature: &DownloadedFeature{
			SourceWithoutTag: sourceWithoutTag, Tag: tag, CanonicalName: canonical, Source: sourceWithoutTag + ":" + tag,
		},
	}
}

func TestGc5CompareResolvedFeature(t *testing.T) {
	tests := []struct {
		name string
		a, b *ResolvedFeature
		sign int
	}{
		{"source first", gc5Feature("a", "1.0.0", "x", nil), gc5Feature("b", "1.0.0", "x", nil), -1},
		{"older tag first", gc5Feature("a", "1.0.0", "x", nil), gc5Feature("a", "2.0.0", "x", nil), -1},
		{"latest is newest", gc5Feature("a", FeatureDefaultTag, "x", nil), gc5Feature("a", "2.0.0", "x", nil), 1},
		{"latest is newest reversed", gc5Feature("a", "2.0.0", "x", nil), gc5Feature("a", FeatureDefaultTag, "x", nil), -1},
		{"both latest are equal on tag", gc5Feature("a", FeatureDefaultTag, "x", nil),
			gc5Feature("a", FeatureDefaultTag, "y", nil), -1},
		{"more options sort later", gc5Feature("a", "1.0.0", "x", map[string]any{"k": "v"}),
			gc5Feature("a", "1.0.0", "x", nil), 1},
		{"option keys", gc5Feature("a", "1.0.0", "x", map[string]any{"a": "v"}),
			gc5Feature("a", "1.0.0", "x", map[string]any{"b": "v"}), -1},
		{"option values", gc5Feature("a", "1.0.0", "x", map[string]any{"k": "a"}),
			gc5Feature("a", "1.0.0", "x", map[string]any{"k": true}), -1},
		{"canonical name", gc5Feature("a", "1.0.0", "y", nil), gc5Feature("a", "1.0.0", "x", nil), 1},
		{"equal", gc5Feature("a", "1.0.0", "x", map[string]any{"k": "v"}),
			gc5Feature("a", "1.0.0", "x", map[string]any{"k": "v"}), 0},
		{"unparseable tags are equal on tag", gc5Feature("a", "weird", "x", nil), gc5Feature("a", "other", "x", nil), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareResolvedFeature(tt.a, tt.b)
			switch {
			case tt.sign < 0:
				require.Negative(t, got)
			case tt.sign > 0:
				require.Positive(t, got)
			default:
				require.Zero(t, got)
			}
		})
	}
}

func TestGc5CompareTags(t *testing.T) {
	cmp, err := compareTags("1.2.3", "1.2.4")
	require.NoError(t, err)
	require.Negative(t, cmp)

	_, err = compareTags("nope", "1.0.0")
	require.Error(t, err)
	_, err = compareTags("1.0.0", "nope")
	require.Error(t, err)
}

func TestGc5SortedOptions(t *testing.T) {
	keys, values := getSortedOptions(map[string]any{"b": true, "a": "x", "c": 5})

	require.Equal(t, []string{"a", "b", "c"}, keys)
	require.Equal(t, []string{"x", "true"}, values)
}

func TestGc5ResolvedFeaturePrint(t *testing.T) {
	f := gc5Feature("ghcr.io/org/feat", "1", "x", nil)
	f.ResolvedOptions = map[string]string{"version": "1.2"}

	out := f.Print()
	require.True(t, strings.HasPrefix(out, "ghcr.io/org/feat:1 "))
	require.Contains(t, out, "version=1.2")
}

func TestGc5OptionDefinitionValidateValue(t *testing.T) {
	boolDef := &OptionDefinition{Type: enum.FeatureOptionValueTypeBoolean}
	stringDef := &OptionDefinition{Type: enum.FeatureOptionValueTypeString, Enum: []string{"a", "b"}}
	freeStringDef := &OptionDefinition{Type: enum.FeatureOptionValueTypeString}

	tests := []struct {
		name    string
		def     *OptionDefinition
		value   any
		want    string
		wantErr string
	}{
		{"bool value", boolDef, true, "true", ""},
		{"bool string", boolDef, "FALSE", "false", ""},
		{"bool invalid string", boolDef, "maybe", "", "expects boolean"},
		{"bool wrong type", boolDef, 3, "", "expects boolean"},
		{"string allowed", stringDef, "a", "a", ""},
		{"string not allowed", stringDef, "z", "", "not allowed"},
		{"string wrong type", stringDef, true, "", "expects string"},
		{"string free", freeStringDef, "anything", "anything", ""},
		{"unsupported type", &OptionDefinition{Type: "number"}, 1, "", "unsupported option type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.def.ValidateValue(tt.value, "opt", "src")
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGc5CriteriaToKey(t *testing.T) {
	key, err := GitspaceSettingsCriteria{}.ToKey()
	require.NoError(t, err)
	require.Empty(t, key)

	key, err = GitspaceSettingsCriteria{
		"b": map[string]any{"y": 2, "x": "one"},
		"a": true,
	}.ToKey()
	require.NoError(t, err)
	require.Equal(t, CriteriaKey("a/true/b.x/one/b.y/2"), key)
}

func TestGc5FlattenCriteriaDepthLimit(t *testing.T) {
	nested := map[string]any{"leaf": "v"}
	for range 12 {
		nested = map[string]any{"n": nested}
	}

	out := map[string]string{}
	flattenCriteria("", nested, out)

	require.Empty(t, out, "values nested deeper than the limit are dropped")

	shallow := map[string]string{}
	flattenCriteria("pre", map[string]any{"a": map[string]any{"b": 1}}, shallow)
	require.Equal(t, map[string]string{"pre.a.b": "1"}, shallow)
}

func TestGc5AccessList(t *testing.T) {
	var nilList *AccessList[string]
	require.True(t, nilList.IsAllowed("x"))
	require.False(t, nilList.Remove("x"))

	empty := &AccessList[string]{Mode: ListModeAllow}
	require.True(t, empty.IsAllowed("x"))
	require.False(t, empty.Remove("x"))

	allow := &AccessList[string]{Mode: ListModeAllow, List: []string{"a", "b"}}
	require.True(t, allow.IsAllowed("a"))
	require.False(t, allow.IsAllowed("c"))

	deny := &AccessList[string]{Mode: ListModeDeny, List: []string{"a", "b"}}
	require.False(t, deny.IsAllowed("a"))
	require.True(t, deny.IsAllowed("c"))

	require.True(t, deny.Remove("a"))
	require.Equal(t, []string{"b"}, deny.List)
	require.False(t, deny.Remove("zzz"))
}

func TestGc5DecodeAndValidateSettings(t *testing.T) {
	data := map[string]any{
		"access_list":                map[string]any{"mode": "allow", "list": []any{"a"}},
		"auto_stopping_time_in_mins": 30,
	}

	decoded, err := DecodeSettings[InfraProviderSettings](data)
	require.NoError(t, err)
	require.Equal(t, 30, *decoded.AutoStoppingTimeInMins)
	require.Equal(t, ListModeAllow, decoded.AccessList.Mode)

	_, err = DecodeSettings[InfraProviderSettings](map[string]any{"auto_stopping_time_in_mins": "soon"})
	require.Error(t, err)
	_, err = DecodeSettings[InfraProviderSettings](map[string]any{"bad": make(chan int)})
	require.Error(t, err)

	got, err := ValidateAndDecodeSettings(enum.SettingsTypeInfraProvider, data)
	require.NoError(t, err)
	require.IsType(t, &InfraProviderSettings{}, got)

	got, err = ValidateAndDecodeSettings(enum.SettingsTypeGitspaceConfig, map[string]any{
		"ide": map[string]any{"disable_ssh": true},
	})
	require.NoError(t, err)
	gitspaceCfg, ok := got.(*GitspaceConfigSettings)
	require.True(t, ok)
	require.True(t, gitspaceCfg.IDEs.DisableSSH)

	_, err = ValidateAndDecodeSettings(enum.GitspaceSettingsType("unknown"), data)
	require.ErrorContains(t, err, "no schema registered")
	_, err = ValidateAndDecodeSettings(enum.SettingsTypeInfraProvider, map[string]any{"bad": make(chan int)})
	require.Error(t, err)
	_, err = ValidateAndDecodeSettings(enum.SettingsTypeInfraProvider, map[string]any{"auto_stopping_time_in_mins": "x"})
	require.ErrorContains(t, err, "schema mismatch")
}

func TestGc5GitspaceInstanceGetGitspaceState(t *testing.T) {
	tests := []struct {
		in   enum.GitspaceInstanceStateType
		want enum.GitspaceStateType
	}{
		{enum.GitspaceInstanceStateRunning, enum.GitspaceStateRunning},
		{enum.GitspaceInstanceStateStopped, enum.GitspaceStateStopped},
		{enum.GitspaceInstanceStateUninitialized, enum.GitspaceStateUninitialized},
		{enum.GitspaceInstanceStateDeleted, enum.GitspaceStateUninitialized},
		{enum.GitspaceInstanceStateCleaned, enum.GitspaceStateUninitialized},
		{"", enum.GitspaceStateUninitialized},
		{enum.GitspaceInstanceStateError, enum.GitspaceStateError},
		{enum.GitspaceInstanceStateUnknown, enum.GitspaceStateError},
		{enum.GitspaceInstanceStateStarting, enum.GitspaceStateStarting},
		{enum.GitspaceInstanceStateStopping, enum.GitspaceStateStopping},
		{enum.GitSpaceInstanceStateCleaning, enum.GitSpaceStateCleaning},
		{enum.GitSpaceInstanceStateResetting, enum.GitSpaceStateCleaning},
	}
	for _, tt := range tests {
		t.Run(string(tt.in), func(t *testing.T) {
			got, err := (&GitspaceInstance{State: tt.in}).GetGitspaceState()
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}

	got, err := (&GitspaceInstance{State: enum.GitspaceInstanceStatePendingCleanup}).GetGitspaceState()
	require.Error(t, err)
	require.Equal(t, enum.GitspaceStateError, got)

	var nilInstance *GitspaceInstance
	_, err = nilInstance.GetGitspaceState()
	require.Error(t, err)
}

func TestGc5RunArgValueString(t *testing.T) {
	require.Equal(t, "a, b", (&RunArgValue{Values: []string{"a", "b"}}).String())
	require.Empty(t, (&RunArgValue{}).String())
}
