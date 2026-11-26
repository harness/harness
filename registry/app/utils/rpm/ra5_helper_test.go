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
	"encoding/binary"
	"testing"

	"github.com/sassoftware/go-rpmutils"
	"github.com/stretchr/testify/require"
)

func TestRa5ParsePackageInvalid(t *testing.T) {
	_, err := ParsePackage(bytes.NewReader([]byte("not an rpm")))
	require.Error(t, err)

	_, err = ParsePackage(bytes.NewReader(nil))
	require.Error(t, err)
}

type ra5Tag struct {
	tag, typ, count int32
	data            []byte
}

func ra5Strs(tag int, typ int32, vals ...string) ra5Tag {
	var b bytes.Buffer
	for _, v := range vals {
		b.WriteString(v)
		b.WriteByte(0)
	}
	return ra5Tag{int32(tag), typ, int32(len(vals)), b.Bytes()}
}

func ra5Str(tag int, val string) ra5Tag {
	return ra5Strs(tag, rpmutils.RPM_STRING_TYPE, val)
}

func ra5Arr(tag int, vals ...string) ra5Tag {
	return ra5Strs(tag, rpmutils.RPM_STRING_ARRAY_TYPE, vals...)
}

func ra5Ints(tag int, vals ...uint32) ra5Tag {
	b := make([]byte, 0, 4*len(vals))
	for _, v := range vals {
		b = binary.BigEndian.AppendUint32(b, v)
	}
	return ra5Tag{int32(tag), rpmutils.RPM_INT32_TYPE, int32(len(vals)), b}
}

func ra5Shorts(tag int, vals ...uint16) ra5Tag {
	b := make([]byte, 0, 2*len(vals))
	for _, v := range vals {
		b = binary.BigEndian.AppendUint16(b, v)
	}
	return ra5Tag{int32(tag), rpmutils.RPM_INT16_TYPE, int32(len(vals)), b}
}

func ra5Header(tags []ra5Tag, pad8 bool) []byte {
	var table, store bytes.Buffer
	for _, tg := range tags {
		align := 1
		switch tg.typ {
		case rpmutils.RPM_INT16_TYPE:
			align = 2
		case rpmutils.RPM_INT32_TYPE:
			align = 4
		}
		for store.Len()%align != 0 {
			store.WriteByte(0)
		}
		for _, v := range []int32{tg.tag, tg.typ, int32(store.Len()), tg.count} { //nolint:gosec // test data.
			table.Write(binary.BigEndian.AppendUint32(nil, uint32(v))) //nolint:gosec // test data.
		}
		store.Write(tg.data)
	}
	var out bytes.Buffer
	out.Write([]byte{0x8e, 0xad, 0xe8, 0x01, 0, 0, 0, 0})
	out.Write(binary.BigEndian.AppendUint32(nil, uint32(len(tags))))   //nolint:gosec // test data.
	out.Write(binary.BigEndian.AppendUint32(nil, uint32(store.Len()))) //nolint:gosec // test data.
	out.Write(table.Bytes())
	out.Write(store.Bytes())
	for pad8 && out.Len()%8 != 0 {
		out.WriteByte(0)
	}
	return out.Bytes()
}

func ra5BuildRPM(isSource bool, tags ...ra5Tag) []byte {
	lead := make([]byte, 96)
	copy(lead, []byte{0xed, 0xab, 0xee, 0xdb, 3, 0})
	if isSource {
		lead[7] = 1
	}
	out := make([]byte, 0, len(lead)+len(tags)*32)
	out = append(out, lead...)
	out = append(out, ra5Header(nil, true)...)
	return append(out, ra5Header(tags, false)...)
}

func ra5NevraTags(name string) []ra5Tag {
	return []ra5Tag{
		ra5Str(rpmutils.NAME, name),
		ra5Str(rpmutils.VERSION, "1.2"),
		ra5Str(rpmutils.RELEASE, "3"),
		ra5Ints(rpmutils.EPOCH, 2),
		ra5Str(rpmutils.ARCH, "x86_64"),
	}
}

func TestRa5ParseSyntheticPackage(t *testing.T) {
	const (
		ge = rpmutils.RPMSENSE_GREATER | rpmutils.RPMSENSE_EQUAL
		le = rpmutils.RPMSENSE_LESS | rpmutils.RPMSENSE_EQUAL
	)
	tags := append(ra5NevraTags("full"),
		ra5Str(rpmutils.SUMMARY, "a summary"),
		ra5Str(rpmutils.DESCRIPTION, "a description"),
		ra5Str(rpmutils.LICENSE, "Apache-2.0"),
		ra5Str(rpmutils.URL, "https://example.com/full"),
		ra5Str(rpmutils.VENDOR, "acme"),
		ra5Str(rpmutils.GROUP, "tools"),
		ra5Str(rpmutils.PACKAGER, "dev"),
		ra5Str(rpmutils.SOURCERPM, "full-1.2-3.src.rpm"),
		ra5Str(rpmutils.BUILDHOST, "builder"),
		ra5Ints(rpmutils.BUILDTIME, 111),
		ra5Ints(rpmutils.FILEMTIMES, 222),
		ra5Ints(rpmutils.SIZE, 333),
		ra5Arr(rpmutils.PROVIDENAME, "p-ge", "p-le", "p-gt", "p-lt", "p-eq", "p-none"),
		ra5Arr(rpmutils.PROVIDEVERSION, "2:1.0-5", "1.0", "3.0", "4.0-1", "5.0", ""),
		ra5Ints(rpmutils.PROVIDEFLAGS, ge, le, rpmutils.RPMSENSE_GREATER, rpmutils.RPMSENSE_LESS,
			rpmutils.RPMSENSE_EQUAL, 0),
		ra5Arr(rpmutils.REQUIRENAME, "r1", "r2"),
		ra5Arr(rpmutils.REQUIREVERSION, "1.0"),
		ra5Ints(rpmutils.REQUIREFLAGS, ge, ge),
		ra5Arr(rpmutils.CONFLICTNAME, "c1"),
		ra5Arr(rpmutils.CONFLICTVERSION, "1.0"),
		ra5Ints(rpmutils.CONFLICTFLAGS, le),
		ra5Arr(rpmutils.OBSOLETENAME, "o1"),
		ra5Arr(rpmutils.OBSOLETEVERSION, "1.0"),
		ra5Ints(rpmutils.OBSOLETEFLAGS, le),
		ra5Arr(rpmutils.BASENAMES, "tool", "tool.conf", "tool.log", "share", "orphan", "nodir"),
		ra5Arr(rpmutils.DIRNAMES, "/usr/bin/", "/etc/"),
		ra5Ints(rpmutils.DIRINDEXES, 0, 1, 1, 0, 9),
		ra5Ints(rpmutils.FILEFLAGS, 0, 0, rpmutils.RPMFILE_GHOST, 0),
		ra5Shorts(rpmutils.FILEMODES, 0o100755, 0o100644, 0o100644, 0o40755),
		ra5Arr(rpmutils.CHANGELOGTEXT, "- first", "- second"),
		ra5Arr(rpmutils.CHANGELOGNAME, "dev one", "dev two"),
		ra5Ints(rpmutils.CHANGELOGTIME, 1000, 2000),
	)
	p, err := ParsePackage(bytes.NewReader(ra5BuildRPM(false, tags...)))
	require.NoError(t, err)

	require.Equal(t, "full", p.Name)
	require.Equal(t, "1.2-3", p.Version)
	require.Equal(t, "https://example.com/full", p.VersionMetadata.ProjectURL)
	fm := p.FileMetadata
	require.Equal(t, "2", fm.Epoch)
	require.Equal(t, uint64(111), fm.BuildTime)
	require.Equal(t, uint64(222), fm.FileTime)
	require.Equal(t, uint64(333), fm.InstalledSize)
	require.Equal(t, "acme", fm.Vendor)
	require.Equal(t, "tools", fm.Group)
	require.Equal(t, "dev", fm.Packager)

	flags := make([]string, 0, len(fm.Provides))
	for _, e := range fm.Provides {
		flags = append(flags, e.Flags)
	}
	require.Equal(t, []string{"GE", "LE", "GT", "LT", "EQ", ""}, flags)
	require.Equal(t, "2", fm.Provides[0].Epoch)
	require.Equal(t, "1.0", fm.Provides[0].Version)
	require.Equal(t, "5", fm.Provides[0].Release)
	require.Equal(t, "0", fm.Provides[1].Epoch)
	require.Equal(t, "4.0", fm.Provides[3].Version)
	require.Equal(t, "1", fm.Provides[3].Release)
	require.Empty(t, fm.Provides[5].Version)

	// Mismatched array lengths are ignored.
	require.Nil(t, fm.Requires)
	require.Len(t, fm.Conflicts, 1)
	require.Len(t, fm.Obsoletes, 1)

	require.Len(t, fm.Files, 4)
	require.Equal(t, "/usr/bin/tool", fm.Files[0].Path)
	require.True(t, fm.Files[0].IsExecutable)
	require.False(t, fm.Files[1].IsExecutable)
	require.Equal(t, "ghost", fm.Files[2].Type)
	require.Equal(t, "dir", fm.Files[3].Type)

	require.Len(t, fm.Changelogs, 2)
	require.Equal(t, "dev two", fm.Changelogs[1].Author)
	require.Equal(t, int64(2000), fm.Changelogs[1].Date)
	require.Equal(t, "- second", fm.Changelogs[1].Text)
}

func TestRa5ParseSyntheticVariants(t *testing.T) {
	t.Run("source rpm and invalid url", func(t *testing.T) {
		tags := append(ra5NevraTags("srcpkg"), ra5Str(rpmutils.URL, "not a url"))
		p, err := ParsePackage(bytes.NewReader(ra5BuildRPM(true, tags...)))
		require.NoError(t, err)
		require.Equal(t, "src", p.FileMetadata.Architecture)
		require.Empty(t, p.VersionMetadata.ProjectURL)
	})

	t.Run("mismatched optional arrays", func(t *testing.T) {
		tags := append(ra5NevraTags("odd"),
			ra5Arr(rpmutils.PROVIDENAME, "a"),
			ra5Ints(rpmutils.PROVIDEFLAGS),
			ra5Arr(rpmutils.REQUIRENAME, "a"),
			ra5Ints(rpmutils.REQUIREFLAGS, 0),
			ra5Arr(rpmutils.CONFLICTNAME, "a"),
			ra5Arr(rpmutils.CONFLICTVERSION, "1"),
			ra5Ints(rpmutils.CONFLICTFLAGS, 0, 0),
			ra5Arr(rpmutils.CHANGELOGTEXT, "t"),
			ra5Arr(rpmutils.CHANGELOGNAME, "n"),
			ra5Ints(rpmutils.CHANGELOGTIME, 1, 2),
		)
		p, err := ParsePackage(bytes.NewReader(ra5BuildRPM(false, tags...)))
		require.NoError(t, err)
		require.Nil(t, p.FileMetadata.Provides)
		require.Nil(t, p.FileMetadata.Requires)
		require.Nil(t, p.FileMetadata.Conflicts)
		require.Nil(t, p.FileMetadata.Changelogs)
	})

	t.Run("changelog missing pieces", func(t *testing.T) {
		for _, tags := range [][]ra5Tag{
			{ra5Arr(rpmutils.CHANGELOGTEXT, "t")},
			{ra5Arr(rpmutils.CHANGELOGTEXT, "t"), ra5Arr(rpmutils.CHANGELOGNAME, "n")},
		} {
			p, err := ParsePackage(bytes.NewReader(ra5BuildRPM(false, append(ra5NevraTags("cl"), tags...)...)))
			require.NoError(t, err)
			require.Nil(t, p.FileMetadata.Changelogs)
		}
	})

	t.Run("missing nevra", func(t *testing.T) {
		_, err := ParsePackage(bytes.NewReader(ra5BuildRPM(false, ra5Str(rpmutils.NAME, "only"))))
		require.Error(t, err)
	})
}
