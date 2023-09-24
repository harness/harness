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

package rpc

const (
	// MetadataKeyRequestID is the key used to store the request ID in the metadata.
	MetadataKeyRequestID = "x-request-id"

	MetadataKeyEnvironmentVariables = "x-gitrpc-envars"

	// ServiceUploadPack is the service constant used for triggering the upload pack operation.
	ServiceUploadPack = "upload-pack"

	// ServiceReceivePack is the service constant used for triggering the receive pack operation.
	ServiceReceivePack = "receive-pack"
)
