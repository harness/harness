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
	"encoding/xml"
	"time"

	"github.com/harness/gitness/registry/app/metadata/nuget"
	"github.com/harness/gitness/registry/app/pkg"
)

type ArtifactInfo struct {
	pkg.ArtifactInfo
	Version       string
	Filename      string
	ProxyEndpoint string
	Metadata      nuget.Metadata
}

// BaseArtifactInfo implements pkg.PackageArtifactInfo interface.
func (a ArtifactInfo) BaseArtifactInfo() pkg.ArtifactInfo {
	return a.ArtifactInfo
}

func (a ArtifactInfo) GetImageVersion() (exists bool, imageVersion string) {
	if a.Image != "" && a.Version != "" {
		return true, pkg.JoinWithSeparator(":", a.Image, a.Version)
	}
	return false, ""
}

func (a ArtifactInfo) GetVersion() string {
	return a.Version
}

type File struct {
	FileURL string
	Name    string
}

type PackageMetadata struct {
	Name  string
	Files []File
}

type ServiceEndpoint struct {
	Version   string     `json:"version"`
	Resources []Resource `json:"resources"`
}

type Resource struct {
	//nolint:revive
	ID   string `json:"@id"`
	Type string `json:"@type"`
}

type AtomTitle struct {
	Type string `xml:"type,attr"`
	Text string `xml:",chardata"`
}

type ServiceCollection struct {
	Href  string    `xml:"href,attr"`
	Title AtomTitle `xml:"atom:title"`
}

type ServiceWorkspace struct {
	Title      AtomTitle         `xml:"atom:title"`
	Collection ServiceCollection `xml:"collection"`
}

type ServiceEndpointV2 struct {
	XMLName   xml.Name         `xml:"service"`
	Base      string           `xml:"base,attr"`
	Xmlns     string           `xml:"xmlns,attr"`
	XmlnsAtom string           `xml:"xmlns:atom,attr"`
	Workspace ServiceWorkspace `xml:"workspace"`
}

type PackageVersion struct {
	Versions []string `json:"versions"`
}

type RegistrationResponse interface {
	isRegistrationResponse()
}

// https://docs.microsoft.com/en-us/nuget/api/registration-base-url-resource#response
type RegistrationIndexResponse struct {
	RegistrationIndexURL string                           `json:"@id"`
	Count                int                              `json:"count"`
	Pages                []*RegistrationIndexPageResponse `json:"items"`
}

func (r RegistrationIndexResponse) isRegistrationResponse() {

}

// https://docs.microsoft.com/en-us/nuget/api/registration-base-url-resource#registration-page-object
type RegistrationIndexPageResponse struct {
	RegistrationPageURL string                       `json:"@id"`
	Lower               string                       `json:"lower"`
	Upper               string                       `json:"upper"`
	Count               int                          `json:"count"`
	Items               []*RegistrationIndexPageItem `json:"items,omitempty"`
}

// https://docs.microsoft.com/en-us/nuget/api/registration-base-url-resource#registration-leaf-object-in-a-page
type RegistrationIndexPageItem struct {
	RegistrationLeafURL string `json:"@id"`
	//nolint: tagliatelle
	PackageContentURL string `json:"packageContent"`
	//nolint: tagliatelle
	CatalogEntry *CatalogEntry `json:"catalogEntry"`
}

func (r RegistrationIndexPageResponse) isRegistrationResponse() {
}

// https://docs.microsoft.com/en-us/nuget/api/registration-base-url-resource#registration-leaf
type RegistrationLeafResponse struct {
	RegistrationLeafURL string   `json:"@id"`
	Type                []string `json:"@type"`
	Listed              bool     `json:"listed"`
	//nolint: tagliatelle
	PackageContentURL    string    `json:"packageContent"`
	Published            time.Time `json:"published"`
	RegistrationIndexURL string    `json:"registration"`
}

// https://github.com/NuGet/NuGet.Client/blob/dev/src/NuGet.Core/NuGet.Protocol/LegacyFeed/V2FeedQueryBuilder.cs
type FeedEntryResponse struct {
	XMLName         xml.Name             `xml:"entry"`
	Xmlns           string               `xml:"xmlns,attr,omitempty"`
	XmlnsD          string               `xml:"xmlns:d,attr,omitempty"`
	XmlnsM          string               `xml:"xmlns:m,attr,omitempty"`
	Base            string               `xml:"xml:base,attr,omitempty"`
	ID              string               `xml:"id,omitempty"`
	Category        FeedEntryCategory    `xml:"category"`
	Title           TypedValue[string]   `xml:"title"`
	Updated         time.Time            `xml:"updated"`
	Author          string               `xml:"author>name"`
	Summary         string               `xml:"summary"`
	Content         string               `xml:",innerxml"`
	Properties      *FeedEntryProperties `xml:"m:properties"`
	DownloadContent *FeedEntryContent    `xml:"content,omitempty"`
}

type FeedResponse struct {
	XMLName xml.Name             `xml:"feed"`
	Xmlns   string               `xml:"xmlns,attr,omitempty"`
	XmlnsD  string               `xml:"xmlns:d,attr,omitempty"`
	XmlnsM  string               `xml:"xmlns:m,attr,omitempty"`
	Base    string               `xml:"xml:base,attr,omitempty"`
	ID      string               `xml:"id"`
	Title   TypedValue[string]   `xml:"title"`
	Updated time.Time            `xml:"updated"`
	Links   []FeedEntryLink      `xml:"link"`
	Entries []*FeedEntryResponse `xml:"entry"`
	Count   int64                `xml:"m:count"`
}

type FeedEntryCategory struct {
	Term   string `xml:"term,attr"`
	Scheme string `xml:"scheme,attr"`
}

type FeedEntryContent struct {
	Type   string `xml:"type,attr,omitempty"`
	Source string `xml:"src,attr,omitempty"`
}

type FeedEntryLink struct {
	Rel  string       `xml:"rel,attr"`
	Href xml.CharData `xml:"href,attr"`
}

type TypedValue[T any] struct {
	Type  string `xml:"m:type,attr,omitempty"`
	Value T      `xml:",chardata"`
}

type FeedEntryProperties struct {
	Id                       string                `xml:"d:Id"`
	Version                  string                `xml:"d:Version"`
	NormalizedVersion        string                `xml:"d:NormalizedVersion"`
	Authors                  string                `xml:"d:Authors"`
	Dependencies             string                `xml:"d:Dependencies"`
	Description              string                `xml:"d:Description"`
	VersionDownloadCount     TypedValue[int64]     `xml:"d:VersionDownloadCount"`
	DownloadCount            TypedValue[int64]     `xml:"d:DownloadCount"`
	PackageSize              TypedValue[int64]     `xml:"d:PackageSize"`
	Created                  TypedValue[time.Time] `xml:"d:Created"`
	LastUpdated              TypedValue[time.Time] `xml:"d:LastUpdated"`
	Published                TypedValue[time.Time] `xml:"d:Published"`
	ProjectURL               string                `xml:"d:ProjectUrl,omitempty"`
	ReleaseNotes             string                `xml:"d:ReleaseNotes,omitempty"`
	RequireLicenseAcceptance TypedValue[bool]      `xml:"d:RequireLicenseAcceptance"`
	Title                    string                `xml:"d:Title"`
}

// https://docs.microsoft.com/en-us/nuget/api/registration-base-url-resource#catalog-entry
type CatalogEntry struct {
	CatalogLeafURL string `json:"@id"`
	//nolint: tagliatelle
	PackageContentURL string `json:"packageContent"`
	ID                string `json:"id"`
	Version           string `json:"version"`
	Description       string `json:"description"`
	//nolint: tagliatelle
	ReleaseNotes string `json:"releaseNotes"`
	Authors      string `json:"authors"`
	//nolint: tagliatelle
	RequireLicenseAcceptance bool `json:"requireLicenseAcceptance"`
	//nolint: tagliatelle
	ProjectURL string `json:"projectURL"`
	//nolint: tagliatelle
	DependencyGroups []*PackageDependencyGroup `json:"dependencyGroups,omitempty"`
}

// https://docs.microsoft.com/en-us/nuget/api/registration-base-url-resource#package-dependency-group
type PackageDependencyGroup struct {
	//nolint: tagliatelle
	TargetFramework string               `json:"targetFramework"`
	Dependencies    []*PackageDependency `json:"dependencies"`
}

// https://docs.microsoft.com/en-us/nuget/api/registration-base-url-resource#package-dependency
type PackageDependency struct {
	ID    string `json:"id"`
	Range string `json:"range"`
}
