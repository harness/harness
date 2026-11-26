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

package huggingface

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/metadata"
	huggingfacemetadata "github.com/harness/gitness/registry/app/metadata/huggingface"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	hftype "github.com/harness/gitness/registry/app/pkg/types/huggingface"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

type rpfFakeDriver struct {
	storagedriver.StorageDriver
	content []byte
}

func (d *rpfFakeDriver) Reader(_ context.Context, _ string, _ int64) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(d.content)), nil
}

func rpfFileReader(ctx context.Context, t *testing.T, content []byte) *storage.FileReader {
	t.Helper()
	fr, err := storage.NewFileReader(ctx, &rpfFakeDriver{content: content}, "p", int64(len(content)))
	require.NoError(t, err)
	return fr
}

type rpfURLProvider struct {
	urlprovider.Provider
}

func (rpfURLProvider) PackageURL(_ context.Context, regRef string, pkgType string, _ ...string) string {
	return "https://harness.test/pkg/" + regRef + "/" + pkgType
}

type rpfImageRepo struct {
	store.ImageRepository
	image *types.Image
	err   error
}

func (r *rpfImageRepo) GetByNameAndType(
	context.Context, int64, string, *artifact.ArtifactType, ...types.QueryOption,
) (*types.Image, error) {
	return r.image, r.err
}

type rpfArtifactRepo struct {
	store.ArtifactRepository
	artifact *types.Artifact
	err      error
	versErr  error
}

func (r *rpfArtifactRepo) GetByName(context.Context, int64, string, ...types.QueryOption) (*types.Artifact, error) {
	return r.artifact, r.err
}

func (r *rpfArtifactRepo) GetByRegistryImageAndVersion(
	context.Context, int64, string, string,
) (*types.Artifact, error) {
	return r.artifact, r.versErr
}

type rpfFileManager struct {
	filemanager.FileManager
	filePath    string
	filePathErr error
	uploadErr   error
	uploaded    []byte
	headDigest  bool
	headErr     error
	headFileErr error
	byPathErr   error
	readme      string
	readmeErr   error
	t           *testing.T
}

func (f *rpfFileManager) GetFilePath(context.Context, string, int64, int64) (string, error) {
	return f.filePath, f.filePathErr
}

func (f *rpfFileManager) UploadFileNoDBUpdate(
	_ context.Context, _ string, _ multipart.File, r io.Reader, _ int64, _ int64,
) (types.FileInfo, error) {
	f.uploaded, _ = io.ReadAll(r)
	return types.FileInfo{Filename: "blob"}, f.uploadErr
}

func (f *rpfFileManager) HeadByDigest(
	context.Context, string, types.FileInfo, int64, int64,
) (bool, int64, error) {
	return f.headDigest, 0, f.headErr
}

func (f *rpfFileManager) HeadFile(context.Context, string, int64) (string, int64, error) {
	if f.headFileErr != nil {
		return "", 0, f.headFileErr
	}
	return "sha-head", 99, nil
}

func (f *rpfFileManager) DownloadFileByPath(
	ctx context.Context, _ string, _ int64, _ string, _ string, _ bool,
) (*storage.FileReader, int64, string, error) {
	if f.byPathErr != nil {
		return nil, 0, "", f.byPathErr
	}
	return rpfFileReader(ctx, f.t, []byte("model")), 5, "https://redirect.test/model", nil
}

func (f *rpfFileManager) DownloadFileByDigest(
	ctx context.Context, _ string, _ types.FileInfo, _ int64, _ int64,
) (*storage.FileReader, error) {
	if f.readmeErr != nil {
		return nil, f.readmeErr
	}
	return rpfFileReader(ctx, f.t, []byte(f.readme)), nil
}

type rpfLocalBase struct {
	base.LocalBase
	err        error
	gotFiles   []types.FileInfo
	gotPrefix  string
	gotVersion string
	gotMeta    metadata.Metadata
}

func (b *rpfLocalBase) MoveMultipleTempFilesAndCreateArtifact(
	_ context.Context, _ *pkg.ArtifactInfo, prefix string, md metadata.Metadata, files *[]types.FileInfo, version string,
) error {
	b.gotPrefix, b.gotVersion, b.gotMeta = prefix, version, md
	b.gotFiles = *files
	return b.err
}

func rpfHFInfo() hftype.ArtifactInfo {
	return hftype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 1, RootIdentifier: "root", RootParentID: 2},
			RegIdentifier: "reg",
			RegistryID:    3,
		},
		Repo:     "org/model",
		Revision: "main",
		RepoType: artifact.ArtifactType("model"),
		SHA256:   "sha-oid",
	}
}

func rpfNewLocal(
	lb *rpfLocalBase, fm *rpfFileManager, images *rpfImageRepo, artifacts *rpfArtifactRepo,
) *localRegistry {
	reg := NewLocalRegistry(lb, fm, nil, nil, nil, images, artifacts, rpfURLProvider{})
	local, ok := reg.(*localRegistry)
	if !ok {
		return nil
	}
	return local
}

func TestRpfLocalStatic(t *testing.T) {
	l := rpfNewLocal(nil, nil, nil, nil)
	require.Equal(t, artifact.RegistryTypeVIRTUAL, l.GetArtifactType())
	require.Equal(t, []artifact.PackageType{artifact.PackageTypeHUGGINGFACE}, l.GetPackageTypes())
	require.Equal(t, "x", *stringPtr("x"))
	require.True(t, isSlice([]any{}))
	require.False(t, isSlice("x"))
	require.True(t, isString("x"))
	require.False(t, isString(1))
}

func TestRpfValidateYaml(t *testing.T) {
	ctx := context.Background()
	l := rpfNewLocal(nil, nil, nil, nil)
	validate := func(body string) (int, *hftype.ValidateYamlResponse, error) {
		headers, resp, err := l.ValidateYaml(ctx, rpfHFInfo(), io.NopCloser(strings.NewReader(body)))
		return headers.Code, resp, err
	}
	front := func(yaml string) string {
		raw, err := json.Marshal(map[string]string{"repoType": "model", "content": "---\n" + yaml + "\n---\n# Title"})
		require.NoError(t, err)
		return string(raw)
	}

	code, resp, err := validate("{not json")
	require.Error(t, err)
	require.Nil(t, resp)
	require.Equal(t, http.StatusBadRequest, code)

	for _, body := range []string{`{"content":"x"}`, `{"repoType":"space","content":"x"}`} {
		code, _, err = validate(body)
		require.Error(t, err, body)
		require.Equal(t, http.StatusBadRequest, code)
	}

	// Content without front matter, or no content at all, is valid.
	for _, body := range []string{`{"repoType":"model"}`, `{"repoType":"dataset","content":"# no front matter"}`} {
		_, resp, err = validate(body)
		require.NoError(t, err)
		require.Empty(t, *resp.Errors)
		require.Empty(t, *resp.Warnings)
	}

	code, resp, err = validate(front("pipeline_tag: text-generation\ntags:\n  - a\nwidget:\n  - text: hi"))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, code)
	require.Empty(t, *resp.Warnings)

	code, resp, err = validate(front("key: [unclosed"))
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, code)
	require.Contains(t, *(*resp.Errors)[0].Message, "Invalid YAML")

	tests := map[string]string{
		"pipeline_tag: [a]": `"pipeline_tag" must be a string`,
		"tags: nope":        `"tags" must be an array`,
		"widget: nope":      `"widget" must be an array`,
	}
	for yaml, want := range tests {
		code, resp, err = validate(front(yaml))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, code)
		require.Equal(t, want, *(*resp.Warnings)[0].Message)
	}
}

func TestRpfPreUpload(t *testing.T) {
	ctx := context.Background()
	l := rpfNewLocal(nil, nil, nil, nil)

	headers, resp, err := l.PreUpload(ctx, rpfHFInfo(), io.NopCloser(strings.NewReader(
		`{"files":[{"path":"a.bin","size":1},{"path":"b.bin","size":2}]}`)))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Len(t, *resp.Files, 2)
	require.Equal(t, hftype.LFSUpload, (*resp.Files)[0].UploadMode)
	require.Equal(t, "b.bin", (*resp.Files)[1].Path)

	for _, body := range []string{"{nope", `{}`, `{"files":[]}`} {
		headers, resp, err = l.PreUpload(ctx, rpfHFInfo(), io.NopCloser(strings.NewReader(body)))
		require.Error(t, err, body)
		require.Nil(t, resp)
		require.Equal(t, http.StatusBadRequest, headers.Code)
	}
}

func TestRpfRevisionInfo(t *testing.T) {
	ctx := context.Background()
	stored, err := json.Marshal(huggingfacemetadata.HuggingFaceMetadata{
		Metadata: huggingfacemetadata.Metadata{ID: "org/model"},
	})
	require.NoError(t, err)
	images := &rpfImageRepo{image: &types.Image{ID: 5}}

	t.Run("xet expansion needs no lookup", func(t *testing.T) {
		l := rpfNewLocal(nil, nil, &rpfImageRepo{err: errors.New("must not be called")}, nil)
		headers, resp, err := l.RevisionInfo(ctx, rpfHFInfo(), map[string][]string{"expand": {"xetEnabled"}})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		require.Equal(t, "org/model", resp.ID)
		require.False(t, resp.XetEnabled)
	})

	t.Run("stored metadata", func(t *testing.T) {
		l := rpfNewLocal(nil, nil, images, &rpfArtifactRepo{artifact: &types.Artifact{Metadata: stored}})
		headers, resp, err := l.RevisionInfo(ctx, rpfHFInfo(), nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		require.Equal(t, "main", resp.SHA)
	})

	t.Run("failures", func(t *testing.T) {
		l := rpfNewLocal(nil, nil, &rpfImageRepo{err: errors.New("x")}, nil)
		_, _, err := l.RevisionInfo(ctx, rpfHFInfo(), nil)
		require.Error(t, err)

		l = rpfNewLocal(nil, nil, images, &rpfArtifactRepo{err: errors.New("x")})
		_, _, err = l.RevisionInfo(ctx, rpfHFInfo(), nil)
		require.Error(t, err)

		l = rpfNewLocal(nil, nil, images, &rpfArtifactRepo{artifact: &types.Artifact{Metadata: []byte("{")}})
		_, _, err = l.RevisionInfo(ctx, rpfHFInfo(), nil)
		require.Error(t, err)

		l = rpfNewLocal(nil, nil, images, &rpfArtifactRepo{artifact: &types.Artifact{Metadata: []byte(`{}`)}})
		headers, _, err := l.RevisionInfo(ctx, rpfHFInfo(), nil)
		require.Error(t, err)
		require.Equal(t, http.StatusNotFound, headers.Code)
	})
}

func TestRpfLfsInfo(t *testing.T) {
	ctx := context.Background()
	body := func(op string) io.ReadCloser {
		return io.NopCloser(strings.NewReader(`{"operation":"` + op + `","objects":[{"oid":"oid-1","size":10}]}`))
	}

	t.Run("upload of a new object returns upload and verify actions", func(t *testing.T) {
		l := rpfNewLocal(nil, &rpfFileManager{}, nil, nil)
		headers, resp, err := l.LfsInfo(ctx, rpfHFInfo(), body("upload"), "Bearer secret")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		require.Len(t, *resp.Objects, 1)
		actions := *(*resp.Objects)[0].Actions
		require.Contains(t, actions["upload"].Href, "/api/models/org/model/main/multipart/upload/oid-1?sig=secret")
		require.Contains(t, actions["verify"].Href, "multipart/verify/oid-1")
		require.Equal(t, "oid-1", actions["upload"].Header["X-Checksum-Sha256"])
	})

	t.Run("upload of an existing object is skipped", func(t *testing.T) {
		l := rpfNewLocal(nil, &rpfFileManager{filePath: "/existing"}, nil, nil)
		_, resp, err := l.LfsInfo(ctx, rpfHFInfo(), body("upload"), "t")
		require.NoError(t, err)
		require.Empty(t, *resp.Objects)
	})

	t.Run("download", func(t *testing.T) {
		l := rpfNewLocal(nil, &rpfFileManager{filePath: "/existing"}, nil, nil)
		_, resp, err := l.LfsInfo(ctx, rpfHFInfo(), body("download"), "t")
		require.NoError(t, err)
		require.Contains(t, (*(*resp.Objects)[0].Actions)["download"].Href, "multipart/download/oid-1")

		l = rpfNewLocal(nil, &rpfFileManager{filePathErr: errors.New("missing")}, nil, nil)
		_, resp, err = l.LfsInfo(ctx, rpfHFInfo(), body("download"), "t")
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, (*resp.Objects)[0].Error.Code)
	})

	t.Run("unsupported operation", func(t *testing.T) {
		l := rpfNewLocal(nil, &rpfFileManager{}, nil, nil)
		_, resp, err := l.LfsInfo(ctx, rpfHFInfo(), body("delete"), "t")
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, (*resp.Objects)[0].Error.Code)
	})

	t.Run("bad and empty bodies", func(t *testing.T) {
		l := rpfNewLocal(nil, &rpfFileManager{}, nil, nil)
		headers, _, err := l.LfsInfo(ctx, rpfHFInfo(), io.NopCloser(strings.NewReader("{nope")), "t")
		require.Error(t, err)
		require.Equal(t, http.StatusBadRequest, headers.Code)

		headers, resp, err := l.LfsInfo(ctx, rpfHFInfo(), io.NopCloser(strings.NewReader(`{"operation":"upload"}`)), "t")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		require.Empty(t, *resp.Objects)
	})
}

func TestRpfLfsUploadAndVerify(t *testing.T) {
	ctx := context.Background()

	fm := &rpfFileManager{}
	l := rpfNewLocal(nil, fm, nil, nil)
	headers, resp, err := l.LfsUpload(ctx, rpfHFInfo(), io.NopCloser(strings.NewReader("weights")))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, headers.Code)
	require.True(t, resp.Success)
	require.Equal(t, "weights", string(fm.uploaded))

	failing := rpfNewLocal(nil, &rpfFileManager{uploadErr: errors.New("disk full")}, nil, nil)
	headers, resp, err = failing.LfsUpload(ctx, rpfHFInfo(), io.NopCloser(strings.NewReader("weights")))
	require.Error(t, err)
	require.Equal(t, http.StatusInternalServerError, headers.Code)
	require.False(t, resp.Success)

	// Verify succeeds when either the path or the digest is known.
	for name, fm := range map[string]*rpfFileManager{
		"by path":   {filePath: "/existing"},
		"by digest": {filePathErr: errors.New("x"), headDigest: true},
	} {
		t.Run(name, func(t *testing.T) {
			verifyHeaders, verifyResp, verifyErr := rpfNewLocal(nil, fm, nil, nil).LfsVerify(ctx, rpfHFInfo(), nil)
			require.NoError(t, verifyErr)
			require.Equal(t, http.StatusOK, verifyHeaders.Code)
			require.True(t, verifyResp.Success)
		})
	}

	_, _, err = rpfNewLocal(nil, &rpfFileManager{headErr: errors.New("x")}, nil, nil).LfsVerify(ctx, rpfHFInfo(), nil)
	require.ErrorContains(t, err, "does not exist")
}

func TestRpfCommitRevision(t *testing.T) {
	ctx := context.Background()
	commit := strings.Join([]string{
		`{"key":"header","value":{"summary":"Add weights","description":"initial commit"}}`,
		`{"key":"lfsFile","value":{"path":"weights.bin","algo":"sha256","oid":"oid-1","size":10}}`,
		`{"key":"lfsFile","value":{"path":"README.md","algo":"sha256","oid":"oid-2","size":4}}`,
		`{"key":"lfsFile","value":"not an object"}`,
		`this line is not json`,
		`{"key":"other","value":{}}`,
	}, "\n")

	t.Run("creates the artifact with the uploaded files", func(t *testing.T) {
		lb := &rpfLocalBase{}
		fm := &rpfFileManager{readme: "# Readme", t: t}
		l := rpfNewLocal(lb, fm, nil, nil)
		headers, resp, err := l.CommitRevision(ctx, rpfHFInfo(), io.NopCloser(strings.NewReader(commit)))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		require.Equal(t, "true", headers.Headers["X-Harness-Commit-Processed"])
		require.True(t, resp.Success)
		require.Equal(t, "model/org/model/commit/main", resp.CommitURL)
		require.Equal(t, "Add weights", resp.CommitMessage)
		require.Equal(t, "initial commit", resp.CommitDescription)
		require.Equal(t, "main", resp.OID)

		require.Equal(t, "/model/org/model/main", lb.gotPrefix)
		require.Equal(t, "main", lb.gotVersion)
		require.Len(t, lb.gotFiles, 2)
		require.Equal(t, "oid-1", lb.gotFiles[0].Sha256)
		hf, ok := lb.gotMeta.(*huggingfacemetadata.HuggingFaceMetadata)
		require.True(t, ok)
		require.Equal(t, "# Readme", hf.Readme)
		require.Equal(t, "org/model", hf.ID)
		require.Len(t, hf.Siblings, 2)
		require.Equal(t, []string{"summary:Add weights"}, hf.CardData.Tags)
		require.True(t, hf.Private)
	})

	t.Run("a missing readme blob is tolerated", func(t *testing.T) {
		lb := &rpfLocalBase{}
		fm := &rpfFileManager{readmeErr: errors.New("gone"), t: t}
		l := rpfNewLocal(lb, fm, nil, nil)
		_, _, err := l.CommitRevision(ctx, rpfHFInfo(), io.NopCloser(strings.NewReader(commit)))
		require.NoError(t, err)
		hf, ok := lb.gotMeta.(*huggingfacemetadata.HuggingFaceMetadata)
		require.True(t, ok)
		require.Empty(t, hf.Readme)
	})

	t.Run("a header that cannot be decoded fails the commit", func(t *testing.T) {
		l := rpfNewLocal(&rpfLocalBase{}, &rpfFileManager{t: t}, nil, nil)
		_, _, err := l.CommitRevision(ctx, rpfHFInfo(), io.NopCloser(strings.NewReader(
			`{"key":"header","value":"not an object"}`)))
		require.Error(t, err)
	})

	t.Run("storage failure", func(t *testing.T) {
		l := rpfNewLocal(&rpfLocalBase{err: errors.New("x")}, &rpfFileManager{t: t}, nil, nil)
		_, _, err := l.CommitRevision(ctx, rpfHFInfo(), io.NopCloser(bytes.NewBufferString(commit)))
		require.Error(t, err)
	})

	t.Run("read failure", func(t *testing.T) {
		l := rpfNewLocal(&rpfLocalBase{}, &rpfFileManager{t: t}, nil, nil)
		huge := strings.Repeat("x", 1<<20) // exceeds the scanner's default token size.
		_, _, err := l.CommitRevision(ctx, rpfHFInfo(), io.NopCloser(strings.NewReader(huge)))
		require.Error(t, err)
	})
}

func TestRpfHeadAndDownloadFile(t *testing.T) {
	ctx := context.Background()
	images := &rpfImageRepo{image: &types.Image{ID: 5}}
	artifacts := &rpfArtifactRepo{artifact: &types.Artifact{}}

	t.Run("head", func(t *testing.T) {
		l := rpfNewLocal(nil, &rpfFileManager{}, images, artifacts)
		headers, err := l.HeadFile(ctx, rpfHFInfo(), "weights.bin")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		require.Equal(t, "99", headers.Headers["Content-Length"])
		require.Equal(t, "main", headers.Headers["X-Repo-Commit"])
		require.Equal(t, "sha-head", headers.Headers["ETag"])

		for name, l := range map[string]*localRegistry{
			"image":    rpfNewLocal(nil, &rpfFileManager{}, &rpfImageRepo{err: errors.New("x")}, artifacts),
			"artifact": rpfNewLocal(nil, &rpfFileManager{}, images, &rpfArtifactRepo{err: errors.New("x")}),
			"file":     rpfNewLocal(nil, &rpfFileManager{headFileErr: errors.New("x")}, images, artifacts),
		} {
			headers, err = l.HeadFile(ctx, rpfHFInfo(), "weights.bin")
			require.Error(t, err, name)
			require.Equal(t, http.StatusNotFound, headers.Code, name)
		}
	})

	t.Run("download", func(t *testing.T) {
		l := rpfNewLocal(nil, &rpfFileManager{t: t}, images, artifacts)
		headers, body, redirect, err := l.DownloadFile(ctx, rpfHFInfo(), "weights.bin")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		require.NotNil(t, body)
		require.Equal(t, "https://redirect.test/model", redirect)

		deleted := &rpfArtifactRepo{artifact: &types.Artifact{}, versErr: errors.New("x")}
		gone := rpfNewLocal(nil, &rpfFileManager{t: t}, images, deleted)
		_, _, _, err = gone.DownloadFile(ctx, rpfHFInfo(), "weights.bin")
		require.ErrorContains(t, err, "artifact not found or deleted")

		noHead := rpfNewLocal(nil, &rpfFileManager{t: t, headFileErr: errors.New("x")}, images, artifacts)
		headers, _, _, err = noHead.DownloadFile(ctx, rpfHFInfo(), "weights.bin")
		require.Error(t, err)
		require.Equal(t, http.StatusNotFound, headers.Code)

		noStorage := rpfNewLocal(nil, &rpfFileManager{t: t, byPathErr: errors.New("x")}, images, artifacts)
		_, _, _, err = noStorage.DownloadFile(ctx, rpfHFInfo(), "weights.bin")
		require.Error(t, err)
	})

	t.Run("exists", func(t *testing.T) {
		require.True(t, rpfNewLocal(nil, &rpfFileManager{headDigest: true}, nil, nil).FileExists(ctx, rpfHFInfo()))
		require.False(t, rpfNewLocal(nil, &rpfFileManager{headErr: errors.New("x")}, nil, nil).FileExists(ctx, rpfHFInfo()))
	})
}
