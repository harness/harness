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

package rpm

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"mime/multipart"
	"net/http"
	"sort"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/metadata"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	rpmtype "github.com/harness/gitness/registry/app/pkg/types/rpm"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

const (
	rpgTagName    = 1000
	rpgTagVersion = 1001
	rpgTagRelease = 1002
	rpgTagEpoch   = 1003
	rpgTagArch    = 1022
	rpgTagSummary = 1004

	rpgTypeInt32  = 4
	rpgTypeString = 6
	rpgIntroMagic = 0x8eade801
)

type rpgHeaderTag struct {
	tag    int32
	str    string
	number int32
}

// rpgBuildRPM assembles a minimal rpm: lead, empty signature header and a general header.
func rpgBuildRPM(tags ...rpgHeaderTag) []byte {
	sort.Slice(tags, func(i, j int) bool { return tags[i].tag < tags[j].tag })
	var table, data bytes.Buffer
	for _, tag := range tags {
		if tag.str != "" {
			_ = binary.Write(&table, binary.BigEndian, [4]int32{tag.tag, rpgTypeString, int32(data.Len()), 1})
			data.WriteString(tag.str)
			data.WriteByte(0)
			continue
		}
		for data.Len()%4 != 0 {
			data.WriteByte(0)
		}
		_ = binary.Write(&table, binary.BigEndian, [4]int32{tag.tag, rpgTypeInt32, int32(data.Len()), 1})
		_ = binary.Write(&data, binary.BigEndian, tag.number)
	}

	var out bytes.Buffer
	lead := make([]byte, 96)
	binary.BigEndian.PutUint32(lead, 0xedabeedb)
	out.Write(lead)
	_ = binary.Write(&out, binary.BigEndian, [4]uint32{rpgIntroMagic, 0, 0, 0})
	_ = binary.Write(&out, binary.BigEndian, [4]uint32{rpgIntroMagic, 0, uint32(len(tags)), uint32(data.Len())})
	out.Write(table.Bytes())
	out.Write(data.Bytes())
	return out.Bytes()
}

type rpgDriver struct {
	storagedriver.StorageDriver
	content []byte
}

func (d *rpgDriver) Reader(context.Context, string, int64) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(d.content)), nil
}

type rpgFileManager struct {
	filemanager.FileManager
	content []byte
	t       *testing.T
}

func (f *rpgFileManager) UploadFileNoDBUpdate(
	context.Context, string, multipart.File, io.Reader, int64, int64,
) (types.FileInfo, error) {
	return types.FileInfo{Sha256: "abc", Size: int64(len(f.content))}, nil
}

func (f *rpgFileManager) DownloadFileByDigest(
	ctx context.Context, _ string, _ types.FileInfo, _ int64, _ int64,
) (*storage.FileReader, error) {
	return storage.NewFileReader(ctx, &rpgDriver{content: f.content}, "p", int64(len(f.content)))
}

type rpgLocalBase struct {
	base.LocalBase
	err      error
	existent bool
	gotPath  string
	gotMeta  metadata.Metadata
	gotVer   string
}

func (b *rpgLocalBase) UpdateFileManagerAndCreateArtifact(
	_ context.Context, _ pkg.ArtifactInfo, version, path string, metadata metadata.Metadata, _ types.FileInfo, _ bool,
) (*commons.ResponseHeaders, string, int64, bool, error) {
	b.gotVer, b.gotPath, b.gotMeta = version, path, metadata
	return &commons.ResponseHeaders{Code: http.StatusCreated}, "sha", 7, b.existent, b.err
}

func rpgInfo() rpmtype.ArtifactInfo {
	return rpmtype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 1, RootIdentifier: "root", RootParentID: 1},
			RegIdentifier: "reg",
			RegistryID:    3,
		},
	}
}

func TestRpgUploadPackage(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		tags        []rpgHeaderTag
		wantVersion string
		wantPath    string
	}{
		{
			name: "no epoch",
			tags: []rpgHeaderTag{
				{tag: rpgTagName, str: "demo"}, {tag: rpgTagVersion, str: "1.2"}, {tag: rpgTagRelease, str: "3"},
				{tag: rpgTagArch, str: "x86_64"}, {tag: rpgTagSummary, str: "A demo package"},
			},
			wantVersion: "1.2-3.x86_64",
			wantPath:    "demo/1.2-3/x86_64/demo-1.2-3.x86_64.rpm",
		},
		{
			name: "with epoch",
			tags: []rpgHeaderTag{
				{tag: rpgTagName, str: "demo"}, {tag: rpgTagVersion, str: "1.2"}, {tag: rpgTagRelease, str: "3"},
				{tag: rpgTagArch, str: "noarch"}, {tag: rpgTagEpoch, number: 2},
			},
			wantVersion: "2:1.2-3.noarch",
			wantPath:    "demo/1.2-3/noarch/2/demo-1.2-3.noarch.rpm",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := rpgBuildRPM(tt.tags...)
			localBase := &rpgLocalBase{existent: true}
			helper := NewRegistryHelper(localBase, &rpgFileManager{content: content, t: t}, nil)

			headers, sha, err := helper.UploadPackage(ctx, rpgInfo(), bytes.NewReader(content))
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, headers.Code)
			require.Equal(t, "sha", sha)
			require.Equal(t, tt.wantPath, localBase.gotPath)
			require.Equal(t, tt.wantVersion, localBase.gotVer)
			require.NotNil(t, localBase.gotMeta)
		})
	}

	t.Run("artifact store failure", func(t *testing.T) {
		content := rpgBuildRPM(
			rpgHeaderTag{tag: rpgTagName, str: "demo"}, rpgHeaderTag{tag: rpgTagVersion, str: "1"},
			rpgHeaderTag{tag: rpgTagRelease, str: "1"}, rpgHeaderTag{tag: rpgTagArch, str: "x86_64"},
		)
		localBase := &rpgLocalBase{err: io.ErrUnexpectedEOF}
		helper := NewRegistryHelper(localBase, &rpgFileManager{content: content, t: t}, nil)
		_, _, err := helper.UploadPackage(ctx, rpgInfo(), bytes.NewReader(content))
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})
}

func TestRpgWireProviders(t *testing.T) {
	localBase := &rpgLocalBase{}
	fm := &rpgFileManager{}

	helper := RegistryHelperProvider(localBase, fm, nil)
	require.NotNil(t, helper)

	local := LocalRegistryProvider(localBase, fm, nil, nil, nil, nil, nil, nil, helper)
	require.NotNil(t, local)

	proxy := ProxyProvider(nil, nil, nil, nil, fm, nil, nil, localBase, helper, refcache.SpaceFinder{}, nil)
	require.NotNil(t, proxy)
}
