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
	"encoding/xml"

	rpmmetadata "github.com/harness/gitness/registry/app/metadata/rpm"
)

type primaryVersion struct {
	Epoch   string `xml:"epoch,attr"`
	Version string `xml:"ver,attr"`
	Release string `xml:"rel,attr"`
}

type primaryChecksum struct {
	Checksum string `xml:",chardata"` //nolint: tagliatelle
	Type     string `xml:"type,attr"`
	Pkgid    string `xml:"pkgid,attr"`
}

type primaryTimes struct {
	File  uint64 `xml:"file,attr"`
	Build uint64 `xml:"build,attr"`
}

type primarySizes struct {
	Package   int64  `xml:"package,attr"`
	Installed uint64 `xml:"installed,attr"`
	Archive   uint64 `xml:"archive,attr"`
}

type PrimaryLocation struct {
	Href string `xml:"href,attr"`
}

type primaryEntryList struct {
	Entries []*rpmmetadata.Entry `xml:"rpm:entry"`
}

type primaryFormat struct {
	License   string              `xml:"rpm:license"`
	Vendor    string              `xml:"rpm:vendor"`
	Group     string              `xml:"rpm:group"`
	Buildhost string              `xml:"rpm:buildhost"`
	Sourcerpm string              `xml:"rpm:sourcerpm"`
	Provides  primaryEntryList    `xml:"rpm:provides"`
	Requires  primaryEntryList    `xml:"rpm:requires"`
	Conflicts primaryEntryList    `xml:"rpm:conflicts"`
	Obsoletes primaryEntryList    `xml:"rpm:obsoletes"`
	Files     []*rpmmetadata.File `xml:"file"`
}

type primaryPackage struct {
	XMLName      xml.Name        `xml:"package"`
	Type         string          `xml:"type,attr"`
	Name         string          `xml:"name"`
	Architecture string          `xml:"arch"`
	Version      primaryVersion  `xml:"version"`
	Checksum     primaryChecksum `xml:"checksum"`
	Summary      string          `xml:"summary"`
	Description  string          `xml:"description"`
	Packager     string          `xml:"packager"`
	URL          string          `xml:"url"`
	Time         primaryTimes    `xml:"time"`
	Size         primarySizes    `xml:"size"`
	Location     PrimaryLocation `xml:"location"`
	Format       primaryFormat   `xml:"format"`
}

type primaryMetadata struct {
	XMLName      xml.Name          `xml:"metadata"`
	Xmlns        string            `xml:"xmlns,attr"`
	XmlnsRpm     string            `xml:"xmlns:rpm,attr"`
	PackageCount int               `xml:"packages,attr"`
	Packages     []*primaryPackage `xml:"package"`
}

type otherVersion struct {
	Epoch   string `xml:"epoch,attr"`
	Version string `xml:"ver,attr"`
	Release string `xml:"rel,attr"`
}

type otherPackage struct {
	Pkgid        string                   `xml:"pkgid,attr"`
	Name         string                   `xml:"name,attr"`
	Architecture string                   `xml:"arch,attr"`
	Version      otherVersion             `xml:"version"`
	Changelogs   []*rpmmetadata.Changelog `xml:"changelog"`
}

type otherdata struct {
	XMLName      xml.Name        `xml:"otherdata"`
	Xmlns        string          `xml:"xmlns,attr"`
	PackageCount int             `xml:"packages,attr"`
	Packages     []*otherPackage `xml:"package"`
}

type fileListVersion struct {
	Epoch   string `xml:"epoch,attr"`
	Version string `xml:"ver,attr"`
	Release string `xml:"rel,attr"`
}

type fileListPackage struct {
	Pkgid        string              `xml:"pkgid,attr"`
	Name         string              `xml:"name,attr"`
	Architecture string              `xml:"arch,attr"`
	Version      fileListVersion     `xml:"version"`
	Files        []*rpmmetadata.File `xml:"file"`
}

type filelists struct {
	XMLName      xml.Name           `xml:"filelists"`
	Xmlns        string             `xml:"xmlns,attr"`
	PackageCount int                `xml:"packages,attr"`
	Packages     []*fileListPackage `xml:"package"`
}

type repomd struct {
	XMLName  xml.Name    `xml:"repomd"`
	Xmlns    string      `xml:"xmlns,attr"`
	XmlnsRpm string      `xml:"xmlns:rpm,attr"`
	Data     []*repoData `xml:"data"`
}

type repoChecksum struct {
	Value string `xml:",chardata"` //nolint: tagliatelle
	Type  string `xml:"type,attr"`
}

type repoLocation struct {
	Href string `xml:"href,attr"`
}

type repoData struct {
	Type         string       `xml:"type,attr"`
	Checksum     repoChecksum `xml:"checksum"`
	OpenChecksum repoChecksum `xml:"open-checksum"` //nolint: tagliatelle
	Location     repoLocation `xml:"location"`
	Timestamp    int64        `xml:"timestamp"`
	Size         int64        `xml:"size"`
	OpenSize     int64        `xml:"open-size"` //nolint: tagliatelle
}

type packageInfo struct {
	Name            string
	Sha256          string
	Size            int64
	VersionMetadata *rpmmetadata.VersionMetadata
	FileMetadata    *rpmmetadata.FileMetadata
}

type Package struct {
	Name            string
	Version         string
	VersionMetadata *rpmmetadata.VersionMetadata
	FileMetadata    *rpmmetadata.FileMetadata
}
