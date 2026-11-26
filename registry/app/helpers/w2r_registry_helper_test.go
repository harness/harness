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

package helpers

import (
	"context"
	"testing"
	"time"

	localurlprovider "github.com/harness/gitness/app/url"
	artifactapi "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

type w2rURLProvider struct {
	localurlprovider.Provider
}

func (w2rURLProvider) PackageURL(_ context.Context, ref, pkgType string, _ ...string) string {
	return "https://pkg.example.com/" + ref + "/" + pkgType
}

func (w2rURLProvider) RegistryURL(_ context.Context, parts ...string) string {
	return "https://registry.example.com/" + parts[0]
}

func w2rPtr(s string) *string { return &s }

func TestW2rRegistryHelperSimpleAccessors(t *testing.T) {
	ctx := context.Background()
	h := NewRegistryHelper(nil, nil, nil, nil, nil, nil, w2rURLProvider{}, "Bearer")

	require.Equal(t, "Bearer", h.GetAuthHeaderPrefix())
	require.Equal(t, "https://pkg.example.com/root/reg/generic", h.GetPackageURL(ctx, "root", "reg", "generic"))
	require.Equal(t, "registry.example.com/root", h.GetHostName(ctx, "root"))
}

func TestW2rSizeAndTimeFormatting(t *testing.T) {
	require.Equal(t, "1700000000000", GetTimeInMs(time.UnixMilli(1700000000000)))
	require.Equal(t, GetSize(2048), GetImageSize("2048"))
	require.Equal(t, GetSize(0), GetImageSize("not-a-number"))
	require.NotEqual(t, GetSize(1), GetSize(1024*1024))

	sums := GetCheckSums(types.FileNodeMetadata{Sha512: "5", Sha256: "2", Sha1: "1", MD5: "m"})
	require.Equal(t, []string{"SHA-512: 5", "SHA-256: 2", "SHA-1: 1", "MD5: m"}, sums)
}

func w2rSetupSection(t *testing.T, commands ...artifactapi.ClientSetupStepCommand) artifactapi.ClientSetupSection {
	t.Helper()
	var section artifactapi.ClientSetupSection
	step := artifactapi.ClientSetupStep{Commands: &commands}
	steps := []artifactapi.ClientSetupStep{step}
	require.NoError(t, section.FromClientSetupStepConfig(artifactapi.ClientSetupStepConfig{Steps: &steps}))
	return section
}

func TestW2rReplacePlaceholders(t *testing.T) {
	ctx := context.Background()
	h := &registryHelper{SetupDetailsAuthHeaderPrefix: "Bearer"}
	image := artifactapi.ArtifactParam("my-image")
	version := artifactapi.VersionParam("1.2.3")

	sections := []artifactapi.ClientSetupSection{w2rSetupSection(t, artifactapi.ClientSetupStepCommand{
		Label: w2rPtr("login as <USERNAME> to <REGISTRY_URL>"),
		Value: w2rPtr("curl -H '<AUTH_HEADER_PREFIX> t' -u <USERNAME> <REGISTRY_URL>/<GROUP_ID>/<REGISTRY_NAME>/" +
			"<IMAGE_NAME>/<ARTIFACT_ID>/<ARTIFACT_NAME>/<TAG>/<VERSION>/<DIGEST> <UPLOAD_URL> <HOSTNAME> <LOGIN_HOSTNAME>"),
	})}

	h.ReplacePlaceholders(ctx, &sections, "alice", "root/reg", &image, &version,
		"https://r.example.com", "grp", "https://upload", "r.example.com")

	cfg, err := sections[0].AsClientSetupStepConfig()
	require.NoError(t, err)
	cmd := (*(*cfg.Steps)[0].Commands)[0]
	require.Equal(t, "login as alice to https://r.example.com", *cmd.Label)
	require.Equal(t, "curl -H 'Bearer t' -u alice https://r.example.com/grp/reg/my-image/my-image/my-image/"+
		"1.2.3/1.2.3/1.2.3 https://upload r.example.com r.example.com", *cmd.Value)
}

func TestW2rReplacePlaceholdersWithoutValuesAndTabs(t *testing.T) {
	ctx := context.Background()
	h := &registryHelper{}

	// Empty replacement values keep the placeholders untouched.
	sections := []artifactapi.ClientSetupSection{w2rSetupSection(t,
		artifactapi.ClientSetupStepCommand{Value: w2rPtr("<USERNAME> <TAG> <REGISTRY_URL>")})}
	h.ReplacePlaceholders(ctx, &sections, "", "root/reg", nil, nil, "", "", "", "")
	cfg, err := sections[0].AsClientSetupStepConfig()
	require.NoError(t, err)
	require.Equal(t, "<USERNAME> <TAG> <REGISTRY_URL>", *(*(*cfg.Steps)[0].Commands)[0].Value)

	// Sections nested in tabs are processed too.
	inner := []artifactapi.ClientSetupSection{w2rSetupSection(t,
		artifactapi.ClientSetupStepCommand{Value: w2rPtr("hello <USERNAME>")})}
	tabs := []artifactapi.TabSetupStep{{Header: w2rPtr("tab"), Sections: &inner}}
	var tabSection artifactapi.ClientSetupSection
	require.NoError(t, tabSection.FromTabSetupStepConfig(artifactapi.TabSetupStepConfig{Tabs: &tabs}))
	outer := []artifactapi.ClientSetupSection{tabSection}

	h.ReplacePlaceholders(ctx, &outer, "bob", "root/reg", nil, nil, "", "", "", "")
	tabCfg, err := outer[0].AsTabSetupStepConfig()
	require.NoError(t, err)
	innerCfg, err := (*(*tabCfg.Tabs)[0].Sections)[0].AsClientSetupStepConfig()
	require.NoError(t, err)
	require.Equal(t, "hello bob", *(*(*innerCfg.Steps)[0].Commands)[0].Value)
}

func TestW2rReplacePlaceholdersInSectionIgnoresEmptySteps(t *testing.T) {
	h := &registryHelper{}
	var empty artifactapi.ClientSetupSection
	require.NoError(t, empty.FromClientSetupStepConfig(artifactapi.ClientSetupStepConfig{}))
	h.ReplacePlaceholdersInSection(context.Background(), &empty, "u", "root/reg", nil, nil, "", "", "", "")

	noCommands := artifactapi.ClientSetupStep{}
	steps := []artifactapi.ClientSetupStep{noCommands}
	var section artifactapi.ClientSetupSection
	require.NoError(t, section.FromClientSetupStepConfig(artifactapi.ClientSetupStepConfig{Steps: &steps}))
	h.ReplacePlaceholdersInSection(context.Background(), &section, "u", "root/reg", nil, nil, "", "", "", "")
}
