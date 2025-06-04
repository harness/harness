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

package event

import (
	"context"
	"errors"
	"strings"

	a "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	s "github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

func ReportEventAsync(
	ctx context.Context,
	accountID string,
	reporter Reporter,
	action BlobAction,
	blobID int64,
	genericBlobID string,
	sha256 string,
	conf *types.Config,
) {
	var path string
	var err error
	switch {
	case blobID != 0:
		path, err = s.BlobPath(accountID, string(a.PackageTypeDOCKER), sha256)
	case genericBlobID != "":
		path, err = s.BlobPath(accountID, string(a.PackageTypeGENERIC), sha256)
	default:
		err = errors.New("blobID or genericBlobID must be set")
	}

	if err != nil {
		log.Error().
			Err(err).
			Int64("blobID", blobID).
			Str("genericBlobID", genericBlobID).
			Str("action", string(action)).
			Msg("Failed to determine blob path for event reporting")
		return
	}

	source := CloudLocation{
		Provider: ProviderValue[strings.ToUpper(conf.Registry.Storage.S3Storage.Provider)],
		Endpoint: conf.Registry.Storage.S3Storage.RegionEndpoint,
		Region:   conf.Registry.Storage.S3Storage.Region,
		Bucket:   conf.Registry.Storage.S3Storage.Bucket,
	}

	destinations := []CloudLocation{source} // TODO: Use the correct destination

	go reporter.ReportEvent(ctx, &ReplicationDetails{
		AccountID:     accountID,
		Action:        action,
		BlobID:        blobID,
		GenericBlobID: genericBlobID,
		Path:          path,
		Source:        source,
		Destinations:  destinations,
	}, "")
}
