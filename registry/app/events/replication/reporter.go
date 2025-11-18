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

import (
	"context"
	"errors"
	"strings"

	"github.com/harness/gitness/events"
	a "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	s "github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

const RegistryBlobsReplication = "registry-blobs-replication"

type Reporter interface {
	ReportEventAsync(
		ctx context.Context,
		accountID string,
		action BlobAction,
		blobID int64,
		genericBlobID string,
		sha256 string,
		conf *types.Config,
		destinationBuckets []CloudLocation,
	)
}

type reporter struct {
	innerReporter *events.GenericReporter
}

func NewReporter(
	eventsSystem *events.System,
) (Reporter, error) {
	innerReporter, err := events.NewReporter(eventsSystem, RegistryBlobsReplication)
	if err != nil {
		return nil, errors.New("failed to create new GenericReporter for registry blobs replication from event system")
	}

	return &reporter{
		innerReporter: innerReporter,
	}, nil
}

func (r reporter) ReportEventAsync(
	ctx context.Context,
	accountID string,
	action BlobAction,
	blobID int64,
	genericBlobID string,
	sha256 string,
	conf *types.Config,
	destinationBuckets []CloudLocation,
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
		log.Ctx(ctx).Error().
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

	destinations := destinationBuckets
	if len(destinations) == 0 {
		return
	}

	go func() {
		eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, RegistryBlobCreatedEvent, &ReplicationDetails{
			AccountID:     accountID,
			Action:        action,
			BlobID:        blobID,
			GenericBlobID: genericBlobID,
			Path:          path,
			Source:        source,
			Destinations:  destinations,
		})
		if err != nil {
			log.Ctx(ctx).Err(err).Msgf("failed to send blob replication created event")
			return
		}
		log.Ctx(ctx).Debug().Msgf("reported blob replication event with id '%s'", eventID)
	}()
}

type Noop struct {
}

func (*Noop) ReportEventAsync(
	_ context.Context, _ string, _ BlobAction, _ int64, _ string, _ string, _ *types.Config, _ []CloudLocation,
) {
}
