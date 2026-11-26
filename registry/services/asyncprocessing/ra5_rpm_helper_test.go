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

package asyncprocessing

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	appcache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/metadata"
	rpmmetadata "github.com/harness/gitness/registry/app/metadata/rpm"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	rpmtypes "github.com/harness/gitness/registry/app/utils/rpm/types"
	"github.com/harness/gitness/registry/types"
	gtypes "github.com/harness/gitness/types"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
)

const (
	ra5PrimaryXMLFmt = `<?xml version="1.0" encoding="UTF-8"?>` +
		`<metadata xmlns="http://linux.duke.edu/metadata/common" ` +
		`xmlns:rpm="http://linux.duke.edu/metadata/rpm" packages="1">` +
		`<package type="rpm"><name>%s</name><arch>x86_64</arch>` +
		`<version epoch="0" ver="%s" rel="1"/>` +
		`<checksum type="sha256" pkgid="YES">sum%s</checksum>` +
		`<summary>sum</summary><location href="%s"/></package></metadata>`
	ra5OtherXMLFmt = `<?xml version="1.0" encoding="UTF-8"?>` +
		`<otherdata xmlns="http://linux.duke.edu/metadata/other" packages="1">` +
		`<package pkgid="%s" name="%s" arch="x86_64"><version epoch="0" ver="1.0" rel="1"/></package></otherdata>`
	ra5FileListsXMLFmt = `<?xml version="1.0" encoding="UTF-8"?>` +
		`<filelists xmlns="http://linux.duke.edu/metadata/filelists" packages="1">` +
		`<package pkgid="%s" name="%s" arch="x86_64"><version epoch="0" ver="1.0" rel="1"/>` +
		`<file>/usr/bin/%s</file></package></filelists>`
	ra5RepomdFmt = `<?xml version="1.0" encoding="UTF-8"?>` +
		`<repomd xmlns="http://linux.duke.edu/metadata/repo">` +
		`<data type="primary"><location href="repodata/primary.xml.gz"/></data>` +
		`<data type="filelists"><location href="repodata/filelists.xml.gz"/></data>` +
		`<data type="other"><location href="repodata/other.xml.gz"/></data>` +
		`</repomd>`
	ra5RepomdPrimaryOnly = `<repomd><data type="primary"><location href="repodata/primary.xml.gz"/></data></repomd>`
	ra5Pkg               = "localpkg"
	ra5Root              = "rootspace"
	ra5RepomdPath        = "/repodata/repomd.xml"
	ra5PrimaryRef        = "repodata/primary.xml.gz"
	ra5FileListsRef      = "repodata/filelists.xml.gz"
	ra5OtherRef          = "repodata/other.xml.gz"
	ra5Up                = "up"
)

var errRa5Boom = errors.New("boom")

func ra5Decoder(s string) *xml.Decoder { return xml.NewDecoder(strings.NewReader(s)) }

func ra5Gz(t *testing.T, s string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write([]byte(s))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func ra5Gunzip(t *testing.T, b []byte) string {
	t.Helper()
	r, err := gzip.NewReader(bytes.NewReader(b))
	require.NoError(t, err)
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

func ra5Xz(t *testing.T, s string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := xz.NewWriter(&buf)
	require.NoError(t, err)
	_, err = w.Write([]byte(s))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func ra5Zst(t *testing.T, s string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := zstd.NewWriter(&buf)
	require.NoError(t, err)
	_, err = w.Write([]byte(s))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

// ra5Driver is a storage driver serving fixed content.
type ra5Driver struct {
	driver.StorageDriver
	content []byte
}

func (d ra5Driver) Reader(_ context.Context, _ string, offset int64) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(d.content[offset:])), nil
}

// ra5FileManager is an in-memory filemanager.FileManager.
type ra5FileManager struct {
	filemanager.FileManager

	mu        sync.Mutex
	files     map[string][]byte
	dlErrs    map[string]error
	uploaded  [][]byte
	postPaths []string
	repomd    []byte
	repomdAt  string

	uploadErr     error
	postErr       error
	uploadFileErr error
}

func (f *ra5FileManager) key(regID int64, path string) string {
	return fmt.Sprintf("%d%s", regID, path)
}

func (f *ra5FileManager) UploadFileNoDBUpdate(
	_ context.Context, _ string, _ multipart.File, r io.Reader, _ int64, _ int64,
) (types.FileInfo, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return types.FileInfo{}, err
	}
	if f.uploadErr != nil {
		return types.FileInfo{}, f.uploadErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.uploaded = append(f.uploaded, data)
	sum := sha256.Sum256(data)
	return types.FileInfo{Sha256: hex.EncodeToString(sum[:]), Size: int64(len(data))}, nil
}

func (f *ra5FileManager) PostFileUpload(
	_ context.Context, filePath string, _ int64, _ int64, _ string, _ types.FileInfo, _ int64,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.postPaths = append(f.postPaths, filePath)
	return f.postErr
}

func (f *ra5FileManager) UploadFile(
	_ context.Context, filePath string, _ int64, _ int64, _ string, _ multipart.File, r io.Reader, _ int64,
) (types.FileInfo, error) {
	if f.uploadFileErr != nil {
		return types.FileInfo{}, f.uploadFileErr
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return types.FileInfo{}, err
	}
	f.repomd = data
	f.repomdAt = filePath
	return types.FileInfo{}, nil
}

func (f *ra5FileManager) DownloadFileByPath(
	ctx context.Context, filePath string, registryID int64, _ string, _ string, _ bool,
) (*storage.FileReader, int64, string, error) {
	k := f.key(registryID, filePath)
	if err, ok := f.dlErrs[k]; ok {
		return nil, 0, "", err
	}
	data, ok := f.files[k]
	if !ok {
		return nil, 0, "", errors.New("file not found: " + k)
	}
	fr, err := storage.NewFileReader(ctx, ra5Driver{content: data}, filePath, int64(len(data)))
	return fr, int64(len(data)), "", err
}

type ra5Remote struct {
	files map[string][]byte
}

func (r ra5Remote) GetMetadataFile(_ context.Context, p string) (io.ReadCloser, error) {
	data, ok := r.files[p]
	if !ok {
		return nil, errRa5Boom
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (r ra5Remote) GetPackage(ctx context.Context, p string) (io.ReadCloser, error) {
	return r.GetMetadataFile(ctx, p)
}

type ra5ArtifactRepo struct {
	store.ArtifactRepository
	items []types.ArtifactMetadata
	err   error
}

func (a ra5ArtifactRepo) GetAllArtifactsByRepo(
	_ context.Context, _ int64, batch int, lastID int64,
) (*[]types.ArtifactMetadata, error) {
	if a.err != nil {
		return nil, a.err
	}
	out := make([]types.ArtifactMetadata, 0, batch)
	for _, it := range a.items {
		if it.ID > lastID && len(out) < batch {
			out = append(out, it)
		}
	}
	return &out, nil
}

type ra5UpstreamStore struct {
	store.UpstreamProxyConfigRepository
	proxy *types.UpstreamProxy
	err   error
}

func (u ra5UpstreamStore) Get(_ context.Context, _ int64, _ ...types.QueryOption) (*types.UpstreamProxy, error) {
	return u.proxy, u.err
}

type ra5RegistryRepo struct {
	store.RegistryRepository
	byName *types.Registry
	byIDs  []types.Registry
	err    error
	idsErr error
}

func (r ra5RegistryRepo) GetByParentIDAndName(
	_ context.Context, _ int64, _ string, _ ...types.QueryOption,
) (*types.Registry, error) {
	return r.byName, r.err
}

func (r ra5RegistryRepo) GetByIDIn(_ context.Context, _ []int64, _ ...types.QueryOption) (*[]types.Registry, error) {
	return &r.byIDs, r.idsErr
}

type ra5SpaceCache struct {
	space *gtypes.SpaceCore
	err   error
}

func (c ra5SpaceCache) Stats() (int64, int64) { return 0, 0 }

func (c ra5SpaceCache) Get(_ context.Context, _ int64) (*gtypes.SpaceCore, error) {
	return c.space, c.err
}

func (c ra5SpaceCache) Evict(_ context.Context, _ int64) {}

func ra5Helper(fm *ra5FileManager, art store.ArtifactRepository, spaceErr error) *rpmHelper {
	sf := refcache.NewSpaceFinder(ra5SpaceCache{space: &gtypes.SpaceCore{ID: 9, Identifier: ra5Root}, err: spaceErr},
		nil, nil, appcache.Evictor[*gtypes.SpaceCore]{})
	return &rpmHelper{fileManager: fm, artifactDao: art, spaceFinder: sf}
}

func ra5PackageInfo() []*rpmtypes.PackageInfo {
	return []*rpmtypes.PackageInfo{{
		Name:   ra5Pkg,
		Sha256: "sha" + ra5Pkg,
		Size:   10,
		VersionMetadata: &rpmmetadata.VersionMetadata{
			Summary: "local summary", License: "MIT",
		},
		FileMetadata: &rpmmetadata.FileMetadata{
			Architecture: "x86_64", Epoch: "1", Version: "2.0", Release: "3",
			Files:      []*rpmmetadata.File{{Path: "/usr/bin/tool"}, {Path: "/opt/data"}, {Path: "/etc/tool.conf"}},
			Changelogs: []*rpmmetadata.Changelog{{Author: "dev", Date: 1, Text: "init"}},
		},
	}}
}

func ra5LocalData(fm *ra5FileManager, regName, ref string) registryData {
	return &localRepoData{fileManager: fm, fileRef: ref, registryID: 5, registryIdentifier: regName,
		rootIdentifier: ra5Root}
}

func TestRa5GetReader(t *testing.T) {
	const payload = "hello"
	tests := []struct {
		name    string
		ref     string
		data    []byte
		wantErr string
	}{
		{"gzip", "a.xml.gz", ra5Gz(t, payload), ""},
		{"xz upper", "a.XML.XZ", ra5Xz(t, payload), ""},
		{"zst", "a.xml.zst", ra5Zst(t, payload), ""},
		{"bad gzip", "a.xml.gz", []byte("nope"), "gzip reader"},
		{"bad xz", "a.xml.xz", []byte("nope"), "xz reader"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := getReader(tc.ref, io.NopCloser(bytes.NewReader(tc.data)))
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			out, err := io.ReadAll(r)
			require.NoError(t, err)
			require.Equal(t, payload, string(out))
			if c, ok := r.(io.Closer); ok {
				require.NoError(t, c.Close())
			}
		})
	}
}

func TestRa5GetRefs(t *testing.T) {
	ctx := context.Background()
	h := &rpmHelper{}

	p, f, o, err := h.getRefs(ctx, ra5Remote{files: map[string][]byte{ra5RepomdPath: []byte(ra5RepomdFmt)}})
	require.NoError(t, err)
	require.Equal(t, ra5PrimaryRef, p)
	require.Equal(t, ra5FileListsRef, f)
	require.Equal(t, ra5OtherRef, o)

	_, _, _, err = h.getRefs(ctx, ra5Remote{files: map[string][]byte{}})
	require.ErrorIs(t, err, errRa5Boom)

	_, _, _, err = h.getRefs(ctx, ra5Remote{files: map[string][]byte{ra5RepomdPath: []byte("<repomd>")}})
	require.ErrorContains(t, err, "failed to parse repomd.xml")
}

func TestRa5GetRefsForHarnessRepos(t *testing.T) {
	ctx := context.Background()
	fm := &ra5FileManager{
		files: map[string][]byte{
			"1" + ra5RepomdPath: []byte(ra5RepomdFmt),
			"2" + ra5RepomdPath: []byte("<repomd>"),
		},
		dlErrs: map[string]error{"4" + ra5RepomdPath: errRa5Boom},
	}
	h := &rpmHelper{fileManager: fm}
	virtual := artifact.RegistryTypeVIRTUAL

	ref, err := h.getRefsForHarnessRepos(ctx, 1, "r1", ra5Root, "filelists", virtual)
	require.NoError(t, err)
	require.Equal(t, ra5FileListsRef, ref)

	ref, err = h.getRefsForHarnessRepos(ctx, 1, "r1", ra5Root, "unknown", virtual)
	require.NoError(t, err)
	require.Empty(t, ref)

	_, err = h.getRefsForHarnessRepos(ctx, 2, "r2", ra5Root, "primary", virtual)
	require.ErrorContains(t, err, "failed to parse repomd.xml")

	// Missing repomd is tolerated for virtual registries only.
	ref, err = h.getRefsForHarnessRepos(ctx, 3, "r3", ra5Root, "primary", virtual)
	require.NoError(t, err)
	require.Empty(t, ref)
	_, err = h.getRefsForHarnessRepos(ctx, 3, "r3", ra5Root, "primary", artifact.RegistryTypeUPSTREAM)
	require.ErrorContains(t, err, "file not found")

	_, err = h.getRefsForHarnessRepos(ctx, 4, "r4", ra5Root, "primary", virtual)
	require.ErrorIs(t, err, errRa5Boom)
}

func TestRa5ValidateRootElement(t *testing.T) {
	h := &rpmHelper{}
	require.NoError(t, h.validateRootElement(ra5Decoder(`<?xml version="1.0"?><metadata/>`), "metadata"))
	require.ErrorContains(t, h.validateRootElement(ra5Decoder(`<other/>`), "metadata"), "unexpected root element")
	require.ErrorContains(t, h.validateRootElement(ra5Decoder(``), "metadata"), "failed to read start element")
}

func TestRa5GetPrimaryPackage(t *testing.T) {
	pi := ra5PackageInfo()[0]
	key, pp := getPrimaryPackage(pi, "../../repo/rpm/package")
	require.Equal(t, "localpkg:2.0-3:x86_64", key)
	require.Equal(t, "../../repo/rpm/package/localpkg/1:2.0-3/x86_64/localpkg-2.0-3.x86_64.rpm", pp.Location.Href)
	require.Len(t, pp.Format.Files, 2)

	pi.FileMetadata.Epoch = "0"
	_, pp = getPrimaryPackage(pi, "root")
	require.Equal(t, "root/localpkg/2.0-3/x86_64/localpkg-2.0-3.x86_64.rpm", pp.Location.Href)
}

func TestRa5BuildPrimary(t *testing.T) {
	ctx := context.Background()
	remotePrimary := ra5Gz(t, fmt.Sprintf(ra5PrimaryXMLFmt, "remotepkg", "1.0", "r", "Packages/remotepkg.rpm"))
	dupPrimary := ra5Gz(t, fmt.Sprintf(ra5PrimaryXMLFmt, ra5Pkg, "2.0", "d", "Packages/dup.rpm"))
	dupPrimary = ra5Gz(t, strings.Replace(ra5Gunzip(t, dupPrimary), `rel="1"`, `rel="3"`, 1))
	localPrimary := ra5Gz(t, fmt.Sprintf(ra5PrimaryXMLFmt, "childpkg", "1.0", "c", "Packages/child.rpm"))
	prefixed := ra5Gz(t, fmt.Sprintf(ra5PrimaryXMLFmt, "prefixed", "1.0", "p", "../../childa/rpm/Packages/p.rpm"))

	t.Run("remote and local data merged", func(t *testing.T) {
		fm := &ra5FileManager{files: map[string][]byte{
			"5/" + ra5PrimaryRef:  localPrimary,
			"5/repodata/p.xml.gz": prefixed,
		}}
		h := ra5Helper(fm, nil, nil)
		remote := &remoteRepoData{
			helper:  ra5Remote{files: map[string][]byte{"/" + ra5PrimaryRef: remotePrimary, "/dup.gz": dupPrimary}},
			fileRef: ra5PrimaryRef,
		}
		dup := &remoteRepoData{helper: remote.helper, fileRef: "dup.gz"}
		data, err := h.buildPrimary(ctx, ra5PackageInfo(), 1, 9, ra5Root, ra5Up, true,
			[]registryData{remote, dup}, 3)
		require.NoError(t, err)
		require.Equal(t, "primary", data.Type)
		require.Equal(t, []string{"repodata/" + data.Checksum.Value + "-primary.xml.gz"}, fm.postPaths)

		out := ra5Gunzip(t, fm.uploaded[0])
		require.Contains(t, out, "<name>localpkg</name>")
		require.Equal(t, 1, strings.Count(out, "<name>localpkg</name>"))
		require.Contains(t, out, "<name>remotepkg</name>")
		require.Contains(t, out, "../../up/rpm/package/remotepkg/1.0-1/x86_64/remotepkg-1.0-1.x86_64.rpm/Packages/")

		fm2 := &ra5FileManager{files: fm.files}
		h2 := ra5Helper(fm2, nil, nil)
		_, err = h2.buildPrimary(ctx, nil, 1, 9, ra5Root, "virt", false, []registryData{
			ra5LocalData(fm2, "childa", ra5PrimaryRef),
			ra5LocalData(fm2, "childa", "repodata/p.xml.gz"),
		}, 3)
		require.NoError(t, err)
		out = ra5Gunzip(t, fm2.uploaded[0])
		require.Contains(t, out, `href="../../childa/rpm/Packages/child.rpm"`)
		require.Contains(t, out, `href="../../childa/rpm/Packages/p.rpm"`)
		require.NotContains(t, out, "rpm/../../childa")
	})

	failures := []struct {
		name    string
		fm      *ra5FileManager
		data    []registryData
		wantErr string
	}{
		{
			name:    "reader fetch fails",
			fm:      &ra5FileManager{files: map[string][]byte{}},
			data:    []registryData{ra5LocalData(nil, "c", "missing.gz")},
			wantErr: "failed to get primary.xml.gz",
		},
		{
			name:    "bad compression",
			fm:      &ra5FileManager{files: map[string][]byte{"5/bad.gz": []byte("nope")}},
			data:    []registryData{ra5LocalData(nil, "c", "bad.gz")},
			wantErr: "failed to create reader",
		},
		{
			name:    "bad root",
			fm:      &ra5FileManager{files: map[string][]byte{"5/root.gz": ra5Gz(t, "<other/>")}},
			data:    []registryData{ra5LocalData(nil, "c", "root.gz")},
			wantErr: "failed to validate primary.xml root element",
		},
		{
			name:    "truncated package",
			fm:      &ra5FileManager{files: map[string][]byte{"5/trunc.gz": ra5Gz(t, "<metadata><package>")}},
			data:    []registryData{ra5LocalData(nil, "c", "trunc.gz")},
			wantErr: "failed to decode package",
		},
		{
			name:    "truncated xml",
			fm:      &ra5FileManager{files: map[string][]byte{"5/trunc.gz": ra5Gz(t, "<metadata>")}},
			data:    []registryData{ra5LocalData(nil, "c", "trunc.gz")},
			wantErr: "error reading primary.xml",
		},
		{
			name: "upload fails",
			fm:   &ra5FileManager{uploadErr: errRa5Boom},
			// no data, error comes from the upload itself.
			wantErr: "failed to upload file",
		},
		{
			name:    "post upload fails",
			fm:      &ra5FileManager{postErr: errRa5Boom},
			wantErr: "failed to move temp file",
		},
	}
	for _, tc := range failures {
		t.Run(tc.name, func(t *testing.T) {
			for _, d := range tc.data {
				d.(*localRepoData).fileManager = tc.fm //nolint:errcheck,forcetypeassert // test fixture.
			}
			h := ra5Helper(tc.fm, nil, nil)
			_, err := h.buildPrimary(ctx, nil, 1, 9, ra5Root, "virt", false, tc.data, 3)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestRa5BuildOtherAndFileLists(t *testing.T) {
	ctx := context.Background()
	other := ra5Gz(t, fmt.Sprintf(ra5OtherXMLFmt, "otherid", "otherpkg"))
	dupOther := ra5Gz(t, fmt.Sprintf(ra5OtherXMLFmt, "sha"+ra5Pkg, ra5Pkg))
	lists := ra5Gz(t, fmt.Sprintf(ra5FileListsXMLFmt, "listid", "listpkg", "listpkg"))
	dupLists := ra5Gz(t, fmt.Sprintf(ra5FileListsXMLFmt, "sha"+ra5Pkg, ra5Pkg, ra5Pkg))
	files := map[string][]byte{
		"5/o.gz": other, "5/od.gz": dupOther, "5/l.gz": lists, "5/ld.gz": dupLists,
		"5/ob.gz": []byte("nope"), "5/oroot.gz": ra5Gz(t, "<bad/>"),
		"5/ltrunc.gz": ra5Gz(t, "<filelists><package>"), "5/ltrunc2.gz": ra5Gz(t, "<filelists>"),
	}

	t.Run("other merged", func(t *testing.T) {
		fm := &ra5FileManager{files: files}
		h := ra5Helper(fm, nil, nil)
		data, err := h.buildOther(ctx, ra5PackageInfo(), 1, 9, ra5Root, []registryData{
			ra5LocalData(fm, "c", "o.gz"), ra5LocalData(fm, "c", "od.gz"),
		}, 3)
		require.NoError(t, err)
		require.Equal(t, "other", data.Type)
		out := ra5Gunzip(t, fm.uploaded[0])
		require.Contains(t, out, "otherpkg")
		require.Equal(t, 1, strings.Count(out, `pkgid="sha`+ra5Pkg+`"`))
		require.Contains(t, out, `author="dev"`)
		require.Contains(t, fm.postPaths[0], "-other.xml.gz")
	})

	t.Run("filelists merged", func(t *testing.T) {
		fm := &ra5FileManager{files: files}
		h := ra5Helper(fm, nil, nil)
		data, err := h.buildFileLists(ctx, ra5PackageInfo(), 1, 9, ra5Root, []registryData{
			ra5LocalData(fm, "c", "l.gz"), ra5LocalData(fm, "c", "ld.gz"),
		}, 3)
		require.NoError(t, err)
		require.Equal(t, "filelists", data.Type)
		out := ra5Gunzip(t, fm.uploaded[0])
		require.Contains(t, out, "listpkg")
		require.Equal(t, 1, strings.Count(out, `pkgid="sha`+ra5Pkg+`"`))
		require.Contains(t, fm.postPaths[0], "-filelists.xml.gz")
	})

	type buildFn func(h *rpmHelper, d []registryData) error
	builders := map[string]buildFn{
		"other": func(h *rpmHelper, d []registryData) error {
			_, err := h.buildOther(ctx, nil, 1, 9, ra5Root, d, 3)
			return err
		},
		"filelists": func(h *rpmHelper, d []registryData) error {
			_, err := h.buildFileLists(ctx, nil, 1, 9, ra5Root, d, 3)
			return err
		},
	}
	for name, build := range builders {
		cases := []struct {
			sub     string
			ref     string
			wantErr string
		}{
			{"fetch", "missing.gz", "failed to get"},
			{"compression", "ob.gz", "failed to create reader"},
			{"root", "oroot.gz", "root element"},
		}
		for _, tc := range cases {
			t.Run(name+" "+tc.sub, func(t *testing.T) {
				fm := &ra5FileManager{files: files}
				err := build(ra5Helper(fm, nil, nil), []registryData{ra5LocalData(fm, "c", tc.ref)})
				require.ErrorContains(t, err, tc.wantErr)
			})
		}
		t.Run(name+" upload", func(t *testing.T) {
			require.ErrorContains(t, build(ra5Helper(&ra5FileManager{uploadErr: errRa5Boom}, nil, nil), nil),
				"failed to upload file")
		})
		t.Run(name+" post", func(t *testing.T) {
			require.ErrorContains(t, build(ra5Helper(&ra5FileManager{postErr: errRa5Boom}, nil, nil), nil),
				"failed to move temp file")
		})
	}

	t.Run("filelists truncated package", func(t *testing.T) {
		fm := &ra5FileManager{files: files}
		_, err := ra5Helper(fm, nil, nil).buildFileLists(ctx, nil, 1, 9, ra5Root,
			[]registryData{ra5LocalData(fm, "c", "ltrunc.gz")}, 3)
		require.ErrorContains(t, err, "failed to decode filelists package")
	})

	t.Run("filelists truncated", func(t *testing.T) {
		fm := &ra5FileManager{files: files}
		_, err := ra5Helper(fm, nil, nil).buildFileLists(ctx, nil, 1, 9, ra5Root,
			[]registryData{ra5LocalData(fm, "c", "ltrunc2.gz")}, 3)
		require.ErrorContains(t, err, "error reading filelists.xml")
	})
}

func TestRa5BuildRepomd(t *testing.T) {
	ctx := context.Background()
	fm := &ra5FileManager{}
	h := ra5Helper(fm, nil, nil)
	data := []*rpmtypes.RepoData{{Type: "primary", Location: rpmtypes.RepoLocation{Href: "repodata/x"}}, nil}
	require.NoError(t, h.buildRepoMDFile(ctx, 1, 9, data[0], nil, nil, ra5Root, 3))
	require.Equal(t, "repodata/repomd.xml", fm.repomdAt)
	require.Contains(t, string(fm.repomd), `<data type="primary">`)

	fm.uploadFileErr = errRa5Boom
	require.ErrorIs(t, h.buildRepomd(ctx, data, 1, 9, ra5Root, 3), errRa5Boom)
}

func ra5ArtifactItems(t *testing.T, n int) []types.ArtifactMetadata {
	t.Helper()
	items := make([]types.ArtifactMetadata, 0, n)
	for i := 1; i <= n; i++ {
		md := rpmmetadata.RpmMetadata{
			Metadata: rpmmetadata.Metadata{
				FileMetadata:    rpmmetadata.FileMetadata{Architecture: "x86_64", Version: "1.0", Release: "1"},
				VersionMetadata: rpmmetadata.VersionMetadata{Summary: "s"},
			},
			Files: []metadata.File{{Size: 7, Sha256: fmt.Sprintf("sha%d", i)}},
		}
		raw, err := json.Marshal(md)
		require.NoError(t, err)
		items = append(items, types.ArtifactMetadata{ID: int64(i), Name: fmt.Sprintf("pkg%d", i), Metadata: raw})
	}
	return items
}

func TestRa5GetExistingArtifactInfos(t *testing.T) {
	ctx := context.Background()

	h := ra5Helper(nil, ra5ArtifactRepo{items: ra5ArtifactItems(t, artifactBatchLimit+3)}, nil)
	infos, err := h.getExistingArtifactInfos(ctx, 1)
	require.NoError(t, err)
	require.Len(t, infos, artifactBatchLimit+3)
	require.Equal(t, "sha1", infos[0].Sha256)
	require.Equal(t, int64(7), infos[0].Size)

	h = ra5Helper(nil, ra5ArtifactRepo{err: errRa5Boom}, nil)
	_, err = h.getExistingArtifactInfos(ctx, 1)
	require.ErrorIs(t, err, errRa5Boom)

	h = ra5Helper(nil, ra5ArtifactRepo{items: []types.ArtifactMetadata{{ID: 1, Metadata: []byte("{")}}}, nil)
	_, err = h.getExistingArtifactInfos(ctx, 1)
	require.Error(t, err)
}

func TestRa5BuildRegistryFilesVirtual(t *testing.T) {
	ctx := context.Background()
	files := map[string][]byte{
		"2" + ra5RepomdPath: []byte(ra5RepomdFmt),
		"2/" + ra5PrimaryRef: ra5Gz(t, fmt.Sprintf(ra5PrimaryXMLFmt, "childpkg", "1.0", "c",
			"Packages/childpkg.rpm")),
		"2/" + ra5FileListsRef: ra5Gz(t, fmt.Sprintf(ra5FileListsXMLFmt, "cid", "childpkg", "childpkg")),
		"2/" + ra5OtherRef:     ra5Gz(t, fmt.Sprintf(ra5OtherXMLFmt, "cid", "childpkg")),
	}
	reg := types.Registry{ID: 1, Name: "virt", ParentID: 7, RootParentID: 9, Type: artifact.RegistryTypeVIRTUAL}
	regDao := ra5RegistryRepo{
		byName: &types.Registry{ID: 1, Name: "virt", UpstreamProxies: []int64{2, 3, 4}},
		byIDs: []types.Registry{
			{ID: 2, Name: "childa", Type: artifact.RegistryTypeVIRTUAL},
			{ID: 3, Name: "childb", Type: artifact.RegistryTypeVIRTUAL},
		},
	}

	t.Run("success", func(t *testing.T) {
		fm := &ra5FileManager{files: files}
		h := ra5Helper(fm, ra5ArtifactRepo{items: ra5ArtifactItems(t, 2)}, nil)
		h.registryDao = regDao
		require.NoError(t, h.BuildRegistryFiles(ctx, reg, 3))
		require.Len(t, fm.uploaded, 3)
		require.Len(t, fm.postPaths, 3)
		require.Equal(t, "repodata/repomd.xml", fm.repomdAt)
		require.Equal(t, 3, strings.Count(string(fm.repomd), "<data type="))
		require.Contains(t, ra5Gunzip(t, fm.uploaded[0]), "../../childa/rpm/Packages/childpkg.rpm")
	})

	t.Run("space lookup fails", func(t *testing.T) {
		h := ra5Helper(&ra5FileManager{}, nil, errRa5Boom)
		require.ErrorContains(t, h.BuildRegistryFiles(ctx, reg, 3), "failed to find root space by ID")
	})

	t.Run("artifact listing fails", func(t *testing.T) {
		h := ra5Helper(&ra5FileManager{}, ra5ArtifactRepo{err: errRa5Boom}, nil)
		require.ErrorIs(t, h.BuildRegistryFiles(ctx, reg, 3), errRa5Boom)
	})

	t.Run("registry not found", func(t *testing.T) {
		h := ra5Helper(&ra5FileManager{}, ra5ArtifactRepo{}, nil)
		h.registryDao = ra5RegistryRepo{err: errRa5Boom}
		require.Error(t, h.BuildRegistryFiles(ctx, reg, 3))
	})

	t.Run("child repomd fails", func(t *testing.T) {
		fm := &ra5FileManager{dlErrs: map[string]error{"2" + ra5RepomdPath: errRa5Boom}}
		h := ra5Helper(fm, ra5ArtifactRepo{}, nil)
		h.registryDao = regDao
		require.ErrorIs(t, h.BuildRegistryFiles(ctx, reg, 3), errRa5Boom)
	})

	t.Run("later stage failures", func(t *testing.T) {
		// Fail at the n-th data lookup by breaking the matching metadata file.
		for name, drop := range map[string]string{
			"primary":   "2/" + ra5PrimaryRef,
			"filelists": "2/" + ra5FileListsRef,
			"other":     "2/" + ra5OtherRef,
		} {
			t.Run(name, func(t *testing.T) {
				broken := make(map[string][]byte, len(files))
				for k, v := range files {
					broken[k] = v
				}
				delete(broken, drop)
				fm := &ra5FileManager{files: broken}
				h := ra5Helper(fm, ra5ArtifactRepo{}, nil)
				h.registryDao = regDao
				require.ErrorContains(t, h.BuildRegistryFiles(ctx, reg, 3), "failed to get")
			})
		}
	})
}

func TestRa5BuildRegistryFilesUpstream(t *testing.T) {
	ctx := context.Background()
	mux := http.NewServeMux()
	serve := func(path string, body []byte) {
		mux.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(body)
		})
	}
	var repomd string
	mux.HandleFunc(ra5RepomdPath, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(repomd))
	})
	serve("/"+ra5PrimaryRef, ra5Gz(t, fmt.Sprintf(ra5PrimaryXMLFmt, "remotepkg", "1.0", "r", "Packages/remotepkg.rpm")))
	serve("/"+ra5FileListsRef, ra5Gz(t, fmt.Sprintf(ra5FileListsXMLFmt, "rid", "remotepkg", "remotepkg")))
	serve("/"+ra5OtherRef, ra5Gz(t, fmt.Sprintf(ra5OtherXMLFmt, "rid", "remotepkg")))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	reg := types.Registry{ID: 1, Name: ra5Up, RootParentID: 9, Type: artifact.RegistryTypeUPSTREAM}
	proxy := &types.UpstreamProxy{
		RepoKey: ra5Up, RepoURL: srv.URL, RepoAuthType: string(artifact.AuthTypeAnonymous),
		PackageType: artifact.PackageTypeRPM,
	}
	newHelper := func(fm *ra5FileManager, up ra5UpstreamStore) *rpmHelper {
		h := ra5Helper(fm, ra5ArtifactRepo{items: ra5ArtifactItems(t, 1)}, nil)
		h.upstreamProxyStore = up
		return h
	}

	t.Run("full", func(t *testing.T) {
		repomd = ra5RepomdFmt
		fm := &ra5FileManager{}
		require.NoError(t, newHelper(fm, ra5UpstreamStore{proxy: proxy}).BuildRegistryFiles(ctx, reg, 3))
		require.Len(t, fm.uploaded, 3)
		require.Contains(t, ra5Gunzip(t, fm.uploaded[0]), "remotepkg")
		require.Equal(t, 3, strings.Count(string(fm.repomd), "<data type="))
	})

	t.Run("primary only", func(t *testing.T) {
		repomd = ra5RepomdPrimaryOnly
		fm := &ra5FileManager{}
		require.NoError(t, newHelper(fm, ra5UpstreamStore{proxy: proxy}).BuildRegistryFiles(ctx, reg, 3))
		require.Len(t, fm.uploaded, 1)
		require.Equal(t, 1, strings.Count(string(fm.repomd), "<data type="))
	})

	t.Run("store fails", func(t *testing.T) {
		fm := &ra5FileManager{}
		require.ErrorIs(t, newHelper(fm, ra5UpstreamStore{err: errRa5Boom}).BuildRegistryFiles(ctx, reg, 3), errRa5Boom)
	})

	t.Run("repomd missing upstream", func(t *testing.T) {
		repomd = ra5RepomdFmt
		bad := *proxy
		bad.RepoURL = srv.URL + "/missing"
		fm := &ra5FileManager{}
		require.Error(t, newHelper(fm, ra5UpstreamStore{proxy: &bad}).BuildRegistryFiles(ctx, reg, 3))
	})

	t.Run("primary missing", func(t *testing.T) {
		repomd = strings.Replace(ra5RepomdFmt, ra5PrimaryRef, "repodata/gone.xml.gz", 1)
		fm := &ra5FileManager{}
		require.Error(t, newHelper(fm, ra5UpstreamStore{proxy: proxy}).BuildRegistryFiles(ctx, reg, 3))
	})

	t.Run("filelists missing", func(t *testing.T) {
		repomd = strings.Replace(ra5RepomdFmt, ra5FileListsRef, "repodata/gone.xml.gz", 1)
		fm := &ra5FileManager{}
		require.Error(t, newHelper(fm, ra5UpstreamStore{proxy: proxy}).BuildRegistryFiles(ctx, reg, 3))
	})

	t.Run("other missing", func(t *testing.T) {
		repomd = strings.Replace(ra5RepomdFmt, ra5OtherRef, "repodata/gone.xml.gz", 1)
		fm := &ra5FileManager{}
		require.Error(t, newHelper(fm, ra5UpstreamStore{proxy: proxy}).BuildRegistryFiles(ctx, reg, 3))
	})
}

func TestRa5NewRpmHelper(t *testing.T) {
	require.NotNil(t, NewRpmHelper(nil, nil, nil, refcache.SpaceFinder{}, nil, nil))
}
