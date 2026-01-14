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

package manifest

import (
	"errors"
	"fmt"
	"strings"

	"github.com/opencontainers/go-digest"
)

// ErrAccessDenied is returned when an access to a requested resource is
// denied.
var ErrAccessDenied = errors.New("access denied")

// ErrManifestNotModified is returned when a conditional manifest GetByTag
// returns nil due to the client indicating it has the latest version.
var ErrManifestNotModified = errors.New("manifest not modified")

// ErrUnsupported is returned when an unimplemented or unsupported action is
// performed.
var ErrUnsupported = errors.New("operation unsupported")

// ErrSchemaV1Unsupported is returned when a client tries to upload a schema v1
// manifest but the registry is configured to reject it.
var ErrSchemaV1Unsupported = errors.New("manifest schema v1 unsupported")

// TagUnknownError is returned if the given tag is not known by the tag service.
type TagUnknownError struct {
	Tag string
}

func (err TagUnknownError) Error() string {
	return fmt.Sprintf("unknown tag=%s", err.Tag)
}

// RegistryUnknownError is returned if the named repository is not known by
// the registry.
type RegistryUnknownError struct {
	Name string
}

func (err RegistryUnknownError) Error() string {
	return fmt.Sprintf("unknown registry name=%s", err.Name)
}

// RegistryNameInvalidError should be used to denote an invalid registry
// name. Reason may set, indicating the cause of invalidity.
type RegistryNameInvalidError struct {
	Name   string
	Reason error
}

func (err RegistryNameInvalidError) Error() string {
	return fmt.Sprintf("registry name %q invalid: %v", err.Name, err.Reason)
}

// ManifestUnknownError is returned if the manifest is not known by the
// registry.
type UnknownError struct {
	Name string
	Tag  string
}

func (err UnknownError) Error() string {
	return fmt.Sprintf("unknown manifest name=%s tag=%s", err.Name, err.Tag)
}

// ManifestUnknownRevisionError is returned when a manifest cannot be found by
// revision within a registry.
type UnknownRevisionError struct {
	Name     string
	Revision digest.Digest
}

func (err UnknownRevisionError) Error() string {
	return fmt.Sprintf("unknown manifest name=%s revision=%s", err.Name, err.Revision)
}

// ManifestUnverifiedError is returned when the registry is unable to verify
// the manifest.
type UnverifiedError struct{}

func (UnverifiedError) Error() string {
	return "unverified manifest"
}

// ManifestReferencesExceedLimitError is returned when a manifest has too many references.
type ReferencesExceedLimitError struct {
	References int
	Limit      int
}

func (err ReferencesExceedLimitError) Error() string {
	return fmt.Sprintf("%d manifest references exceed reference limit of %d", err.References, err.Limit)
}

// ManifestPayloadSizeExceedsLimitError is returned when a manifest is bigger than the configured payload
// size limit.
type PayloadSizeExceedsLimitError struct {
	PayloadSize int
	Limit       int
}

// Error implements the error interface for ManifestPayloadSizeExceedsLimitError.
func (err PayloadSizeExceedsLimitError) Error() string {
	return fmt.Sprintf("manifest payload size of %d exceeds limit of %d", err.PayloadSize, err.Limit)
}

// ManifestVerificationErrors provides a type to collect errors encountered
// during manifest verification. Currently, it accepts errors of all types,
// but it may be narrowed to those involving manifest verification.
type VerificationErrors []error

func (errs VerificationErrors) Error() string {
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		parts = append(parts, err.Error())
	}

	return fmt.Sprintf("errors verifying manifest: %v", strings.Join(parts, ","))
}

// ManifestBlobUnknownError returned when a referenced blob cannot be found.
type BlobUnknownError struct {
	Digest digest.Digest
}

func (err BlobUnknownError) Error() string {
	return fmt.Sprintf("unknown blob %v on manifest", err.Digest)
}

// ManifestNameInvalidError should be used to denote an invalid manifest
// name. Reason may set, indicating the cause of invalidity.
type NameInvalidError struct {
	Name   string
	Reason error
}

func (err NameInvalidError) Error() string {
	return fmt.Sprintf("manifest name %q invalid: %v", err.Name, err.Reason)
}
