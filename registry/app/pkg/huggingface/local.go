// Copyright 2023 Harness, Inc.
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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	urlprovider "github.com/harness/gitness/app/url"
	apicontract "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	huggingfacemetadata "github.com/harness/gitness/registry/app/metadata/huggingface"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	huggingfacetype "github.com/harness/gitness/registry/app/pkg/types/huggingface"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

var (
	frontMatterRE = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---`)
	allowedTypes  = map[string]bool{"model": true, "dataset": true}
)

const (
	rootPathString   = "/"
	tmp              = "tmp"
	maxCommitEntries = 50000 // Add reasonable limit
	contentTypeJSON  = "application/json"
)

type localRegistry struct {
	localBase   base.LocalBase
	fileManager filemanager.FileManager
	proxyStore  store.UpstreamProxyConfigRepository
	tx          dbtx.Transactor
	registryDao store.RegistryRepository
	imageDao    store.ImageRepository
	artifactDao store.ArtifactRepository
	urlProvider urlprovider.Provider
}

func (c *localRegistry) ValidateYaml(_ context.Context, _ huggingfacetype.ArtifactInfo, body io.ReadCloser) (
	headers *commons.ResponseHeaders, response *huggingfacetype.ValidateYamlResponse, err error) {
	headers = &commons.ResponseHeaders{
		Headers: map[string]string{"Content-Type": contentTypeJSON},
	}
	req := &huggingfacetype.ValidateYamlRequest{}

	if err = json.NewDecoder(body).Decode(&req); err != nil {
		err = usererror.BadRequest("invalid request json body")
		headers.Code = http.StatusBadRequest
		return headers, nil, err
	}

	if req.RepoType == nil || !allowedTypes[*req.RepoType] {
		err = usererror.BadRequest("unsupported repoType")
		headers.Code = http.StatusBadRequest
		return headers, nil, err
	}
	if req.Content == nil || frontMatterRE.FindStringSubmatch(*req.Content) == nil {
		return headers, &huggingfacetype.ValidateYamlResponse{
			Errors:   &[]huggingfacetype.Debug{},
			Warnings: &[]huggingfacetype.Debug{},
		}, nil
	}

	// Parse YAML
	var meta map[string]interface{}
	if err = yaml.Unmarshal([]byte(frontMatterRE.FindStringSubmatch(*req.Content)[1]), &meta); err != nil {
		errMsg := stringPtr(fmt.Sprintf("Invalid YAML: %v", err))
		errors := &[]huggingfacetype.Debug{
			{Message: errMsg},
		}
		response = &huggingfacetype.ValidateYamlResponse{
			Errors:   errors,
			Warnings: &[]huggingfacetype.Debug{},
		}
		headers.Code = http.StatusBadRequest
		return headers, response, nil
	}

	// Validate fields
	// Create a slice of validations to perform
	validations := []struct {
		field    string
		validate func(map[string]interface{}, string) (*huggingfacetype.ValidateYamlResponse, bool)
	}{
		{"license", validateString},
		{"pipeline_tag", validateString},
		{"tags", validateSlice},
		{"widget", validateSlice},
	}

	// Run all validations
	for _, v := range validations {
		if res, ok := v.validate(meta, v.field); !ok {
			headers.Code = http.StatusBadRequest
			return headers, res, nil
		}
	}
	headers.Code = http.StatusOK
	return headers,
		&huggingfacetype.ValidateYamlResponse{
			Errors:   &[]huggingfacetype.Debug{},
			Warnings: &[]huggingfacetype.Debug{},
		}, nil
}

func (c *localRegistry) PreUpload(_ context.Context, _ huggingfacetype.ArtifactInfo, body io.ReadCloser) (
	headers *commons.ResponseHeaders, response *huggingfacetype.PreUploadResponse, err error) {
	headers = &commons.ResponseHeaders{
		Headers: map[string]string{"Content-Type": contentTypeJSON},
	}
	req := &huggingfacetype.PreUploadRequest{}
	if err = json.NewDecoder(body).Decode(&req); err != nil {
		err = usererror.BadRequest("invalid request json body")
		headers.Code = http.StatusBadRequest
		return headers, nil, err
	}
	if req.Files == nil || len(*req.Files) == 0 {
		err = usererror.BadRequest("invalid request json body")
		headers.Code = http.StatusBadRequest
		return
	}

	// Create the response file slice
	files := make([]huggingfacetype.PreUploadResponseFile, 0, len(*req.Files))
	for _, f := range *req.Files {
		files = append(files, huggingfacetype.NewPreUploadResponseFile(f.Path))
	}

	resp := &huggingfacetype.PreUploadResponse{
		Files: &files,
	}
	headers.Code = http.StatusOK
	return headers, resp, nil
}

func (c *localRegistry) RevisionInfo(ctx context.Context, info huggingfacetype.ArtifactInfo,
	queryParams map[string][]string) (
	headers *commons.ResponseHeaders, response *huggingfacetype.RevisionInfoResponse, err error) {
	headers = &commons.ResponseHeaders{
		Headers: map[string]string{"Content-Type": contentTypeJSON},
	}
	expand := queryParams["expand"]
	if len(expand) > 0 && expand[0] == "xetEnabled" {
		headers.Code = http.StatusOK
		return headers, &huggingfacetype.RevisionInfoResponse{
			XetEnabled: false,
			Metadata: huggingfacemetadata.Metadata{
				ID: info.Repo,
			},
		}, nil
	}

	//todo: add logs
	image, err := c.imageDao.GetByNameAndType(ctx, info.RegistryID, info.Repo, &info.RepoType)
	if err != nil {
		return headers, nil, err
	}

	artifact, err := c.artifactDao.GetByName(ctx, image.ID, info.Revision)
	if err != nil {
		return headers, nil, err
	}

	metadata := &huggingfacemetadata.HuggingFaceMetadata{}
	err = json.Unmarshal(artifact.Metadata, metadata)
	if err != nil {
		return headers, nil, err
	}
	metadata.SHA = info.Revision

	if metadata.ID == "" {
		err = usererror.BadRequest("repo id not found")
		headers.Code = http.StatusNotFound
		return headers, nil, err
	}

	headers.Code = http.StatusOK
	return headers, &huggingfacetype.RevisionInfoResponse{
		XetEnabled: false,
		Metadata:   metadata.Metadata,
	}, nil
}

func (c *localRegistry) LfsInfo(ctx context.Context, info huggingfacetype.ArtifactInfo, body io.ReadCloser,
	token string) (headers *commons.ResponseHeaders, response *huggingfacetype.LfsInfoResponse, err error) {
	headers = &commons.ResponseHeaders{
		Headers: map[string]string{"Content-Type": "application/vnd.git-lfs+json"},
	}
	resp := &huggingfacetype.LfsInfoResponse{
		Objects: &[]huggingfacetype.LfsObjectResponse{},
	}

	pkgURL := c.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "huggingface")

	req := &huggingfacetype.LfsInfoRequest{}
	if err = json.NewDecoder(body).Decode(req); err != nil {
		err = usererror.BadRequest("invalid LFS info body")
		headers.Code = http.StatusBadRequest
		return headers, nil, err
	}

	if req.Objects == nil || len(*req.Objects) == 0 {
		headers.Code = http.StatusOK
		return headers, resp, nil
	}

	for _, obj := range *req.Objects {
		objResp := huggingfacetype.LfsObjectResponse{
			Oid:  obj.Oid,
			Size: obj.Size,
		}

		filePath, _ := c.fileManager.HeadSHA256(ctx, obj.Oid, info.RegistryID, info.RootParentID)

		exists := filePath != ""

		switch req.Operation {
		case "upload":
			if exists {
				continue
			}
			objResp.Actions = &map[string]huggingfacetype.LfsAction{
				req.Operation: lfsAction(getBlobURL(pkgURL, req.Operation, obj.Oid, token, info), obj.Oid, token),
				"verify":      lfsAction(getBlobURL(pkgURL, "verify", obj.Oid, token, info), obj.Oid, token),
			}

		case "download":
			if exists {
				objResp.Actions = &map[string]huggingfacetype.LfsAction{
					req.Operation: lfsAction(getBlobURL(pkgURL, req.Operation, obj.Oid, token, info), obj.Oid, token),
				}
			} else {
				objResp.Error = &huggingfacetype.LfsError{
					Code:    http.StatusNotFound,
					Message: "object not found",
				}
			}

		default:
			objResp.Error = &huggingfacetype.LfsError{
				Code:    http.StatusBadRequest,
				Message: "unsupported operation: " + req.Operation,
			}
		}

		*resp.Objects = append(*resp.Objects, objResp)
	}
	headers.Code = http.StatusOK
	return headers, resp, nil
}

func (c *localRegistry) LfsUpload(ctx context.Context, info huggingfacetype.ArtifactInfo,
	body io.ReadCloser) (headers *commons.ResponseHeaders, response *huggingfacetype.LfsUploadResponse, err error) {
	headers = &commons.ResponseHeaders{
		Headers: map[string]string{"Content-Type": contentTypeJSON},
	}
	resp := &huggingfacetype.LfsUploadResponse{}
	tmpFileName := getTmpFileName(info, info.SHA256)

	fileInfo, tmpFileName, err := c.fileManager.UploadTempFileToPath(ctx, info.RootIdentifier, nil,
		tmpFileName, tmpFileName, body)
	if err != nil {
		log.Ctx(ctx).Info().Msgf("Upload failed for file %s, %v", tmpFileName, err)
		headers.Code = http.StatusInternalServerError
		return headers, resp, err
	}
	log.Ctx(ctx).Info().Msgf("Uploaded file %s to %s", tmpFileName, fileInfo.Filename)
	resp.Success = true
	headers.Code = http.StatusCreated
	return headers, resp, nil
}

func (c *localRegistry) LfsVerify(ctx context.Context, info huggingfacetype.ArtifactInfo,
	_ io.ReadCloser) (headers *commons.ResponseHeaders, response *huggingfacetype.LfsVerifyResponse, err error) {
	headers = &commons.ResponseHeaders{
		Headers: map[string]string{"Content-Type": contentTypeJSON},
	}
	resp := &huggingfacetype.LfsVerifyResponse{}

	filePath, _ := c.fileManager.HeadSHA256(ctx, info.SHA256, info.RegistryID, info.RootParentID)
	exists := c.FileExists(ctx, info)
	if filePath == "" && !exists {
		log.Ctx(ctx).Info().Msgf("File %s does not exist", info.SHA256)
		return headers, nil, fmt.Errorf("file %s does not exist", info.SHA256)
	}

	resp.Success = true
	headers.Code = http.StatusOK
	return headers, resp, nil
}

func (c *localRegistry) CommitRevision(ctx context.Context, info huggingfacetype.ArtifactInfo, body io.ReadCloser) (
	headers *commons.ResponseHeaders, response *huggingfacetype.CommitRevisionResponse, err error) {
	headers = &commons.ResponseHeaders{
		Headers: map[string]string{"Content-Type": contentTypeJSON},
	}
	headerInfo := &huggingfacetype.HeaderInfo{}
	lfsFiles := &[]huggingfacetype.LfsFileInfo{}
	siblings := &[]huggingfacemetadata.Sibling{}
	session, _ := request.AuthSessionFrom(ctx)
	commitBytes, _ := io.ReadAll(body)
	commits := string(commitBytes)
	scanner := bufio.NewScanner(strings.NewReader(commits))
	processedEntries := 0
	for scanner.Scan() {
		if processedEntries >= maxCommitEntries {
			log.Ctx(ctx).Warn().Msgf("Reached maximum commit entries limit: %d", maxCommitEntries)
			break
		}
		processedEntries++
		line := scanner.Text()
		var entry huggingfacetype.CommitEntry
		if err = json.Unmarshal([]byte(line), &entry); err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("Failed to unmarshal commit entry: %s", line)
			continue
		}
		data, err2 := json.Marshal(entry.Value)
		if err2 != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Failed to marshal header info")
			return headers, nil, err2
		}

		switch entry.Key {
		case "header":
			if err = json.Unmarshal(data, &headerInfo); err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal header info")
				return headers, nil, err
			}
		case "lfsFile":
			var lfsFile huggingfacetype.LfsFileInfo
			if err = json.Unmarshal(data, &lfsFile); err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal LFS file info")
				continue
			}
			*lfsFiles = append(*lfsFiles, lfsFile)
			*siblings = append(*siblings, huggingfacemetadata.Sibling{RFilename: lfsFile.Path})
		}
	}

	if err = scanner.Err(); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error reading commits string")
		return headers, nil, err
	}

	var readme string
	movedBlobs := make(map[string]bool)
	for _, lfsFile := range *lfsFiles {
		filePath := fmt.Sprintf("/%s/%s/%s/%s", info.RepoType, info.Repo, info.Revision, lfsFile.Path)
		savedPath, _ := c.fileManager.HeadSHA256(ctx, lfsFile.Oid, info.RegistryID, info.RootParentID)
		if savedPath == filePath {
			movedBlobs[lfsFile.Oid] = true
			continue
		}
		if savedPath != "" && !movedBlobs[lfsFile.Oid] {
			err = c.fileManager.SaveNodes(ctx, filePath, info.RegistryID, info.RootParentID, session.Principal.ID,
				lfsFile.Oid)
			if err != nil {
				log.Ctx(ctx).Info().Msgf("Failed to move file with sha %s to %s", lfsFile.Oid, lfsFile.Path)
				return headers, nil, err
			}
			continue
		}
		tmpFileName := getTmpFileName(info, lfsFile.Oid)
		if strings.EqualFold(lfsFile.Path, "Readme.md") {
			readme = c.readme(ctx, info, tmpFileName, lfsFile.Size)
		}
		fileInfo := types.FileInfo{
			Size:     lfsFile.Size,
			Sha256:   lfsFile.Oid,
			Filename: lfsFile.Path,
		}
		err = c.fileManager.MoveTempFile(ctx, filePath, info.RegistryID, info.RootParentID, info.RootIdentifier,
			fileInfo, tmpFileName, session.Principal.ID)
		movedBlobs[lfsFile.Oid] = true

		if err != nil {
			log.Ctx(ctx).Info().Msgf("Failed to move file %s to %s", tmpFileName, lfsFile.Path)
			return headers, nil, err
		}
	}

	readme = modifyReadme(readme)

	modelMetadata := huggingfacemetadata.Metadata{
		ID:           info.Repo,
		ModelID:      info.Repo,
		SHA:          "",
		LibraryName:  "unknown",
		Siblings:     *siblings,
		LastModified: time.Now().UTC().Format(time.RFC3339),
		Private:      true,
		Readme:       readme,
		CardData:     &huggingfacemetadata.CardData{},
	}
	if headerInfo.Summary != "" {
		modelMetadata.CardData.Tags = append(modelMetadata.CardData.Tags, "summary:"+headerInfo.Summary)
	}

	modelMetadataBytes, _ := json.Marshal(modelMetadata)

	err = c.tx.WithTx(
		ctx, func(ctx context.Context) error {
			dbImage := &types.Image{
				Name:         info.Repo,
				RegistryID:   info.RegistryID,
				Enabled:      true,
				ArtifactType: &info.RepoType,
			}
			if err2 := c.imageDao.CreateOrUpdate(ctx, dbImage); err2 != nil {
				log.Ctx(ctx).Error().Err(err2).Msgf("Failed to create image: %s", info.Repo)
				return err2
			}
			dbArtifact := &types.Artifact{
				ImageID:   dbImage.ID,
				Version:   info.Revision,
				Metadata:  json.RawMessage(modelMetadataBytes),
				UpdatedAt: time.Now(),
			}
			if _, err2 := c.artifactDao.CreateOrUpdate(ctx, dbArtifact); err2 != nil {
				log.Ctx(ctx).Error().Err(err2).Msgf("Failed to create artifact: %s", info.Revision)
				return err2
			}

			return nil
		})

	if err != nil {
		return headers, nil, err
	}
	commitURL := fmt.Sprintf("%s/%s/commit/%s", info.RepoType, info.Repo, info.Revision)

	headers.Code = http.StatusOK
	headers.Headers["X-Harness-Commit-Processed"] = "true"
	resp := &huggingfacetype.CommitRevisionResponse{
		CommitURL:         commitURL,
		CommitMessage:     headerInfo.Summary,
		CommitDescription: headerInfo.Description,
		OID:               info.Revision,
		Success:           true,
	}
	return headers, resp, nil
}

func (c *localRegistry) HeadFile(ctx context.Context, info huggingfacetype.ArtifactInfo, fileName string) (
	headers *commons.ResponseHeaders, err error) {
	headers = &commons.ResponseHeaders{
		Headers: map[string]string{},
	}
	dbImage, err := c.imageDao.GetByNameAndType(ctx, info.RegistryID, info.Repo, &info.RepoType)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to get image: %s", string(info.RepoType)+"/"+info.Repo)
		headers.Headers["Content-Type"] = contentTypeJSON
		headers.Code = http.StatusNotFound
		return headers, err
	}

	_, err = c.artifactDao.GetByName(ctx, dbImage.ID, info.Revision)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to get artifact: %s", info.Revision)
		headers.Headers["Content-Type"] = contentTypeJSON
		headers.Code = http.StatusNotFound
		return headers, err
	}

	sha256, size, err := c.fileManager.HeadFile(ctx, "/"+string(info.RepoType)+"/"+info.Repo+"/"+info.Revision+"/"+fileName,
		info.RegistryID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to get file: %s", fileName)
		headers.Headers["Content-Type"] = contentTypeJSON
		headers.Code = http.StatusNotFound
		return headers, err
	}
	headers.Code = http.StatusOK
	headers.Headers["Content-Type"] = "application/octet-stream"
	headers.Headers["Content-Length"] = fmt.Sprintf("%d", size)
	headers.Headers["X-Repo-Commit"] = info.Revision
	headers.Headers["ETag"] = sha256
	return headers, nil
}

func (c *localRegistry) DownloadFile(ctx context.Context, info huggingfacetype.ArtifactInfo, fileName string) (
	headers *commons.ResponseHeaders, body *storage.FileReader, redirectURL string, err error) {
	headers, err = c.HeadFile(ctx, info, fileName)
	if err != nil {
		return headers, nil, "", err
	}

	body, _, redirectURL, err = c.fileManager.DownloadFile(ctx,
		"/"+string(info.RepoType)+"/"+info.Repo+"/"+info.Revision+"/"+fileName, info.RegistryID,
		info.RegIdentifier, info.RootIdentifier, true)
	return headers, body, redirectURL, err
}

func (c *localRegistry) FileExists(ctx context.Context, info huggingfacetype.ArtifactInfo) bool {
	tmpFileName := getTmpFileName(info, info.SHA256)
	tmpPath := path.Join(rootPathString, info.RootIdentifier, tmp, tmpFileName)
	exists, _, err := c.fileManager.FileExists(ctx, info.RootIdentifier, tmpPath)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to check if file exists: %s", tmpPath)
	}
	return exists
}

func getBlobURL(pkgURL, operation, sha256, token string, info huggingfacetype.ArtifactInfo) string {
	return fmt.Sprintf(pkgURL+"/api/%ss/%s/%s/multipart/%s/%s?sig=%s",
		info.RepoType, info.Repo, info.Revision, operation, sha256, strings.TrimPrefix(token, "Bearer "))
}

func modifyReadme(readme string) string {
	// Check if the README contains YAML frontmatter
	yamlRegex := regexp.MustCompile(`(?s)^---\n(.+?)\n---\n(.+)$`)
	matches := yamlRegex.FindStringSubmatch(readme)

	if len(matches) < 3 {
		// No YAML frontmatter found or invalid format, return original content
		return readme
	}

	// Extract YAML and Markdown parts
	yamlContent := matches[1]
	markdownContent := matches[2]

	// Parse YAML into a map
	var yamlData map[string]interface{}
	err := yaml.Unmarshal([]byte(yamlContent), &yamlData)
	if err != nil {
		// Failed to parse YAML, return original content
		return readme
	}

	// Marshal YAML data back to string with pretty formatting
	prettyYamlBytes, err := yaml.Marshal(yamlData)
	if err != nil {
		// Failed to marshal YAML, return original content
		return readme
	}

	// Convert bytes to string and prepare formatted output
	prettyYamlString := string(prettyYamlBytes)

	// Create pretty output with YAML in code block followed by markdown
	var buffer bytes.Buffer
	buffer.WriteString("## Card details\n")
	buffer.WriteString("```yaml\n")
	buffer.WriteString(prettyYamlString)
	buffer.WriteString("\n```\n\n")
	buffer.WriteString(markdownContent)

	return buffer.String()
}

func (c *localRegistry) readme(ctx context.Context, info huggingfacetype.ArtifactInfo, tmpFileName string,
	fileSize int64) string {
	reader, _, err := c.fileManager.DownloadTempFile(ctx, fileSize, tmpFileName, info.RootIdentifier)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("Failed to download readme file %s", tmpFileName)
		return ""
	}
	defer reader.Close()
	readmeBytes, err2 := io.ReadAll(reader)
	if err2 != nil {
		log.Ctx(ctx).Warn().Msgf("Failed to read readme file %s", tmpFileName)
		return ""
	}
	return string(readmeBytes)
}

func lfsAction(blobURL, oid, token string) huggingfacetype.LfsAction {
	return huggingfacetype.LfsAction{
		Href: blobURL,
		Header: map[string]string{
			"X-Checksum-Sha256": oid,
			"Authorization":     token,
		},
	}
}

func validateSlice(meta map[string]interface{}, metaKey string) (*huggingfacetype.ValidateYamlResponse, bool) {
	if v, ok := meta[metaKey]; ok && !isSlice(v) {
		msg := stringPtr(fmt.Sprintf(`"%s" must be an array`, metaKey))
		warnings := &[]huggingfacetype.Debug{
			{Message: msg},
		}
		return &huggingfacetype.ValidateYamlResponse{
			Errors:   &[]huggingfacetype.Debug{},
			Warnings: warnings,
		}, false
	}
	return nil, true
}

func getTmpFileName(info huggingfacetype.ArtifactInfo, sha256 string) string {
	return info.RootIdentifier + "/" + info.RegIdentifier + "/" + info.Repo + "/" + info.Revision + "/upload/" + sha256
}

func validateString(meta map[string]interface{}, metaKey string) (*huggingfacetype.ValidateYamlResponse, bool) {
	if v, ok := meta[metaKey]; ok && !isString(v) {
		msg := stringPtr(fmt.Sprintf(`"%s" must be a string`, metaKey))
		warnings := &[]huggingfacetype.Debug{
			{Message: msg},
		}
		return &huggingfacetype.ValidateYamlResponse{
			Errors:   &[]huggingfacetype.Debug{},
			Warnings: warnings,
		}, false
	}
	return nil, true
}

type LocalRegistry interface {
	Registry
}

func NewLocalRegistry(
	localBase base.LocalBase,
	fileManager filemanager.FileManager,
	proxyStore store.UpstreamProxyConfigRepository,
	tx dbtx.Transactor,
	registryDao store.RegistryRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	urlProvider urlprovider.Provider,
) LocalRegistry {
	return &localRegistry{
		localBase:   localBase,
		fileManager: fileManager,
		proxyStore:  proxyStore,
		tx:          tx,
		registryDao: registryDao,
		imageDao:    imageDao,
		artifactDao: artifactDao,
		urlProvider: urlProvider,
	}
}

func (c *localRegistry) GetArtifactType() apicontract.RegistryType {
	return apicontract.RegistryTypeVIRTUAL
}

func (c *localRegistry) GetPackageTypes() []apicontract.PackageType {
	return []apicontract.PackageType{apicontract.PackageTypeHUGGINGFACE}
}

func isSlice(v interface{}) bool { _, ok := v.([]interface{}); return ok }

func isString(v interface{}) bool { _, ok := v.(string); return ok }

// stringPtr returns a pointer to the given string.
func stringPtr(s string) *string {
	return &s
}
