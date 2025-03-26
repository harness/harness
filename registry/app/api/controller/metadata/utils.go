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

package metadata

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	a "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg/commons"

	"github.com/inhies/go-bytesize"
	"github.com/rs/zerolog/log"
)

var registrySort = []string{
	"identifier",
	"lastModified",
	"registrySize",
	"artifactsCount",
	"downloadsCount",
	"type",
}

const (
	RepositoryResource         = "repository"
	ArtifactResource           = "artifact"
	ArtifactVersionResource    = "artifactversion"
	ArtifactFilesResource      = "artifactFiles"
	RegistryIdentifierErrorMsg = "registry name should be 1~255 characters long with lower case characters, numbers " +
		"and ._- and must be start with numbers or characters"
	RegexIdentifierPattern    = "^[a-z0-9]+(?:[._-][a-z0-9]+)*$"
	internalWebhookIdentifier = "harnesstriggerwebhok"
)

var RegistrySortMap = map[string]string{
	"identifier":     "name",
	"lastModified":   "updated_at",
	"registrySize":   "size",
	"artifactsCount": "artifact_count",
	"downloadsCount": "download_count",
	"createdAt":      "created_at",
	"type":           "type",
}

var artifactSort = []string{
	"repoKey",
	"name",
	"lastModified",
	"downloadsCount",
}

var artifactSortMap = map[string]string{
	"repoKey":        "name",
	"lastModified":   "updated_at",
	"name":           "image_name",
	"downloadsCount": "download_count",
	"createdAt":      "created_at",
}

var artifactVersionSort = []string{
	"name",
	"size",
	"pullCommand",
	"downloadsCount",
	"lastModified",
}

var artifactFilesSort = []string{
	"name",
	"size",
	"createdAt",
}

var artifactVersionSortMap = map[string]string{
	"name":           "name",
	"size":           "name",
	"pullCommand":    "name",
	"downloadsCount": "download_count",
	"lastModified":   "updated_at",
	"createdAt":      "created_at",
}

var artifactFilesSortMap = map[string]string{
	"name":      "name",
	"size":      "size",
	"createdAt": "created_at",
}

var validRepositoryTypes = []string{
	string(a.RegistryTypeUPSTREAM),
	string(a.RegistryTypeVIRTUAL),
}

var validPackageTypes = []string{
	string(a.PackageTypeDOCKER),
	string(a.PackageTypeHELM),
	string(a.PackageTypeGENERIC),
	string(a.PackageTypeMAVEN),
	string(a.PackageTypePYTHON),
}

var validUpstreamSources = []string{
	string(a.UpstreamConfigSourceCustom),
	string(a.UpstreamConfigSourceDockerhub),
	string(a.UpstreamConfigSourceAwsEcr),
	string(a.UpstreamConfigSourceMavenCentral),
	string(a.UpstreamConfigSourcePyPi),
}

func ValidatePackageTypes(packageTypes []string) error {
	if commons.IsEmpty(packageTypes) || IsPackageTypesValid(packageTypes) {
		return nil
	}
	return errors.New("invalid package type")
}

func ValidatePackageType(packageType string) error {
	if len(packageType) == 0 || IsPackageTypeValid(packageType) {
		return nil
	}
	return errors.New("invalid package type")
}

func ValidatePackageTypeChange(fromDB, newPackage string) error {
	if len(fromDB) > 0 && len(newPackage) > 0 && fromDB == newPackage {
		return nil
	}
	return errors.New("package type change is not allowed")
}

func ValidateRepoTypeChange(fromDB, newRepo string) error {
	if len(fromDB) > 0 && len(newRepo) > 0 && fromDB == newRepo {
		return nil
	}
	return errors.New("registry type change is not allowed")
}

func ValidateIdentifierChange(fromDB, newIdentifier string) error {
	if len(fromDB) > 0 && len(newIdentifier) > 0 && fromDB == newIdentifier {
		return nil
	}
	return errors.New("registry identifier change is not allowed")
}

func ValidateIdentifier(identifier string) error {
	if len(identifier) == 0 {
		return errors.New(RegistryIdentifierErrorMsg)
	}

	matched, err := regexp.MatchString(RegexIdentifierPattern, identifier)
	if err != nil || !matched {
		return errors.New(RegistryIdentifierErrorMsg)
	}
	return nil
}

func ValidateUpstream(config *a.RegistryConfig) error {
	upstreamConfig, err := config.AsUpstreamConfig()
	if err != nil {
		return err
	}
	if !commons.IsEmpty(config.Type) && config.Type == a.RegistryTypeUPSTREAM &&
		*upstreamConfig.Source != a.UpstreamConfigSourceDockerhub &&
		*upstreamConfig.Source != a.UpstreamConfigSourceMavenCentral &&
		*upstreamConfig.Source != a.UpstreamConfigSourcePyPi {
		if commons.IsEmpty(upstreamConfig.Url) {
			return errors.New("URL is required for upstream repository")
		}
	}
	return nil
}

func ValidateRepoType(repoType string) error {
	if len(repoType) == 0 || IsRepoTypeValid(repoType) {
		return nil
	}
	return errors.New("invalid repository type")
}

func ValidateUpstreamSource(source string) error {
	if len(source) == 0 || IsUpstreamSourceValid(source) {
		return nil
	}
	return errors.New("invalid upstream proxy source")
}

func IsRepoTypeValid(repoType string) bool {
	for _, item := range validRepositoryTypes {
		if item == repoType {
			return true
		}
	}
	return false
}

func IsUpstreamSourceValid(source string) bool {
	for _, item := range validUpstreamSources {
		if item == source {
			return true
		}
	}
	return false
}

func IsPackageTypeValid(packageType string) bool {
	for _, item := range validPackageTypes {
		if item == packageType {
			return true
		}
	}
	return false
}

func IsPackageTypesValid(packageTypes []string) bool {
	for _, item := range packageTypes {
		if !IsPackageTypeValid(item) {
			return false
		}
	}
	return true
}

func GetTimeInMs(t time.Time) string {
	return fmt.Sprint(t.UnixMilli())
}

func GetErrorResponse(code int, message string) *a.Error {
	return &a.Error{
		Code:    fmt.Sprint(code),
		Message: message,
	}
}

func GetSortByOrder(sortOrder string) string {
	defaultSortOrder := "ASC"
	decreasingSortOrder := "DESC"
	if len(sortOrder) == 0 {
		return defaultSortOrder
	}
	if sortOrder == decreasingSortOrder {
		return decreasingSortOrder
	}
	return defaultSortOrder
}

func sortKey(slice []string, target string) string {
	for _, item := range slice {
		if item == target {
			return item
		}
	}
	return "createdAt"
}

func GetSortByField(sortByField string, resource string) string {
	switch resource {
	case RepositoryResource:
		sortkey := sortKey(registrySort, sortByField)
		return RegistrySortMap[sortkey]
	case ArtifactResource:
		sortkey := sortKey(artifactSort, sortByField)
		return artifactSortMap[sortkey]
	case ArtifactVersionResource:
		sortkey := sortKey(artifactVersionSort, sortByField)
		return artifactVersionSortMap[sortkey]
	case ArtifactFilesResource:
		sortkey := sortKey(artifactFilesSort, sortByField)
		return artifactFilesSortMap[sortkey]
	}
	return "created_at"
}

func GetPageLimit(pageSize *a.PageSize) int {
	defaultPageSize := 10
	if pageSize != nil {
		return int(*pageSize)
	}
	return defaultPageSize
}

func GetOffset(pageSize *a.PageSize, pageNumber *a.PageNumber) int {
	defaultOffset := 0
	if pageSize == nil || pageNumber == nil {
		return defaultOffset
	}
	if *pageNumber == 0 {
		return 0
	}
	return (int(*pageSize)) * int(*pageNumber)
}

func GetPageNumber(pageNumber *a.PageNumber) int64 {
	defaultPageNumber := int64(1)
	if pageNumber == nil {
		return defaultPageNumber
	}
	return int64(*pageNumber)
}

func GetSuccessResponse() *a.Success {
	return &a.Success{
		Status: a.StatusSUCCESS,
	}
}

func GetPageCount(count int64, pageSize int) int64 {
	return int64(math.Ceil(float64(count) / float64(pageSize)))
}

func GetImageSize(size string) string {
	sizeVal, _ := strconv.ParseInt(size, 10, 64)
	return GetSize(sizeVal)
}

func GetSize(sizeVal int64) string {
	size := bytesize.New(float64(sizeVal))
	return size.String()
}

func GetRegistryRef(rootIdentifier string, registryName string) string {
	return rootIdentifier + "/" + registryName
}

func GetRepoURLWithoutProtocol(registryURL string) string {
	repoURL := registryURL
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Error parsing URL: ")
		return ""
	}

	return parsedURL.Host + parsedURL.Path
}

func GetTagURL(artifact string, version string, registryURL string) string {
	url := registryURL
	url += "/" + artifact + "/"
	url += version
	return url
}

func GetPullCommand(
	image string, tag string,
	packageType string, registryURL string,
) string {
	switch packageType {
	case string(a.PackageTypeDOCKER):
		return GetDockerPullCommand(image, tag, registryURL)
	case string(a.PackageTypeHELM):
		return GetHelmPullCommand(image, tag, registryURL)
	case string(a.PackageTypeGENERIC):
		return GetGenericArtifactFileDownloadCommand(registryURL, image, tag, "<FILENAME>")
	default:
		return ""
	}
}

func GetDockerPullCommand(
	image string,
	tag string, registryURL string,
) string {
	return "docker pull " + GetRepoURLWithoutProtocol(registryURL) + "/" + image + ":" + tag
}

func GetHelmPullCommand(image string, tag string, registryURL string) string {
	return "helm pull oci://" + GetRepoURLWithoutProtocol(registryURL) + "/" + image + " --version " + tag
}

func GetGenericArtifactFileDownloadCommand(regURL, artifact, version, filename string) string {
	downloadCommand := "curl --location '<HOSTNAME>/<ARTIFACT>:<VERSION>:<FILENAME>' --header 'x-api-key: <API_KEY>'" +
		" -J -O"

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<HOSTNAME>": regURL,
		"<ARTIFACT>": artifact,
		"<VERSION>":  version,
		"<FILENAME>": filename,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func GetMavenArtifactFileDownloadCommand(regURL, artifact, version, filename string) string {
	downloadCommand := "curl --location '<HOSTNAME>/<ARTIFACT>/<VERSION>/<FILENAME>'" +
		" --header 'x-api-key: <IDENTITY_TOKEN>' -O"

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<HOSTNAME>": regURL,
		"<ARTIFACT>": artifact,
		"<VERSION>":  version,
		"<FILENAME>": filename,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

// CleanURLPath removes leading and trailing spaces and trailing slashes from the given URL string.
func CleanURLPath(input *string) {
	if input == nil {
		return
	}
	// Parse the input to URL
	u, err := url.Parse(*input)
	if err != nil {
		return
	}

	// Clean the path by removing trailing slashes and spaces
	cleanedPath := strings.TrimRight(strings.TrimSpace(u.Path), "/")

	// Update the URL path in the original input string
	u.Path = cleanedPath

	// Update the input string with the cleaned URL string representation
	*input = u.String()
}
