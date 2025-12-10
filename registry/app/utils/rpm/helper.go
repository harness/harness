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

//nolint:gosec
import (
	"fmt"
	"io"
	"reflect"
	"strings"

	rpmmetadata "github.com/harness/gitness/registry/app/metadata/rpm"
	rpmtypes "github.com/harness/gitness/registry/app/utils/rpm/types"
	"github.com/harness/gitness/registry/validation"

	"github.com/sassoftware/go-rpmutils"
)

const (
	sIFMT  = 0xf000
	sIFDIR = 0x4000
	sIXUSR = 0x40
	sIXGRP = 0x8
	sIXOTH = 0x1

	RepoDataPrefix = "repodata/"
)

func ParsePackage(r io.Reader) (*rpmtypes.Package, error) {
	rpm, err := rpmutils.ReadRpm(r)
	if err != nil {
		return nil, err
	}

	nevra, err := rpm.Header.GetNEVRA()
	if err != nil {
		return nil, err
	}

	isSourceRpm := false
	if rpm.Header != nil {
		if headerValue := reflect.ValueOf(rpm.Header).Elem(); headerValue.Kind() == reflect.Struct {
			isSourceField := headerValue.FieldByName("isSource")
			if isSourceField.IsValid() && isSourceField.Kind() == reflect.Bool {
				isSourceRpm = isSourceField.Bool()
			}
		}
	}

	if isSourceRpm {
		nevra.Arch = "src"
	}

	version := fmt.Sprintf("%s-%s", nevra.Version, nevra.Release)

	p := &rpmtypes.Package{
		Name:    nevra.Name,
		Version: version,
		VersionMetadata: &rpmmetadata.VersionMetadata{
			Summary:     getString(rpm.Header, rpmutils.SUMMARY),
			Description: getString(rpm.Header, rpmutils.DESCRIPTION),
			License:     getString(rpm.Header, rpmutils.LICENSE),
			ProjectURL:  getString(rpm.Header, rpmutils.URL),
		},
		FileMetadata: &rpmmetadata.FileMetadata{
			Architecture:  nevra.Arch,
			Epoch:         nevra.Epoch,
			Version:       nevra.Version,
			Release:       nevra.Release,
			Vendor:        getString(rpm.Header, rpmutils.VENDOR),
			Group:         getString(rpm.Header, rpmutils.GROUP),
			Packager:      getString(rpm.Header, rpmutils.PACKAGER),
			SourceRpm:     getString(rpm.Header, rpmutils.SOURCERPM),
			BuildHost:     getString(rpm.Header, rpmutils.BUILDHOST),
			BuildTime:     getUInt64(rpm.Header, rpmutils.BUILDTIME),
			FileTime:      getUInt64(rpm.Header, rpmutils.FILEMTIMES),
			InstalledSize: getUInt64(rpm.Header, rpmutils.SIZE),
			ArchiveSize:   getUInt64(rpm.Header, rpmutils.SIG_PAYLOADSIZE),

			Provides:   getEntries(rpm.Header, rpmutils.PROVIDENAME, rpmutils.PROVIDEVERSION, rpmutils.PROVIDEFLAGS),
			Requires:   getEntries(rpm.Header, rpmutils.REQUIRENAME, rpmutils.REQUIREVERSION, rpmutils.REQUIREFLAGS),
			Conflicts:  getEntries(rpm.Header, rpmutils.CONFLICTNAME, rpmutils.CONFLICTVERSION, rpmutils.CONFLICTFLAGS),
			Obsoletes:  getEntries(rpm.Header, rpmutils.OBSOLETENAME, rpmutils.OBSOLETEVERSION, rpmutils.OBSOLETEFLAGS),
			Files:      getFiles(rpm.Header),
			Changelogs: getChangelogs(rpm.Header),
		},
	}

	if !validation.IsValidURL(p.VersionMetadata.ProjectURL) {
		p.VersionMetadata.ProjectURL = ""
	}

	return p, nil
}

func getString(h *rpmutils.RpmHeader, tag int) string {
	values, err := h.GetStrings(tag)
	if err != nil || len(values) < 1 {
		return ""
	}
	return values[0]
}

func getUInt64(h *rpmutils.RpmHeader, tag int) uint64 {
	values, err := h.GetUint64s(tag)
	if err != nil || len(values) < 1 {
		return 0
	}
	return values[0]
}

// nolint: gocritic
func getEntries(h *rpmutils.RpmHeader, namesTag, versionsTag, flagsTag int) []*rpmmetadata.Entry {
	names, err := h.GetStrings(namesTag)
	if err != nil || len(names) == 0 {
		return nil
	}
	flags, err := h.GetUint64s(flagsTag)
	if err != nil || len(flags) == 0 {
		return nil
	}
	versions, err := h.GetStrings(versionsTag)
	if err != nil || len(versions) == 0 {
		return nil
	}
	if len(names) != len(flags) || len(names) != len(versions) {
		return nil
	}

	entries := make([]*rpmmetadata.Entry, 0, len(names))
	for i := range names {
		e := &rpmmetadata.Entry{
			Name: names[i],
		}

		flags := flags[i]
		if (flags&rpmutils.RPMSENSE_GREATER) != 0 && (flags&rpmutils.RPMSENSE_EQUAL) != 0 {
			e.Flags = "GE"
		} else if (flags&rpmutils.RPMSENSE_LESS) != 0 && (flags&rpmutils.RPMSENSE_EQUAL) != 0 {
			e.Flags = "LE"
		} else if (flags & rpmutils.RPMSENSE_GREATER) != 0 {
			e.Flags = "GT"
		} else if (flags & rpmutils.RPMSENSE_LESS) != 0 {
			e.Flags = "LT"
		} else if (flags & rpmutils.RPMSENSE_EQUAL) != 0 {
			e.Flags = "EQ"
		}

		version := versions[i]
		if version != "" {
			parts := strings.Split(version, "-")

			versionParts := strings.Split(parts[0], ":")
			if len(versionParts) == 2 {
				e.Version = versionParts[1]
				e.Epoch = versionParts[0]
			} else {
				e.Version = versionParts[0]
				e.Epoch = "0"
			}

			if len(parts) > 1 {
				e.Release = parts[1]
			}
		}

		entries = append(entries, e)
	}
	return entries
}

func getFiles(h *rpmutils.RpmHeader) []*rpmmetadata.File {
	baseNames, _ := h.GetStrings(rpmutils.BASENAMES)
	dirNames, _ := h.GetStrings(rpmutils.DIRNAMES)
	dirIndexes, _ := h.GetUint32s(rpmutils.DIRINDEXES)
	fileFlags, _ := h.GetUint32s(rpmutils.FILEFLAGS)
	fileModes, _ := h.GetUint32s(rpmutils.FILEMODES)

	files := make([]*rpmmetadata.File, 0, len(baseNames))
	for i := range baseNames {
		if len(dirIndexes) <= i {
			continue
		}
		dirIndex := dirIndexes[i]
		if len(dirNames) <= int(dirIndex) {
			continue
		}

		var fileType string
		var isExecutable bool
		if i < len(fileFlags) && (fileFlags[i]&rpmutils.RPMFILE_GHOST) != 0 {
			fileType = "ghost"
		} else if i < len(fileModes) {
			if (fileModes[i] & sIFMT) == sIFDIR {
				fileType = "dir"
			} else {
				mode := fileModes[i] & ^uint32(sIFMT)
				isExecutable = (mode&sIXUSR) != 0 || (mode&sIXGRP) != 0 || (mode&sIXOTH) != 0
			}
		}

		files = append(files, &rpmmetadata.File{
			Path:         dirNames[dirIndex] + baseNames[i],
			Type:         fileType,
			IsExecutable: isExecutable,
		})
	}

	return files
}

func getChangelogs(h *rpmutils.RpmHeader) []*rpmmetadata.Changelog {
	texts, err := h.GetStrings(rpmutils.CHANGELOGTEXT)
	if err != nil || len(texts) == 0 {
		return nil
	}
	authors, err := h.GetStrings(rpmutils.CHANGELOGNAME)
	if err != nil || len(authors) == 0 {
		return nil
	}
	times, err := h.GetUint32s(rpmutils.CHANGELOGTIME)
	if err != nil || len(times) == 0 {
		return nil
	}
	if len(texts) != len(authors) || len(texts) != len(times) {
		return nil
	}

	changelogs := make([]*rpmmetadata.Changelog, 0, len(texts))
	for i := range texts {
		changelogs = append(changelogs, &rpmmetadata.Changelog{
			Author: authors[i],
			Date:   int64(times[i]),
			Text:   texts[i],
		})
	}
	return changelogs
}
