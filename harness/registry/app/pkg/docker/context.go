// Source: https://github.com/distribution/distribution

// Copyright 2014 https://github.com/distribution/distribution Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package docker

import (
	"context"

	"github.com/harness/gitness/registry/app/storage"

	v2 "github.com/distribution/distribution/v3/registry/api/v2"
	"github.com/opencontainers/go-digest"
)

// Context should contain the request specific context for use in across
// handlers. Resources that don't need to be shared across handlers should not
// be on this object.
type Context struct {
	*App
	context.Context
	URLBuilder   *v2.URLBuilder
	OciBlobStore storage.OciBlobStore
	Upload       storage.BlobWriter
	UUID         string
	Digest       digest.Digest
	State        BlobUploadState
}

// Value overrides context.Context.Value to ensure that calls are routed to
// correct context.
func (ctx *Context) Value(key interface{}) interface{} {
	return ctx.Context.Value(key)
}

// blobUploadState captures the state serializable state of the blob upload.
type BlobUploadState struct {
	// name is the primary repository under which the blob will be linked.
	Name string

	Path string

	// UUID identifies the upload.
	UUID string

	// offset contains the current progress of the upload.
	Offset int64
}
