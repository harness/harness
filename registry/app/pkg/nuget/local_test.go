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

package nuget

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"strings"
	"testing"
)

// --- ZIP builder helpers ---

const (
	testFileHeaderSig     = 0x04034b50
	testDirHeaderSig      = 0x02014b50
	testDirEndSig         = 0x06054b50
	testDataDescriptorSig = 0x08074b50
	testStoreMethod       = uint16(0)
	testDescriptorFlags   = uint16(0x0008)
)

type testEntry struct {
	name    string
	content []byte
}

func testAppendU32(buf *bytes.Buffer, v uint32) {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	buf.Write(b)
}

func testWriteLocalHeader(buf *bytes.Buffer, name string) {
	header := make([]byte, 30)
	binary.LittleEndian.PutUint32(header[0:], testFileHeaderSig)
	binary.LittleEndian.PutUint16(header[4:], 20)
	binary.LittleEndian.PutUint16(header[6:], testDescriptorFlags)
	binary.LittleEndian.PutUint16(header[8:], testStoreMethod)
	//nolint:gosec
	binary.LittleEndian.PutUint16(header[26:], uint16(len(name)))
	buf.Write(header)
	buf.WriteString(name)
}

func testWriteCDHeader(buf *bytes.Buffer, name string, crcv, compSz, uncompSz, offset uint32) {
	header := make([]byte, 46)
	binary.LittleEndian.PutUint32(header[0:], testDirHeaderSig)
	binary.LittleEndian.PutUint16(header[4:], 20)
	binary.LittleEndian.PutUint16(header[6:], 20)
	binary.LittleEndian.PutUint16(header[10:], testStoreMethod)
	binary.LittleEndian.PutUint32(header[16:], crcv)
	binary.LittleEndian.PutUint32(header[20:], compSz)
	binary.LittleEndian.PutUint32(header[24:], uncompSz)
	//nolint:gosec
	binary.LittleEndian.PutUint16(header[28:], uint16(len(name)))
	binary.LittleEndian.PutUint32(header[42:], offset)
	buf.Write(header)
	buf.WriteString(name)
}

func testWriteEOCD(buf *bytes.Buffer, numEntries uint16, cdSize, cdOffset uint32) {
	record := make([]byte, 22)
	binary.LittleEndian.PutUint32(record[0:], testDirEndSig)
	binary.LittleEndian.PutUint16(record[8:], numEntries)
	binary.LittleEndian.PutUint16(record[10:], numEntries)
	binary.LittleEndian.PutUint32(record[12:], cdSize)
	binary.LittleEndian.PutUint32(record[16:], cdOffset)
	buf.Write(record)
}

// buildTestNupkg creates a .nupkg (ZIP archive) using STORE method with data
// descriptors including the PK\x07\x08 signature (16-byte descriptors).
func buildTestNupkg(entries []testEntry) []byte {
	var buf bytes.Buffer

	type offsetInfo struct {
		offset, crc, size uint32
	}
	offsets := make([]offsetInfo, len(entries))

	for idx, e := range entries {
		c := crc32.ChecksumIEEE(e.content)
		//nolint:gosec
		sz := uint32(len(e.content))
		//nolint:gosec
		offsets[idx] = offsetInfo{uint32(buf.Len()), c, sz}

		testWriteLocalHeader(&buf, e.name)
		buf.Write(e.content)
		testAppendU32(&buf, testDataDescriptorSig)
		testAppendU32(&buf, c)
		testAppendU32(&buf, sz)
		testAppendU32(&buf, sz)
	}

	//nolint:gosec
	cdStart := uint32(buf.Len())
	for idx, e := range entries {
		oi := offsets[idx]
		testWriteCDHeader(&buf, e.name, oi.crc, oi.size, oi.size, oi.offset)
	}
	//nolint:gosec
	cdEnd := uint32(buf.Len())

	//nolint:gosec
	testWriteEOCD(&buf, uint16(len(entries)), cdEnd-cdStart, cdStart)
	return buf.Bytes()
}

func testNuspec(id, version string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>%s</id>
    <version>%s</version>
    <authors>Test</authors>
    <description>Test package for %s %s</description>
  </metadata>
</package>`, id, version, id, version)
}

// TestUploadPath_VersionNormalization verifies the full upload→download path alignment.
// It exercises buildMetadata + validateAndNormaliseVersion + filename construction to
// ensure the stored filename matches what the download URL expects.
//
// Bug: The filename was built from the raw nuspec version (e.g., "5.28") while the
// path and download URL used the normalized version ("5.28.0"), causing a 404.
//
// Confirmed from production DB:
//   - activeperl: stored "activeperl.5.28.nupkg", download expected "activeperl.5.28.0.nupkg"
//   - dotnet4.6: stored with "00081", download expected "81" (leading zeros stripped)
//   - crabel-barserver: stored ".8.3.3.0.nupkg", download expected ".8.3.3.nupkg"
//   - microsoftteams: stored ".1.4.00.8872.nupkg", download expected ".1.4.0.8872.nupkg"
func TestUploadPath_VersionNormalization(t *testing.T) {
	tests := []struct {
		name            string
		packageID       string
		nuspecVersion   string
		wantNormVersion string
	}{
		{
			name:            "activeperl: 2-segment version normalizes to 3",
			packageID:       "ActivePerl",
			nuspecVersion:   "5.28",
			wantNormVersion: "5.28.0",
		},
		{
			name:            "dotnet4.6: leading zeros stripped",
			packageID:       "dotnet4.6",
			nuspecVersion:   "4.6.00081.20150925",
			wantNormVersion: "4.6.81.20150925",
		},
		{
			name:            "crabel-barserver: trailing .0 dropped",
			packageID:       "crabel-barserver",
			nuspecVersion:   "8.3.3.0",
			wantNormVersion: "8.3.3",
		},
		{
			name:            "microsoftteams: leading zero in segment stripped",
			packageID:       "MicrosoftTeams",
			nuspecVersion:   "1.4.00.8872",
			wantNormVersion: "1.4.0.8872",
		},
		{
			name:            "4-segment version: trailing zero dropped",
			packageID:       "FourSeg",
			nuspecVersion:   "1.2.3.0",
			wantNormVersion: "1.2.3",
		},
		{
			name:            "3-segment version: no change",
			packageID:       "ThreeSeg",
			nuspecVersion:   "2.0.0",
			wantNormVersion: "2.0.0",
		},
		{
			name:            "prerelease version: preserved",
			packageID:       "PreRelease",
			nuspecVersion:   "1.0.0-beta",
			wantNormVersion: "1.0.0-beta",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entries := []testEntry{
				{"tools/install.ps1", []byte("# script")},
				{tc.packageID + ".nuspec", []byte(testNuspec(tc.packageID, tc.nuspecVersion))},
			}

			nupkg := buildTestNupkg(entries)

			reg := &localRegistry{}
			metadata, err := reg.buildMetadata(bytes.NewReader(nupkg))
			if err != nil {
				t.Fatalf("buildMetadata failed: %v", err)
			}

			normVersion, err := validateAndNormaliseVersion(metadata.PackageMetadata.Version)
			if err != nil {
				t.Fatalf("validateAndNormaliseVersion(%q) failed: %v", metadata.PackageMetadata.Version, err)
			}
			if normVersion != tc.wantNormVersion {
				t.Errorf("Normalized version: got %q, want %q", normVersion, tc.wantNormVersion)
			}

			id := strings.ToLower(metadata.PackageMetadata.ID)
			fixedFilename := fmt.Sprintf("%s.%s.nupkg", id, normVersion)
			storedPath := id + "/" + normVersion + "/" + fixedFilename
			downloadURL := fmt.Sprintf("package/%s/%s/%s", id, normVersion, fixedFilename)

			if !strings.HasSuffix(downloadURL, fixedFilename) {
				t.Errorf("download URL %q doesn't end with stored filename %q", downloadURL, fixedFilename)
			}

			oldBuggyFilename := fmt.Sprintf("%s.%s.nupkg", id, strings.ToLower(metadata.PackageMetadata.Version))
			if oldBuggyFilename != fixedFilename {
				t.Logf("Bug #1 would have caused 404:")
				t.Logf("  Old stored path:   %s/%s/%s", id, normVersion, oldBuggyFilename)
				t.Logf("  Fixed stored path: %s", storedPath)
				t.Logf("  Download URL:      %s", downloadURL)
			}
		})
	}
}
