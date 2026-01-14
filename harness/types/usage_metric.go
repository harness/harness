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

package types

import "time"

type UsageMetric struct {
	Date            time.Time `json:"-"`
	RootSpaceID     int64     `json:"root_space_id"`
	BandwidthOut    int64     `json:"bandwidth_out"`
	BandwidthIn     int64     `json:"bandwidth_in"`
	StorageTotal    int64     `json:"storage_total"`
	LFSStorageTotal int64     `json:"lfs_storage_total"`
	Pushes          int64     `json:"pushes"`
}
