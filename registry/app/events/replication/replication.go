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

package replication

import "github.com/harness/gitness/events"

const RegistryBlobCreatedEvent events.EventType = "registry-blob-created"

type BlobAction string
type Provider string

type CloudLocation struct {
	Provider Provider `json:"provider,omitempty"`
	Endpoint string   `json:"endpoint,omitempty"`
	Region   string   `json:"region,omitempty"`
	Bucket   string   `json:"bucket,omitempty"`
}

// ReplicationDetails represents the ReplicationDetails message from the proto file.
type ReplicationDetails struct {
	AccountID     string          `json:"account_id,omitempty"`
	Action        BlobAction      `json:"action,omitempty"`
	BlobID        int64           `json:"blob_id,omitempty"`
	GenericBlobID string          `json:"generic_blob_id,omitempty"`
	Path          string          `json:"path,omitempty"`
	Source        CloudLocation   `json:"source"`
	Destinations  []CloudLocation `json:"destinations,omitempty"`
}

const (
	BlobCreate BlobAction = "BlobCreate"
	BlobDelete BlobAction = "BlobDelete"
)

const (
	CLOUDFLARE Provider = "CLOUDFLARE"
	GCS        Provider = "GCS"
)

var ProviderValue = map[string]Provider{
	"CLOUDFLARE": CLOUDFLARE,
	"GCS":        GCS,
}
