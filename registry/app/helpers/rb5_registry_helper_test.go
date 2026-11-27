//  Copyright 2023 Harness, Inc.
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
	"encoding/gob"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	localurlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	artifactapi "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	registrypostprocessingevents "github.com/harness/gitness/registry/app/events/asyncprocessing"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	gitnesstypes "github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errRb5Boom = errors.New("rb5 boom")

type rb5Rec struct {
	mu    sync.Mutex
	calls []string
	errs  map[string]error
}

func (r *rb5Rec) hit(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, name)
	return r.errs[name]
}

func (r *rb5Rec) has(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.calls {
		if c == name {
			return true
		}
	}
	return false
}

type rb5Artifacts struct {
	store.ArtifactRepository
	rec *rb5Rec
}

func (a *rb5Artifacts) GetByName(context.Context, int64, string, ...types.QueryOption) (*types.Artifact, error) {
	return &types.Artifact{}, a.rec.hit("GetByName")
}

func (a *rb5Artifacts) DeleteByVersionAndImageName(context.Context, string, string, int64) error {
	return a.rec.hit("DeleteByVersionAndImageName")
}

func (a *rb5Artifacts) DeleteByImageNameAndRegistryID(context.Context, int64, string) error {
	return a.rec.hit("DeleteByImageNameAndRegistryID")
}

type rb5Images struct {
	store.ImageRepository
	rec *rb5Rec
}

func (i *rb5Images) DeleteByImageNameIfNoLinkedArtifacts(context.Context, int64, string) error {
	return i.rec.hit("DeleteByImageNameIfNoLinkedArtifacts")
}

func (i *rb5Images) DeleteByImageNameAndRegID(context.Context, int64, string) error {
	return i.rec.hit("DeleteByImageNameAndRegID")
}

type rb5Files struct {
	filemanager.FileManager
	rec *rb5Rec
}

func (f *rb5Files) DeleteFile(context.Context, int64, string) error { return f.rec.hit("DeleteFile") }

type rb5Tx struct{}

func (rb5Tx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	return fn(ctx)
}

type rb5URLs struct{ localurlprovider.Provider }

func (rb5URLs) PackageURL(_ context.Context, regRef, pkgType string, _ ...string) string {
	return "https://pkg.example.com/" + regRef + "/" + pkgType
}

func (rb5URLs) RegistryURL(_ context.Context, params ...string) string {
	return "https://reg.example.com/" + params[0]
}

type rb5Tasks struct {
	store.TaskRepository
	rec *rb5Rec
}

func (t *rb5Tasks) UpsertTask(_ context.Context, task *types.Task) error {
	return t.rec.hit("UpsertTask:" + task.Key)
}

func (t *rb5Tasks) LockForUpdate(context.Context, *types.Task) (types.TaskStatus, error) {
	return types.TaskStatusPending, nil
}

func (t *rb5Tasks) UpdateStatus(context.Context, string, types.TaskStatus) error { return nil }

func (t *rb5Tasks) SetRunAgain(context.Context, string, bool) error { return nil }

type rb5Sources struct {
	store.TaskSourceRepository
	rec *rb5Rec
}

func (s *rb5Sources) InsertSource(_ context.Context, key string, _ types.SourceRef) error {
	return s.rec.hit("InsertSource:" + key)
}

type rb5TaskEvents struct{}

func (rb5TaskEvents) LogTaskEvent(context.Context, string, string, []byte) error { return nil }

type rb5Producer struct {
	mu   sync.Mutex
	sent []string
}

func (p *rb5Producer) Send(_ context.Context, streamID string, _ map[string]any) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sent = append(p.sent, streamID)
	return "1", nil
}

type rb5Collector struct{}

func (rb5Collector) RecordError(string, string)   {}
func (rb5Collector) RecordDiscard(string, string) {}

var rb5GobOnce sync.Once

func rb5EventSystem(t *testing.T) (*events.System, *rb5Producer) {
	t.Helper()
	rb5GobOnce.Do(func() { gob.Register(&registryevents.CommonArtifact{}) })
	producer := &rb5Producer{}
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return nil, errRb5Boom }, producer, rb5Collector{}, false)
	require.NoError(t, err)
	return system, producer
}

type rb5Env struct {
	helper   *registryHelper
	rec      *rb5Rec
	artifact *rb5Producer
	post     *rb5Producer
}

func rb5NewEnv(t *testing.T, errs map[string]error, withPostProcessing bool) *rb5Env {
	t.Helper()
	rec := &rb5Rec{errs: errs}
	artifactSystem, artifactProducer := rb5EventSystem(t)
	reporter, err := registryevents.NewReporter(artifactSystem)
	require.NoError(t, err)

	env := &rb5Env{rec: rec, artifact: artifactProducer}
	var post *registrypostprocessingevents.Reporter
	if withPostProcessing {
		postSystem, postProducer := rb5EventSystem(t)
		env.post = postProducer
		post, err = registrypostprocessingevents.NewReporter(
			rb5Tx{}, postSystem, &rb5Tasks{rec: rec}, &rb5Sources{rec: rec}, rb5TaskEvents{})
		require.NoError(t, err)
	}

	h := NewRegistryHelper(
		&rb5Artifacts{rec: rec}, &rb5Files{rec: rec}, &rb5Images{rec: rec}, reporter, post, rb5Tx{}, rb5URLs{},
		"Bearer",
	)
	impl, ok := h.(*registryHelper)
	require.True(t, ok)
	env.helper = impl
	return env
}

func rb5Ctx() context.Context {
	return request.WithAuthSession(context.Background(), &auth.Session{Principal: gitnesstypes.Principal{ID: 3}})
}

func TestRb5RegistryHelperDeleteVersion(t *testing.T) {
	ctx := context.Background()
	regInfo := &types.RegistryRequestBaseInfo{RegistryID: 5}
	img := &types.Image{ID: 9, Name: "img"}

	tests := []struct {
		name    string
		failing string
		wantMsg string
	}{
		{name: "success"},
		{name: "missing version", failing: "GetByName", wantMsg: "version doesn't exist for image img"},
		{name: "file delete", failing: "DeleteFile"},
		{name: "artifact delete", failing: "DeleteByVersionAndImageName", wantMsg: "failed to delete version"},
		{name: "image delete", failing: "DeleteByImageNameIfNoLinkedArtifacts", wantMsg: "failed to delete image"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := map[string]error{}
			if tt.failing != "" {
				errs[tt.failing] = errRb5Boom
			}
			env := rb5NewEnv(t, errs, false)
			err := env.helper.DeleteVersion(ctx, regInfo, img, "img", "1.0", "/path")
			if tt.failing == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, errRb5Boom)
			if tt.wantMsg != "" {
				require.ErrorContains(t, err, tt.wantMsg)
			}
		})
	}
}

func TestRb5RegistryHelperDeleteFilesAndImages(t *testing.T) {
	ctx := context.Background()
	regInfo := &types.RegistryRequestBaseInfo{RegistryID: 5}

	env := rb5NewEnv(t, nil, false)
	require.NoError(t, env.helper.DeleteFileNode(ctx, regInfo, "/file"))
	require.NoError(t, env.helper.DeleteGenericImage(ctx, regInfo, "img", "/file"))
	require.True(t, env.rec.has("DeleteByImageNameAndRegID"))

	for _, step := range []string{"DeleteFile", "DeleteByImageNameAndRegistryID", "DeleteByImageNameAndRegID"} {
		failing := rb5NewEnv(t, map[string]error{step: errRb5Boom}, false)
		require.ErrorIs(t, failing.helper.DeleteGenericImage(ctx, regInfo, "img", "/file"), errRb5Boom, step)
	}
	failing := rb5NewEnv(t, map[string]error{"DeleteFile": errRb5Boom}, false)
	require.ErrorIs(t, failing.helper.DeleteFileNode(ctx, regInfo, "/file"), errRb5Boom)
}

func TestRb5RegistryHelperEvents(t *testing.T) {
	ctx := rb5Ctx()

	env := rb5NewEnv(t, nil, true)
	env.helper.ReportDeleteVersionEvent(ctx, &registryevents.ArtifactDeletedPayload{
		RegistryID: 5,
		Artifact:   &registryevents.CommonArtifact{},
	})
	require.Len(t, env.artifact.sent, 1)

	env.helper.ReportBuildPackageIndexEvent(ctx, 5, "img")
	env.helper.ReportBuildPackageMetadataEvent(ctx, 5, "img")
	env.helper.ReportBuildRegistryIndexEvent(ctx, 5, []types.SourceRef{{Type: types.SourceTypeRegistry, ID: 5}})
	require.True(t, env.rec.has("UpsertTask:package_5_img"))
	require.True(t, env.rec.has("UpsertTask:package_5_img__metadata"))
	require.Len(t, env.post.sent, 3)

	// Without a post processing reporter the reindexing events are skipped.
	bare := rb5NewEnv(t, nil, false)
	bare.helper.ReportBuildPackageIndexEvent(ctx, 5, "img")
	bare.helper.ReportBuildPackageMetadataEvent(ctx, 5, "img")
	bare.helper.ReportBuildRegistryIndexEvent(ctx, 5, nil)
	require.False(t, bare.rec.has("UpsertTask:package_5_img"))
}

func TestRb5RegistryHelperURLs(t *testing.T) {
	ctx := context.Background()
	env := rb5NewEnv(t, nil, false)

	require.Equal(t, "Bearer", env.helper.GetAuthHeaderPrefix())
	require.Equal(t, "https://pkg.example.com/root/reg/npm", env.helper.GetPackageURL(ctx, "root", "reg", "npm"))
	require.Equal(t, "reg.example.com/root", env.helper.GetHostName(ctx, "root"))
}

func TestRb5RegistryHelperMetadata(t *testing.T) {
	env := rb5NewEnv(t, nil, false)
	modified := time.UnixMilli(1700000000123)
	reason := "bad"

	meta := env.helper.GetArtifactMetadata(types.ArtifactMetadata{
		RepoName: "reg", RegistryUUID: "ru", UUID: "u", Name: "img", Version: "1.0", Labels: []string{"l"},
		ModifiedAt: modified, PackageType: artifactapi.PackageTypeNPM, DownloadCount: 4, IsQuarantined: true,
		QuarantineReason: &reason,
	}, "npm install img")
	require.Equal(t, "reg", meta.RegistryIdentifier)
	require.Equal(t, "1.0", *meta.Version)
	require.Equal(t, "1700000000123", *meta.LastModified)
	require.Equal(t, "npm install img", *meta.PullCommand)
	require.Equal(t, int64(4), *meta.DownloadsCount)
	require.True(t, *meta.IsQuarantined)
	require.Equal(t, "bad", *meta.QuarantineReason)

	version := env.helper.GetArtifactVersionMetadata(types.NonOCIArtifactMetadata{
		UUID: "u", Name: "1.0", Size: "2048", FileCount: 3, ModifiedAt: modified, DownloadCount: 7,
	}, "pull", "NPM")
	require.Equal(t, artifactapi.PackageType("NPM"), version.PackageType)
	require.Equal(t, int64(3), *version.FileCount)
	require.Equal(t, "2.00KB", *version.Size)
	require.Equal(t, "1700000000123", *version.LastModified)

	file := env.helper.GetFileMetadata(types.FileNodeMetadata{
		Path: "/a/b", Size: 1024, Sha1: "s1", Sha256: "s256", Sha512: "s512", MD5: "m5", CreatedAt: 42,
	}, "b", "curl b")
	require.Equal(t, "b", file.Name)
	require.Equal(t, "/a/b", file.Path)
	require.Equal(t, "42", file.CreatedAt)
	require.Equal(t, "curl b", file.DownloadCommand)
	require.Equal(t, "1.00KB", file.Size)
	require.Equal(t, []string{"SHA-512: s512", "SHA-256: s256", "SHA-1: s1", "MD5: m5"}, file.Checksums)

	img := &types.Image{Name: "img"}
	art := &types.Artifact{Version: "1.0", CreatedAt: modified, UpdatedAt: modified}
	detail := env.helper.GetArtifactDetail(img, art, map[string]any{"size": float64(2048)}, 5)
	require.Equal(t, "img", *detail.Name)
	require.Equal(t, "1.0", detail.Version)
	require.Equal(t, "2.00KB", *detail.Size)
	require.Equal(t, int64(5), *detail.DownloadCount)

	// A missing size falls back to zero bytes.
	detail = env.helper.GetArtifactDetail(img, art, map[string]any{}, 0)
	require.Equal(t, "0.00B", *detail.Size)
}

func TestRb5SizeHelpers(t *testing.T) {
	require.Equal(t, "0.00B", GetSize(0))
	require.Equal(t, "1.00KB", GetSize(1024))
	require.Equal(t, "1.00KB", GetImageSize("1024"))
	require.Equal(t, "0.00B", GetImageSize("not-a-number"))
	require.Equal(t, "1000", GetTimeInMs(time.UnixMilli(1000)))
}

func rb5Command(label, value string) artifactapi.ClientSetupStepCommand {
	return artifactapi.ClientSetupStepCommand{Label: &label, Value: &value}
}

func rb5StepSection(t *testing.T, commands ...artifactapi.ClientSetupStepCommand) artifactapi.ClientSetupSection {
	t.Helper()
	steps := []artifactapi.ClientSetupStep{{Commands: &commands}}
	var section artifactapi.ClientSetupSection
	require.NoError(t, section.FromClientSetupStepConfig(artifactapi.ClientSetupStepConfig{Steps: &steps}))
	return section
}

func rb5SectionCommands(t *testing.T, section artifactapi.ClientSetupSection) []artifactapi.ClientSetupStepCommand {
	t.Helper()
	cfg, err := section.AsClientSetupStepConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg.Steps)
	require.NotNil(t, (*cfg.Steps)[0].Commands)
	return *(*cfg.Steps)[0].Commands
}

func TestRb5ReplacePlaceholders(t *testing.T) {
	ctx := context.Background()
	env := rb5NewEnv(t, nil, false)
	image := artifactapi.ArtifactParam("my-image")
	version := artifactapi.VersionParam("1.2.3")

	const template = "<AUTH_HEADER_PREFIX> <USERNAME> <GROUP_ID> <REGISTRY_URL> <UPLOAD_URL> <HOSTNAME> " +
		"<LOGIN_HOSTNAME> <REGISTRY_NAME> <IMAGE_NAME> <ARTIFACT_ID> <ARTIFACT_NAME> <TAG> <VERSION> <DIGEST>"
	const want = "Bearer alice grp https://reg.example.com https://up.example.com host.example.com:8443 " +
		"host.example.com:8443 reg my-image my-image my-image 1.2.3 1.2.3 1.2.3"

	t.Run("plain section", func(t *testing.T) {
		sections := []artifactapi.ClientSetupSection{
			rb5StepSection(t, rb5Command("login <USERNAME> at <REGISTRY_URL>", template)),
		}
		env.helper.ReplacePlaceholders(ctx, &sections, "alice", "root/reg", &image, &version,
			"https://reg.example.com", "grp", "https://up.example.com", "host.example.com:8443")

		commands := rb5SectionCommands(t, sections[0])
		require.Equal(t, want, *commands[0].Value)
		require.Equal(t, "login alice at https://reg.example.com", *commands[0].Label)
	})

	t.Run("tabbed sections are replaced recursively", func(t *testing.T) {
		inner := []artifactapi.ClientSetupSection{rb5StepSection(t, rb5Command("", "docker login <LOGIN_HOSTNAME>"))}
		tabs := []artifactapi.TabSetupStep{{Sections: &inner}}
		var outer artifactapi.ClientSetupSection
		require.NoError(t, outer.FromTabSetupStepConfig(artifactapi.TabSetupStepConfig{Tabs: &tabs}))
		sections := []artifactapi.ClientSetupSection{outer}

		env.helper.ReplacePlaceholders(ctx, &sections, "", "root/reg", nil, nil, "", "", "", "host.example.com")

		tab, err := sections[0].AsTabSetupStepConfig()
		require.NoError(t, err)
		commands := rb5SectionCommands(t, (*(*tab.Tabs)[0].Sections)[0])
		require.Equal(t, "docker login host.example.com", *commands[0].Value)
	})

	t.Run("empty values leave placeholders untouched", func(t *testing.T) {
		bare := rb5NewEnv(t, nil, false)
		bare.helper.SetupDetailsAuthHeaderPrefix = ""
		sections := []artifactapi.ClientSetupSection{rb5StepSection(t, rb5Command("", "<USERNAME> <TAG>"))}
		bare.helper.ReplacePlaceholders(ctx, &sections, "", "reg", nil, nil, "", "", "", "")
		require.Equal(t, "<USERNAME> <TAG>", *rb5SectionCommands(t, sections[0])[0].Value)
	})

	t.Run("sections without steps or commands are ignored", func(t *testing.T) {
		var empty artifactapi.ClientSetupSection
		require.NoError(t, empty.FromClientSetupStepConfig(artifactapi.ClientSetupStepConfig{}))
		noCommands := []artifactapi.ClientSetupStep{{}}
		var withoutCommands artifactapi.ClientSetupSection
		require.NoError(t, withoutCommands.FromClientSetupStepConfig(
			artifactapi.ClientSetupStepConfig{Steps: &noCommands}))
		sections := []artifactapi.ClientSetupSection{empty, withoutCommands}
		require.NotPanics(t, func() {
			env.helper.ReplacePlaceholders(ctx, &sections, "alice", "reg", &image, &version, "u", "g", "up", "h")
		})
	})
}
