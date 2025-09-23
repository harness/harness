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

var validScopes = []string{
	string(a.Ancestors),
	string(a.Descendants),
	string(a.None),
}

var validPackageTypes = []string{
	string(a.PackageTypeDOCKER),
	string(a.PackageTypeHELM),
	string(a.PackageTypeGENERIC),
	string(a.PackageTypeMAVEN),
	string(a.PackageTypePYTHON),
	string(a.PackageTypeNPM),
	string(a.PackageTypeRPM),
	string(a.PackageTypeNUGET),
	string(a.PackageTypeCARGO),
	string(a.PackageTypeGO),
	string(a.PackageTypeHUGGINGFACE),
}

var validUpstreamSources = []string{
	string(a.UpstreamConfigSourceCustom),
	string(a.UpstreamConfigSourceDockerhub),
	string(a.UpstreamConfigSourceAwsEcr),
	string(a.UpstreamConfigSourceMavenCentral),
	string(a.UpstreamConfigSourcePyPi),
	string(a.UpstreamConfigSourceNpmJs),
	string(a.UpstreamConfigSourceNugetOrg),
	string(a.UpstreamConfigSourceCrates),
	string(a.UpstreamConfigSourceGoProxy),
	string(a.UpstreamConfigSourceHuggingFace),
}

var validArtifactTypesMapping = map[string][]string{
	string(a.PackageTypeHUGGINGFACE): {string(a.ArtifactTypeModel), string(a.ArtifactTypeDataset)},
}

func ValidatePackageTypes(packageTypes []string) error {
	if commons.IsEmpty(packageTypes) || IsPackageTypesValid(packageTypes) {
		return nil
	}
	return errors.New("invalid package type")
}

func ValidateScope(scope string) error {
	if len(scope) == 0 || IsScopeValid(scope) {
		return nil
	}
	return errors.New("invalid scope")
}

func ValidateAndGetArtifactType(packageType a.PackageType, artifactTypeParam string) (*a.ArtifactType, error) {
	validTypes, ok := validArtifactTypesMapping[string(packageType)]
	if !ok {
		return nil, errors.New("invalid package type")
	}
	for _, t := range validTypes {
		if t == artifactTypeParam {
			at := a.ArtifactType(artifactTypeParam)
			return &at, nil
		}
	}
	return nil, errors.New("invalid artifact type for package type")
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
		*upstreamConfig.Source != a.UpstreamConfigSourcePyPi &&
		*upstreamConfig.Source != a.UpstreamConfigSourceNpmJs &&
		*upstreamConfig.Source != a.UpstreamConfigSourceNugetOrg &&
		*upstreamConfig.Source != a.UpstreamConfigSourceCrates &&
		*upstreamConfig.Source != a.UpstreamConfigSourceGoProxy {
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

func IsScopeValid(scope string) bool {
	for _, item := range validScopes {
		if item == scope {
			return true
		}
	}
	return false
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
	image string, version string,
	packageType string, registryURL string, setupDetailsAuthHeaderPrefix string, artifactType *a.ArtifactType,
	byTag bool,
) string {
	switch packageType {
	case string(a.PackageTypeDOCKER):
		return GetDockerPullCommand(image, version, registryURL, byTag)
	case string(a.PackageTypeHELM):
		return GetHelmPullCommand(image, version, registryURL, byTag)
	case string(a.PackageTypeGENERIC):
		return GetGenericArtifactFileDownloadCommand(registryURL, image, version, "<FILENAME>", setupDetailsAuthHeaderPrefix)
	case string(a.PackageTypePYTHON):
		return GetPythonDownloadCommand(image, version)
	case string(a.PackageTypeNPM):
		return GetNPMDownloadCommand(image, version)
	case string(a.PackageTypeRPM):
		return GetRPMDownloadCommand(image, version)
	case string(a.PackageTypeNUGET):
		return GetNugetDownloadCommand(image, version)
	case string(a.PackageTypeCARGO):
		return GetCargoDownloadCommand(image, version)
	case string(a.PackageTypeGO):
		return GetGoDownloadCommand(image, version)
	case string(a.PackageTypeHUGGINGFACE):
		return GetHuggingFaceArtifactFileDownloadCommand(registryURL, image, version, "<FILENAME>",
			setupDetailsAuthHeaderPrefix, artifactType)
	default:
		return ""
	}
}

func GetDockerPullCommand(
	image string, version string, registryURL string, byTag bool,
) string {
	var versionDelimiter string
	if byTag {
		versionDelimiter = ":"
	} else {
		versionDelimiter = "@"
	}
	return "docker pull " + GetRepoURLWithoutProtocol(registryURL) + "/" + image + versionDelimiter + version
}

func GetHelmPullCommand(image string, version string, registryURL string, byTag bool) string {
	var versionDelimiter string
	if byTag {
		versionDelimiter = " --version "
	} else {
		versionDelimiter = "@"
	}
	return "helm pull oci://" + GetRepoURLWithoutProtocol(registryURL) + "/" + image + versionDelimiter + version
}

func GetRPMDownloadCommand(artifact, version string) string {
	downloadCommand := "yum install <ARTIFACT>-<VERSION>"

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<ARTIFACT>": artifact,
		"<VERSION>":  version,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func GetNugetDownloadCommand(artifact, version string) string {
	downloadCommand := "nuget install <ARTIFACT> -Version <VERSION> -Source <SOURCE>"

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<ARTIFACT>": artifact,
		"<VERSION>":  version,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func GetCargoDownloadCommand(artifact, version string) string {
	downloadCommand := "cargo add <ARTIFACT>@<VERSION> --registry <REGISTRY>"

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<ARTIFACT>": artifact,
		"<VERSION>":  version,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func GetGoDownloadCommand(artifact, version string) string {
	downloadCommand := "go get <ARTIFACT>@<VERSION>"

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<ARTIFACT>": artifact,
		"<VERSION>":  version,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func GetNPMDownloadCommand(artifact, version string) string {
	downloadCommand := "npm install <ARTIFACT>@<VERSION> "

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<ARTIFACT>": artifact,
		"<VERSION>":  version,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func GetPythonDownloadCommand(artifact, version string) string {
	downloadCommand := "pip install <ARTIFACT>==<VERSION> "

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<ARTIFACT>": artifact,
		"<VERSION>":  version,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func GetGenericFileDownloadCommand(
	regURL, artifact, version, filename string,
	setupDetailsAuthHeaderPrefix string,
) string {
	downloadCommand := "curl --location '<HOSTNAME>/<ARTIFACT>:<VERSION>:<FILENAME>'" +
		" --header '<AUTH_HEADER_PREFIX> <API_KEY>'" +
		" -J -O"

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<HOSTNAME>":           regURL,
		"<ARTIFACT>":           artifact,
		"<VERSION>":            version,
		"<FILENAME>":           filename,
		"<AUTH_HEADER_PREFIX>": setupDetailsAuthHeaderPrefix,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func GetGenericArtifactFileDownloadCommand(
	regURL, artifact, version, filePath string,
	setupDetailsAuthHeaderPrefix string,
) string {
	downloadCommand := "curl --location '<HOSTNAME>/<ARTIFACT>/<VERSION>/<FILENAME>'" +
		" --header '<AUTH_HEADER_PREFIX> <API_KEY>'" +
		" -J -O"

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<HOSTNAME>":           regURL,
		"<ARTIFACT>":           artifact,
		"<VERSION>":            version,
		"<FILENAME>":           filePath,
		"<AUTH_HEADER_PREFIX>": setupDetailsAuthHeaderPrefix,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func GetHuggingFaceArtifactFileDownloadCommand(
	regURL, artifact, version, filename string,
	setupDetailsAuthHeaderPrefix string, artifactType *a.ArtifactType,
) string {
	downloadCommand := "curl --location '<HOSTNAME>/<ARTIFACT_TYPE>/<ARTIFACT>/resolve/<VERSION>/<FILENAME>'" +
		" --header '<AUTH_HEADER_PREFIX> <API_KEY>'" +
		" -J -O"

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<HOSTNAME>":           regURL,
		"<ARTIFACT_TYPE>":      string(*artifactType),
		"<ARTIFACT>":           artifact,
		"<VERSION>":            version,
		"<FILENAME>":           filename,
		"<AUTH_HEADER_PREFIX>": setupDetailsAuthHeaderPrefix,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func getDownloadURL(
	registryURL string,
	packageType a.PackageType, artifact, version, filename string,
) string {
	//nolint:exhaustive
	switch packageType {
	case a.PackageTypeGENERIC:
		return fmt.Sprintf("%s/%s/%s?filename=%s", registryURL, artifact, version, filename)
	case a.PackageTypeCARGO, a.PackageTypeMAVEN, a.PackageTypeNPM, a.PackageTypeNUGET, a.PackageTypePYTHON,
		a.PackageTypeRPM:
		return fmt.Sprintf("%s/%s/%s/%s", registryURL, artifact, version, filename)
	default:
		return ""
	}
}

func GetNPMArtifactFileDownloadCommand(
	regURL, artifact, version, filename string,
) string {
	downloadCommand := "curl --location '<HOSTNAME>/<ARTIFACT>/-/<FILENAME>'" +
		" --header 'Authorization: Bearer <API_KEY>'" +
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

func GetRPMArtifactFileDownloadCommand(regURL, filename string, setupDetailsAuthHeaderPrefix string) string {
	downloadCommand := "curl --location '<HOSTNAME>/package<FILENAME>' --header '<AUTH_HEADER_PREFIX> <API_KEY>'" +
		" -J -O"
	replacements := map[string]string{
		"<HOSTNAME>":           regURL,
		"<FILENAME>":           filename,
		"<AUTH_HEADER_PREFIX>": setupDetailsAuthHeaderPrefix,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func GetNugetArtifactFileDownloadCommand(
	regURL, artifact, version, filename,
	setupDetailsAuthHeaderPrefix string,
) string {
	downloadCommand := "curl --location '<HOSTNAME>/<ARTIFACT>/<VERSION>/<FILENAME>'" +
		" --header '<AUTH_HEADER_PREFIX> <API_KEY>'" +
		" -J -O"

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<HOSTNAME>":           regURL,
		"<ARTIFACT>":           artifact,
		"<VERSION>":            version,
		"<FILENAME>":           filename,
		"<AUTH_HEADER_PREFIX>": setupDetailsAuthHeaderPrefix,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func GetCargoArtifactFileDownloadCommand(
	regURL, artifact, version,
	setupDetailsAuthHeaderPrefix string,
) string {
	downloadCommand := "curl --location '<HOSTNAME>/api/v1/crates/<ARTIFACT>/<VERSION>/download'" +
		" --header '<AUTH_HEADER_PREFIX> <API_KEY>'" +
		" -J -O"

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<HOSTNAME>":           regURL,
		"<ARTIFACT>":           artifact,
		"<VERSION>":            version,
		"<AUTH_HEADER_PREFIX>": setupDetailsAuthHeaderPrefix,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func GetGoArtifactFileDownloadCommand(
	regURL, artifact, filename,
	setupDetailsAuthHeaderPrefix string,
) string {
	downloadCommand := "curl --location '<HOSTNAME>/<ARTIFACT>/@v/<FILENAME>'" +
		" --header '<AUTH_HEADER_PREFIX> <API_KEY>'" +
		" -J -O"

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<HOSTNAME>":           regURL,
		"<ARTIFACT>":           artifact,
		"<FILENAME>":           filename,
		"<AUTH_HEADER_PREFIX>": setupDetailsAuthHeaderPrefix,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func GetMavenArtifactFileDownloadCommand(
	regURL, artifact, version, filename string,
	setupDetailsAuthHeaderPrefix string,
) string {
	downloadCommand := "curl --location '<HOSTNAME>/<ARTIFACT>/<VERSION>/<FILENAME>'" +
		" --header '<AUTH_HEADER_PREFIX> <IDENTITY_TOKEN>' -O"

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<HOSTNAME>":           regURL,
		"<ARTIFACT>":           artifact,
		"<VERSION>":            version,
		"<FILENAME>":           filename,
		"<AUTH_HEADER_PREFIX>": setupDetailsAuthHeaderPrefix,
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
