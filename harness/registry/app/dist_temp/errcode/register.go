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

package errcode

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"sync"
	"syscall"

	storagedriver "github.com/harness/gitness/registry/app/driver"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/api/googleapi"
)

var (
	errorCodeToDescriptors = map[CodeError]ErrorDescriptor{}
	idToDescriptors        = map[string]ErrorDescriptor{}
	groupToDescriptors     = map[string][]ErrorDescriptor{}
)

var (
	// ErrCodeUnknown is a generic error that can be used as a last
	// resort if there is no situation-specific error message that can be used.
	ErrCodeUnknown = register(
		"errcode", ErrorDescriptor{
			Value:   "UNKNOWN",
			Message: "unknown error",
			Description: `Generic error returned when the error does not have an
			                                            API classification.`,
			HTTPStatusCode: http.StatusInternalServerError,
		},
	)

	// ErrCodeUnsupported is returned when an operation is not supported.
	ErrCodeUnsupported = register(
		"errcode", ErrorDescriptor{
			Value:   "UNSUPPORTED",
			Message: "The operation is unsupported.",
			Description: `The operation was unsupported due to a missing
		implementation or invalid set of parameters.`,
			HTTPStatusCode: http.StatusMethodNotAllowed,
		},
	)

	// ErrCodeUnauthorized is returned if a request requires
	// authentication.
	ErrCodeUnauthorized = register(
		"errcode", ErrorDescriptor{
			Value:   "UNAUTHORIZED",
			Message: "authentication required",
			Description: `The access controller was unable to authenticate
		the client. Often this will be accompanied by a
		Www-Authenticate HTTP response header indicating how to
		authenticate.`,
			HTTPStatusCode: http.StatusUnauthorized,
		},
	)

	// ErrCodeDenied is returned if a client does not have sufficient
	// permission to perform an action.
	ErrCodeDenied = register(
		"errcode", ErrorDescriptor{
			Value:   "DENIED",
			Message: "requested access to the resource is denied",
			Description: `The access controller denied access for the
		operation on a resource.`,
			HTTPStatusCode: http.StatusForbidden,
		},
	)

	// ErrCodeUnavailable provides a common error to report unavailability
	// of a service or endpoint.
	ErrCodeUnavailable = register(
		"errcode", ErrorDescriptor{
			Value:          "UNAVAILABLE",
			Message:        "service unavailable",
			Description:    "Returned when a service is not available",
			HTTPStatusCode: http.StatusServiceUnavailable,
		},
	)

	// ErrCodeTooManyRequests is returned if a client attempts too many
	// times to contact a service endpoint.
	ErrCodeTooManyRequests = register(
		"errcode", ErrorDescriptor{
			Value:   "TOOMANYREQUESTS",
			Message: "too many requests",
			Description: `Returned when a client attempts to contact a
		service too many times`,
			HTTPStatusCode: http.StatusTooManyRequests,
		},
	)

	// ErrCodeConnectionReset provides an error to report a client dropping the
	// connection.
	ErrCodeConnectionReset = register(
		"errcode", ErrorDescriptor{
			Value:       "CONNECTIONRESET",
			Message:     "connection reset by peer",
			Description: "Returned when the client closes the connection unexpectedly",
			// 400 is the most fitting error code in the HTTP spec, 499 is used by
			// nginx (and within this project as well), and is specific to this scenario,
			// but it is preferable to stay within the spec.
			HTTPStatusCode: http.StatusBadRequest,
		},
	)

	// ErrCodeRequestCanceled provides an error to report a canceled request. This is usually due to a
	// context.Canceled error.
	ErrCodeRequestCanceled = register(
		"errcode", ErrorDescriptor{
			Value:          "REQUESTCANCELED",
			Message:        "request canceled",
			Description:    "Returned when the client cancels the request",
			HTTPStatusCode: http.StatusBadRequest,
		},
	)

	// ErrCodeInvalidRequest provides an error when the request is invalid.
	ErrCodeInvalidRequest = register(
		"errcode", ErrorDescriptor{
			Value:          "INVALID REQUEST",
			Message:        "invalid request",
			Description:    "Returned when the request is invalid",
			HTTPStatusCode: http.StatusBadRequest,
		},
	)
)

const errGroup = "registry.api.v2"

var (
	// ErrCodeDigestInvalid is returned when uploading a blob if the
	// provided digest does not match the blob contents.
	ErrCodeDigestInvalid = register(
		errGroup, ErrorDescriptor{
			Value:   "DIGEST_INVALID",
			Message: "provided digest did not match uploaded content",
			Description: `When a blob is uploaded, the registry will check that
		the content matches the digest provided by the client. The error may
		include a detail structure with the key "digest", including the
		invalid digest string. This error may also be returned when a manifest
		includes an invalid layer digest.`,
			HTTPStatusCode: http.StatusBadRequest,
		},
	)

	// ErrCodeSizeInvalid is returned when uploading a blob if the provided.
	ErrCodeSizeInvalid = register(
		errGroup, ErrorDescriptor{
			Value:   "SIZE_INVALID",
			Message: "provided length did not match content length",
			Description: `When a layer is uploaded, the provided size will be
		checked against the uploaded content. If they do not match, this error
		will be returned.`,
			HTTPStatusCode: http.StatusBadRequest,
		},
	)

	// ErrCodeRangeInvalid is returned when uploading a blob if the provided
	// content range is invalid.
	ErrCodeRangeInvalid = register(
		errGroup, ErrorDescriptor{
			Value:   "RANGE_INVALID",
			Message: "invalid content range",
			Description: `When a layer is uploaded, the provided range is checked
		against the uploaded chunk. This error is returned if the range is
		out of order.`,
			HTTPStatusCode: http.StatusRequestedRangeNotSatisfiable,
		},
	)

	// ErrCodeNameInvalid is returned when the name in the manifest does not
	// match the provided name.
	ErrCodeNameInvalid = register(
		errGroup, ErrorDescriptor{
			Value:   "NAME_INVALID",
			Message: "invalid repository name",
			Description: `Invalid repository name encountered either during
		manifest validation or any API operation.`,
			HTTPStatusCode: http.StatusBadRequest,
		},
	)

	// ErrCodeTagInvalid is returned when the tag in the manifest does not
	// match the provided tag.
	ErrCodeTagInvalid = register(
		errGroup, ErrorDescriptor{
			Value:   "TAG_INVALID",
			Message: "manifest tag did not match URI",
			Description: `During a manifest upload, if the tag in the manifest
		does not match the uri tag, this error will be returned.`,
			HTTPStatusCode: http.StatusBadRequest,
		},
	)

	// ErrCodeNameUnknown when the repository name is not known.
	ErrCodeNameUnknown = register(
		errGroup, ErrorDescriptor{
			Value:   "NAME_UNKNOWN",
			Message: "repository name not known to registry",
			Description: `This is returned if the name used during an operation is
		unknown to the registry.`,
			HTTPStatusCode: http.StatusNotFound,
		},
	)

	// ErrCodeManifestUnknown returned when image manifest is unknown.
	ErrCodeManifestUnknown = register(
		errGroup, ErrorDescriptor{
			Value:   "MANIFEST_UNKNOWN",
			Message: "manifest unknown",
			Description: `This error is returned when the manifest, identified by
		name and tag is unknown to the repository.`,
			HTTPStatusCode: http.StatusNotFound,
		},
	)

	// ErrCodeManifestQuarantined returned when image manifest is quarantined.
	ErrCodeManifestQuarantined = register(
		errGroup, ErrorDescriptor{
			Value:   "ARTIFACT_QUARANTINED",
			Message: "artifact quarantined",
			Description: `This error is returned when the manifest, identified by
		name or tag is quarantined`,
			HTTPStatusCode: http.StatusForbidden,
		},
	)

	// ErrCodeManifestReferencedInList is returned when attempting to delete a manifest that is still referenced by at
	// least one manifest list.
	ErrCodeManifestReferencedInList = register(
		errGroup, ErrorDescriptor{
			Value:   "MANIFEST_REFERENCED",
			Message: "manifest referenced by a manifest list",
			Description: `The manifest is still referenced by at least one manifest list and therefore the delete cannot
		proceed.`,
			HTTPStatusCode: http.StatusConflict,
		},
	)

	// ErrCodeManifestInvalid returned when an image manifest is invalid,
	// typically during a PUT operation. This error encompasses all errors
	// encountered during manifest validation that aren't signature errors.
	ErrCodeManifestInvalid = register(
		errGroup, ErrorDescriptor{
			Value:   "MANIFEST_INVALID",
			Message: "manifest invalid",
			Description: `During upload, manifests undergo several checks ensuring
		validity. If those checks fail, this error may be returned, unless a
		more specific error is included. The detail will contain information
		the failed validation.`,
			HTTPStatusCode: http.StatusBadRequest,
		},
	)

	// ErrCodeManifestUnverified is returned when the manifest fails
	// signature verification.
	ErrCodeManifestUnverified = register(
		errGroup, ErrorDescriptor{
			Value:   "MANIFEST_UNVERIFIED",
			Message: "manifest failed signature verification",
			Description: `During manifest upload, if the manifest fails signature
		verification, this error will be returned.`,
			HTTPStatusCode: http.StatusBadRequest,
		},
	)

	// ErrCodeManifestReferenceLimit is returned when a manifest has more
	// references than the configured limit.
	ErrCodeManifestReferenceLimit = register(
		errGroup, ErrorDescriptor{
			Value:   "MANIFEST_REFERENCE_LIMIT",
			Message: "too many manifest references",
			Description: `This error may be returned when a manifest references more than
		the configured limit allows.`,
			HTTPStatusCode: http.StatusBadRequest,
		},
	)

	// ErrCodeManifestPayloadSizeLimit is returned when a manifest payload is
	// bigger than the configured limit.
	ErrCodeManifestPayloadSizeLimit = register(
		errGroup, ErrorDescriptor{
			Value:   "MANIFEST_SIZE_LIMIT",
			Message: "payload size limit exceeded",
			Description: `This error may be returned when a manifest payload size is bigger than
		the configured limit allows.`,
			HTTPStatusCode: http.StatusBadRequest,
		},
	)

	// ErrCodeManifestBlobUnknown is returned when a manifest blob is
	// unknown to the registry.
	ErrCodeManifestBlobUnknown = register(
		errGroup, ErrorDescriptor{
			Value:   "MANIFEST_BLOB_UNKNOWN",
			Message: "blob unknown to registry",
			Description: `This error may be returned when a manifest blob is 
		unknown to the registry.`,
			HTTPStatusCode: http.StatusBadRequest,
		},
	)

	// ErrCodeBlobUnknown is returned when a blob is unknown to the
	// registry. This can happen when the manifest references a nonexistent
	// layer or the result is not found by a blob fetch.
	ErrCodeBlobUnknown = register(
		errGroup, ErrorDescriptor{
			Value:   "BLOB_UNKNOWN",
			Message: "blob unknown to registry",
			Description: `This error may be returned when a blob is unknown to the
		registry in a specified repository. This can be returned with a
		standard get or if a manifest references an unknown layer during
		upload.`,
			HTTPStatusCode: http.StatusNotFound,
		},
	)

	// ErrCodeBlobUploadUnknown is returned when an upload is unknown.
	ErrCodeBlobUploadUnknown = register(
		errGroup, ErrorDescriptor{
			Value:   "BLOB_UPLOAD_UNKNOWN",
			Message: "blob upload unknown to registry",
			Description: `If a blob upload has been cancelled or was never
		started, this error code may be returned.`,
			HTTPStatusCode: http.StatusNotFound,
		},
	)

	// ErrCodeBlobUploadInvalid is returned when an upload is invalid.
	ErrCodeBlobUploadInvalid = register(
		errGroup, ErrorDescriptor{
			Value:   "BLOB_UPLOAD_INVALID",
			Message: "blob upload invalid",
			Description: `The blob upload encountered an error and can no
		longer proceed.`,
			HTTPStatusCode: http.StatusNotFound,
		},
	)

	// ErrCodePaginationNumberInvalid is returned when the `n` parameter is
	// not an integer, or `n` is negative.
	ErrCodePaginationNumberInvalid = register(
		errGroup, ErrorDescriptor{
			Value:   "PAGINATION_NUMBER_INVALID",
			Message: "invalid number of results requested",
			Description: `Returned when the "n" parameter (number of results
		to return) is not an integer, "n" is negative or "n" is bigger than
		the maximum allowed.`,
			HTTPStatusCode: http.StatusBadRequest,
		},
	)
)

const gitnessErrGroup = "gitness.api.v1"

var (
	ErrCodeRootNotFound = register(
		gitnessErrGroup, ErrorDescriptor{
			Value:          "ROOT_NOT_FOUND",
			Message:        "Root not found",
			Description:    "The root does not exist",
			HTTPStatusCode: http.StatusNotFound,
		},
	)
	ErrCodeParentNotFound = register(
		gitnessErrGroup, ErrorDescriptor{
			Value:          "PARENT_NOT_FOUND",
			Message:        "Parent not found",
			Description:    "The parent does not exist",
			HTTPStatusCode: http.StatusNotFound,
		},
	)
	ErrCodeRegNotFound = register(
		gitnessErrGroup, ErrorDescriptor{
			Value:          "REGISTRY_NOT_FOUND",
			Message:        "registry not found",
			Description:    "The registry does not exist",
			HTTPStatusCode: http.StatusNotFound,
		},
	)
)

var (
	nextCode     = 1000
	registerLock sync.Mutex
)

// Register will make the passed-in error known to the environment and
// return a new ErrorCode.
func Register(group string, descriptor ErrorDescriptor) CodeError {
	return register(group, descriptor)
}

// register will make the passed-in error known to the environment and
// return a new ErrorCode.
func register(group string, descriptor ErrorDescriptor) CodeError {
	registerLock.Lock()
	defer registerLock.Unlock()

	descriptor.Code = CodeError(nextCode)

	if _, ok := idToDescriptors[descriptor.Value]; ok {
		panic(fmt.Sprintf("ErrorValue %q is already registered", descriptor.Value))
	}
	if _, ok := errorCodeToDescriptors[descriptor.Code]; ok {
		panic(fmt.Sprintf("ErrorCode %v is already registered", descriptor.Code))
	}

	groupToDescriptors[group] = append(groupToDescriptors[group], descriptor)
	errorCodeToDescriptors[descriptor.Code] = descriptor
	idToDescriptors[descriptor.Value] = descriptor

	nextCode++
	return descriptor.Code
}

type byValue []ErrorDescriptor

func (a byValue) Len() int           { return len(a) }
func (a byValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byValue) Less(i, j int) bool { return a[i].Value < a[j].Value }

// GetGroupNames returns the list of Error group names that are registered.
func GetGroupNames() []string {
	keys := []string{}

	for k := range groupToDescriptors {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GetErrorCodeGroup returns the named group of error descriptors.
func GetErrorCodeGroup(name string) []ErrorDescriptor {
	desc := groupToDescriptors[name]
	sort.Sort(byValue(desc))
	return desc
}

// GetErrorAllDescriptors returns a slice of all ErrorDescriptors that are
// registered, irrespective of what group they're in.
func GetErrorAllDescriptors() []ErrorDescriptor {
	result := []ErrorDescriptor{}

	for _, group := range GetGroupNames() {
		result = append(result, GetErrorCodeGroup(group)...)
	}
	sort.Sort(byValue(result))
	return result
}

// FromUnknownError will try to parse an unknown error and infer the appropriate Error to use.
func FromUnknownError(err error) Error {
	// return if this is an Error already
	var e Error
	if errors.As(err, &e) {
		return e
	}

	// if this is a storage driver catch-all error (storagedriver.Error), extract the enclosed error
	var sdErr storagedriver.Error
	if errors.As(err, &sdErr) {
		err = sdErr.Detail
	}

	// use 503 Service Unavailable for network timeout errors
	var netError net.Error
	if ok := errors.As(err, &netError); ok && netError.Timeout() {
		return ErrCodeUnavailable.WithDetail(err)
	}

	var netOpError *net.OpError
	if errors.As(err, &netOpError) {
		// use 400 Bad Request if the client drops the connection during the request
		var syscallErr *os.SyscallError
		if errors.As(err, &syscallErr) && errors.Is(syscallErr.Err, syscall.ECONNRESET) {
			return ErrCodeConnectionReset.WithDetail(err)
		}

		// use 503 Service Unavailable for network connection refused or unknown host errors
		return ErrCodeUnavailable.WithDetail(err)
	}

	// use 400 Bad Request for canceled requests
	if errors.Is(err, context.Canceled) {
		return ErrCodeRequestCanceled.WithDetail(err)
	}

	// use 503 Service Unavailable for database connection failures
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgerrcode.IsConnectionException(pgErr.Code) {
		return ErrCodeUnavailable.WithDetail(err)
	}

	// propagate a 503 Service Unavailable status from the storage backends
	var gcsErr *googleapi.Error
	if errors.As(err, &gcsErr) {
		if gcsErr.Code == http.StatusServiceUnavailable {
			return ErrCodeUnavailable.WithDetail(gcsErr.Error())
		}
	}

	// otherwise, we're not sure what the error is or how to react, use 500 Internal Server Error
	return ErrCodeUnknown.WithDetail(err)
}
