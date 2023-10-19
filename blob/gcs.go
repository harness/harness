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

package blob

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"
)

type GCSStore struct {
	// Bucket is the name of the GCS bucket to use.
	bucket string
	client *storage.Client
}

func NewGCSStore(cfg Config) (Store, error) {
	// Use service account [Development and Non-GCP environments]
	if cfg.KeyPath != "" {
		client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(cfg.KeyPath))
		if err != nil {
			return nil, fmt.Errorf("failed to create GCS client with service account key: %w", err)
		}
		return &GCSStore{
			bucket: cfg.Bucket,
			client: client,
		}, nil
	}

	// Use workload identity default credentials (GKE environment)
	client, err := storage.NewClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client with workload identity or default credentials: %w", err)
	}
	return &GCSStore{
		bucket: cfg.Bucket,
		client: client,
	}, nil
}

func (c *GCSStore) Upload(ctx context.Context, file io.Reader, filePath string) error {
	wc := c.client.Bucket(c.bucket).Object(filePath).NewWriter(ctx)
	defer func() {
		cErr := wc.Close()
		if cErr != nil {
			log.Ctx(ctx).Err(cErr).
				Msgf("failed to close gcs blob writer for file '%s' in bucket '%s'", filePath, c.bucket)
		}
	}()
	if _, err := io.Copy(wc, file); err != nil {
		// Remove the file if it was created.
		deleteErr := c.client.Bucket(c.bucket).Object(filePath).Delete(ctx)
		if deleteErr != nil {
			return fmt.Errorf("failed to delete file: %s from bucket: %s %w", filePath, c.bucket, deleteErr)
		}

		return fmt.Errorf("failed to write file to GCS: %w", err)
	}

	return nil
}

func (c *GCSStore) GetSignedURL(filePath string) (string, error) {
	signedURL, err := c.client.Bucket(c.bucket).SignedURL(filePath, &storage.SignedURLOptions{
		Method:  http.MethodGet,
		Expires: time.Now().Add(1 * time.Hour),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create signed URL for file: %s %w", filePath, err)
	}
	return signedURL, nil
}
func (c *GCSStore) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}
