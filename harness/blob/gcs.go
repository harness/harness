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
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

// scopes best practice: https://cloud.google.com/compute/docs/access/service-accounts#scopes_best_practice
const defaultScope = "https://www.googleapis.com/auth/cloud-platform"

type GCSStore struct {
	// Bucket is the name of the GCS bucket to use.
	cachedClient        *storage.Client
	config              Config
	tokenExpirationTime time.Time
}

func NewGCSStore(ctx context.Context, cfg Config) (Store, error) {
	switch {
	case cfg.KeyPath != "":
		// Use service account key file [Development and Non-GCP environments]
		client, err := storage.NewClient(ctx, option.WithCredentialsFile(cfg.KeyPath))
		if err != nil {
			return nil, fmt.Errorf("failed to create GCS client with service account key: %w", err)
		}
		return &GCSStore{
			config:              cfg,
			cachedClient:        client,
			tokenExpirationTime: time.Now().Add(cfg.ImpersonationLifetime),
		}, nil
	case cfg.TargetPrincipal == "":
		// Use direct Workload Identity [GKE environments with direct SA access]
		client, err := storage.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCS client with default credentials: %w", err)
		}
		return &GCSStore{
			config:       cfg,
			cachedClient: client,
			// No token expiration for default credentials - managed by GCP
			tokenExpirationTime: time.Time{}, // Zero time - never expires
		}, nil
	default:
		// Use Workload Identity with impersonation [GKE environments with SA impersonation]
		client, err := createNewImpersonatedClient(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCS client with workload identity impersonation: %w", err)
		}
		return &GCSStore{
			config:              cfg,
			cachedClient:        client,
			tokenExpirationTime: time.Now().Add(cfg.ImpersonationLifetime),
		}, nil
	}
}

func (c *GCSStore) Upload(ctx context.Context, file io.Reader, filePath string) error {
	gcsClient, err := c.getClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve latest client: %w", err)
	}

	bkt := gcsClient.Bucket(c.config.Bucket)
	wc := bkt.Object(filePath).NewWriter(ctx)
	defer func() {
		cErr := wc.Close()
		if cErr != nil {
			log.Ctx(ctx).Warn().Err(cErr).
				Msgf("failed to close gcs blob writer for file %q in bucket %q", filePath, c.config.Bucket)
		}
	}()
	if _, err := io.Copy(wc, file); err != nil {
		// Best effort attempt to delete the file on upload failure.
		deleteErr := gcsClient.Bucket(c.config.Bucket).Object(filePath).Delete(ctx)
		if deleteErr != nil {
			log.Ctx(ctx).Warn().Err(deleteErr).Msgf(
				"failed to cleanup file %q from bucket %q after write to gcs failed with %s",
				filePath, c.config.Bucket, err)
		}
		return fmt.Errorf("failed to write file to GCS: %w", err)
	}

	return nil
}

func (c *GCSStore) GetSignedURL(
	ctx context.Context,
	filePath string,
	expire time.Time,
	opts ...SignURLOption) (string, error) {
	gcsClient, err := c.getClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve latest client: %w", err)
	}

	config := SignURLConfig{
		Method: http.MethodGet,
	}

	for _, opt := range opts {
		opt.Apply(&config)
	}

	bkt := gcsClient.Bucket(c.config.Bucket)
	signedURL, err := bkt.SignedURL(filePath, &storage.SignedURLOptions{
		Method:          config.Method,
		Expires:         expire,
		ContentType:     config.ContentType,
		Headers:         config.Headers,
		QueryParameters: config.QueryParameters,
		Insecure:        config.Insecure,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create signed URL for file %q: %w", filePath, err)
	}
	return signedURL, nil
}

func (c *GCSStore) Download(ctx context.Context, filePath string) (io.ReadCloser, error) {
	gcsClient, err := c.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve latest client: %w", err)
	}

	bkt := gcsClient.Bucket(c.config.Bucket)
	rc, err := bkt.Object(filePath).NewReader(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to create reader for file %q in bucket %q: %w", filePath, c.config.Bucket, err)
	}

	return rc, nil
}

func createNewImpersonatedClient(ctx context.Context, cfg Config) (*storage.Client, error) {
	// Use workload identity impersonation default credentials (GKE environment)
	ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
		TargetPrincipal: cfg.TargetPrincipal,
		Scopes:          []string{defaultScope}, // Required field
		Lifetime:        cfg.ImpersonationLifetime,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to impersonate the client service account %q: %w", cfg.TargetPrincipal, err)
	}

	// Generate a new token
	token, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token from impersonated credentials: %w", err)
	}

	client, err := storage.NewClient(ctx, option.WithTokenSource(oauth2.StaticTokenSource(token)))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client with workload identity impersonation: %w", err)
	}
	return client, nil
}

func (c *GCSStore) getClient(ctx context.Context) (*storage.Client, error) {
	// Skip token refresh for direct Workload Identity (no impersonation)
	if c.config.KeyPath == "" && c.config.TargetPrincipal == "" {
		return c.cachedClient, nil
	}

	err := c.checkAndRefreshToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	return c.cachedClient, nil
}

func (c *GCSStore) checkAndRefreshToken(ctx context.Context) error {
	if time.Now().Before(c.tokenExpirationTime) {
		return nil
	}
	now := time.Now()
	client, err := createNewImpersonatedClient(ctx, c.config)
	if err != nil {
		return fmt.Errorf("failed to create GCS client with workload identity impersonation after expiration: %w", err)
	}
	c.cachedClient = client
	c.tokenExpirationTime = now.Add(c.config.ImpersonationLifetime)
	return nil
}
