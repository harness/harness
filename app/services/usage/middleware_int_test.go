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

//go:build integration
// +build integration

package usage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/types"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

const (
	numRequests  = 10
	fileSize     = 1 * 1024 * 1024 // 100 MB
	testURL      = "http://localhost:8080"
	testSpaceRef = "root"
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:      100,              // Allow up to 100 idle connections
		MaxConnsPerHost:   10,               // Maximum concurrent connections per host
		IdleConnTimeout:   30 * time.Second, // Keep idle connections open for reuse
		DisableKeepAlives: false,            // Allow connection reuse
		WriteBufferSize:   64 * 1024,        // 64 KB write buffer
		ReadBufferSize:    64 * 1024,
	},
	Timeout: 2 * time.Minute,
}

// Helper function to generate random file data
func generateRandomData(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}

// Simulates an upload request with proper multipart form boundary
func simulateUploadRequest(t *testing.T) {
	fileData := generateRandomData(fileSize)

	// Create multipart form file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body) // Create a multipart writer

	part, err := writer.CreateFormFile("file", "testing.dat")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	_, _ = part.Write(fileData)
	_ = writer.Close() // Must close the writer to finalize the boundary

	// Create request and set Content-Type properly
	req, err := http.NewRequest(http.MethodPost, testURL+"/testing/"+testSpaceRef, body)
	if err != nil {
		t.Fatalf("Failed to create upload request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType()) // Correctly sets boundary

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("Upload request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	bodyResp, _ := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected HTTP 200 for upload")
	assert.Contains(t, string(bodyResp), "File uploaded successfully", "Upload should be successful")
}

// Simulate a download request
func simulateDownloadRequest(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, testURL+"/testing/"+testSpaceRef, nil)
	if err != nil {
		t.Fatalf("Failed to create download request: %v", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send download request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, _ := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected HTTP 200 OK for download")
	assert.NotEmpty(t, body, "Expected non-empty response body")
}

// File upload handler
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Simulate file processing
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	// For testing, we're just reading the file content (simulation)
	_, err = io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to process file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("File uploaded successfully"))
}

// File download handler (simulating a simple file download)
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write([]byte("This is a dummy file content"))
}

// Test function to run multiple uploads and downloads concurrently
func TestUploadDownloadMiddleware(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	spaceStore := &SpaceFinderMock{
		FindByRefFn: func(ctx context.Context, spaceRef string) (*types.SpaceCore, error) {
			return &types.SpaceCore{
				ID:         1,
				ParentID:   0,
				Path:       "",
				Identifier: "root",
			}, nil
		},
		FindByIDsFn: func(ctx context.Context, spaceIDs ...int64) ([]*types.SpaceCore, error) {
			return []*types.SpaceCore{}, nil
		},
	}

	metricsMock := &MetricsMock{
		UpsertOptimisticFn: func(ctx context.Context, in *types.UsageMetric) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		},
		GetMetricsFn: func(ctx context.Context, rootSpaceID int64, startDate int64, endDate int64) (*types.UsageMetric, error) {
			return &types.UsageMetric{}, nil
		},
		ListFn: func(ctx context.Context, start int64, end int64) ([]types.UsageMetric, error) {
			return []types.UsageMetric{}, nil
		},
	}

	mediator := NewMediator(ctx, spaceStore, metricsMock, Config{})
	// Start the server in a goroutine
	go func() {
		r := chi.NewRouter()
		r.Get("/health", func(writer http.ResponseWriter, r *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		r.Route(fmt.Sprintf("/testing/{%s}", request.PathParamRepoRef), func(r chi.Router) {
			r.Use(Middleware(mediator))
			r.Post("/", uploadHandler)
			r.Get("/", downloadHandler)
		})
		t.Log(http.ListenAndServe(":8080", r))
	}()

	// Allow the server to start before running tests
	waitServer(t)

	// Run the upload and download requests in parallel
	t.Run("UploadDownloadTest", func(t *testing.T) {
		t.Parallel() // Run tests in parallel

		// Create a WaitGroup for syncing concurrent requests
		var wg sync.WaitGroup
		for i := 0; i < numRequests; i++ {
			wg.Add(2)

			// Simulate upload request
			go func() {
				defer wg.Done()
				simulateUploadRequest(t)
			}()

			// Simulate download request
			go func() {
				defer wg.Done()
				simulateDownloadRequest(t)
			}()
		}

		// Wait for all requests to finish
		wg.Wait()
	})
}

func waitServer(t *testing.T) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, testURL+"/health", nil)
	if err != nil {
		t.Fatalf("failed to create health request: %v", err)
		return
	}

	for attempt := 1; attempt <= 5; attempt++ {
		resp, err := httpClient.Do(req)
		if err != nil {
			t.Logf("Failed to send health request after %d attempt with error: %v", attempt, err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			t.Logf("Expected HTTP 200 OK, got %d, attempt=%d, retrying...", resp.StatusCode, attempt)
			continue
		}

		// If it's a success break out of the loop
		break
	}
}
