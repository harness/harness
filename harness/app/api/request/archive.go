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

package request

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/git/api"
)

const (
	PathParamArchiveGitRef       = "*"
	QueryParamArchivePaths       = "path"
	QueryParamArchivePrefix      = "prefix"
	QueryParamArchiveAttributes  = "attributes"
	QueryParamArchiveTime        = "time"
	QueryParamArchiveCompression = "compression"
)

func Ext(path string) string {
	found := ""
	for _, format := range api.ArchiveFormats {
		if strings.HasSuffix(path, "."+string(format)) {
			if len(found) == 0 || len(found) < len(format) {
				found = string(format)
			}
		}
	}
	return found
}

func ParseArchiveParams(r *http.Request) (api.ArchiveParams, string, error) {
	// separate rev and ref part from url, for example:
	// api/v1/repos/root/demo/+/archive/refs/heads/main.zip
	// will produce rev=refs/heads and ref=main.zip
	path := PathParamOrEmpty(r, PathParamArchiveGitRef)
	rev, filename := filepath.Split(path)
	// use ext as format specifier
	format := Ext(filename)
	// prefix is used for git archive to prefix all paths.
	prefix, _ := QueryParam(r, QueryParamArchivePrefix)
	attributes, _ := QueryParam(r, QueryParamArchiveAttributes)

	var mtime *time.Time
	timeStr, _ := QueryParam(r, QueryParamArchiveTime)
	if timeStr != "" {
		value, err := time.Parse(time.DateTime, timeStr)
		if err == nil {
			mtime = &value
		}
	}

	var compression *int
	compressionStr, _ := QueryParam(r, QueryParamArchiveCompression)
	if compressionStr != "" {
		value, err := strconv.Atoi(compressionStr)
		if err == nil {
			compression = &value
		}
	}

	archFormat, err := api.ParseArchiveFormat(format)
	if err != nil {
		return api.ArchiveParams{}, "", err
	}

	// get name from filename
	name := strings.TrimSuffix(filename, "."+format)
	return api.ArchiveParams{
		Format:      archFormat,
		Prefix:      prefix,
		Attributes:  api.ArchiveAttribute(attributes),
		Time:        mtime,
		Compression: compression,
		Treeish:     rev + name,
		Paths:       r.URL.Query()[QueryParamArchivePaths],
	}, filename, nil
}
