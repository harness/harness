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

package gcs

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/harness/gitness/registry/app/dist_temp/dcontext"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/driver/testsuites"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

var (
	gcsDriverConstructor func(rootDirectory string) (storagedriver.StorageDriver, error)
	skipCheck            func(tb testing.TB)
)

func init() {
	bucket := os.Getenv("REGISTRY_STORAGE_GCS_BUCKET")
	credentials := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	// Skip GCS storage driver tests if environment variable parameters are not provided
	skipCheck = func(tb testing.TB) {
		tb.Helper()

		if bucket == "" || credentials == "" {
			tb.Skip("The following environment variables must be set to enable these tests: " +
				"REGISTRY_STORAGE_GCS_BUCKET, GOOGLE_APPLICATION_CREDENTIALS")
		}
	}

	gcsDriverConstructor = func(rootDirectory string) (storagedriver.StorageDriver, error) {
		jsonKey, err := os.ReadFile(credentials)
		if err != nil {
			panic(fmt.Sprintf("Error reading JSON key : %v", err))
		}

		var ts oauth2.TokenSource
		var email string
		var privateKey []byte

		ts, err = google.DefaultTokenSource(dcontext.Background(), storage.ScopeFullControl)
		if err != nil {
			// Assume that the file contents are within the environment variable since it exists
			// but does not contain a valid file path
			jwtConfig, err := google.JWTConfigFromJSON(jsonKey, storage.ScopeFullControl)
			if err != nil {
				panic(fmt.Sprintf("Error reading JWT config : %s", err))
			}
			email = jwtConfig.Email
			privateKey = jwtConfig.PrivateKey
			if len(privateKey) == 0 {
				panic("Error reading JWT config : missing private_key property")
			}
			if email == "" {
				panic("Error reading JWT config : missing client_email property")
			}
			ts = jwtConfig.TokenSource(dcontext.Background())
		}

		gcs, err := storage.NewClient(dcontext.Background(), option.WithCredentialsJSON(jsonKey))
		if err != nil {
			panic(fmt.Sprintf("Error initializing gcs client : %v", err))
		}

		parameters := driverParameters{
			bucket:         bucket,
			rootDirectory:  rootDirectory,
			email:          email,
			privateKey:     privateKey,
			client:         oauth2.NewClient(dcontext.Background(), ts),
			chunkSize:      defaultChunkSize,
			gcs:            gcs,
			maxConcurrency: 8,
		}

		return New(context.Background(), parameters)
	}
}

func newDriverConstructor(tb testing.TB) testsuites.DriverConstructor {
	root := tb.TempDir()

	return func() (storagedriver.StorageDriver, error) {
		return gcsDriverConstructor(root)
	}
}

func TestGCSDriverSuite(t *testing.T) {
	skipCheck(t)
	testsuites.Driver(t, newDriverConstructor(t), false)
}

func BenchmarkGCSDriverSuite(b *testing.B) {
	skipCheck(b)
	testsuites.BenchDriver(b, newDriverConstructor(b))
}

// Test Committing a FileWriter without having called Write.
func TestCommitEmpty(t *testing.T) {
	skipCheck(t)

	validRoot := t.TempDir()

	driver, err := gcsDriverConstructor(validRoot)
	if err != nil {
		t.Fatalf("unexpected error creating rooted driver: %v", err)
	}

	filename := "/test" //nolint:goconst
	ctx := dcontext.Background()

	writer, err := driver.Writer(ctx, filename, false)
	// nolint:errcheck
	defer driver.Delete(ctx, filename)
	if err != nil {
		t.Fatalf("driver.Writer: unexpected error: %v", err)
	}
	err = writer.Commit(context.Background())
	if err != nil {
		t.Fatalf("writer.Commit: unexpected error: %v", err)
	}
	err = writer.Close()
	if err != nil {
		t.Fatalf("writer.Close: unexpected error: %v", err)
	}
	if writer.Size() != 0 {
		t.Fatalf("writer.Size: %d != 0", writer.Size())
	}
	readContents, err := driver.GetContent(ctx, filename)
	if err != nil {
		t.Fatalf("driver.GetContent: unexpected error: %v", err)
	}
	if len(readContents) != 0 {
		t.Fatalf("len(driver.GetContent(..)): %d != 0", len(readContents))
	}
}

// Test Committing a FileWriter after having written exactly
// defaultChunksize bytes.
func TestCommit(t *testing.T) {
	skipCheck(t)

	validRoot := t.TempDir()

	driver, err := gcsDriverConstructor(validRoot)
	if err != nil {
		t.Fatalf("unexpected error creating rooted driver: %v", err)
	}

	filename := "/test"
	ctx := dcontext.Background()

	contents := make([]byte, defaultChunkSize)
	writer, err := driver.Writer(ctx, filename, false)
	// nolint:errcheck
	defer driver.Delete(ctx, filename)
	if err != nil {
		t.Fatalf("driver.Writer: unexpected error: %v", err)
	}
	_, err = writer.Write(contents)
	if err != nil {
		t.Fatalf("writer.Write: unexpected error: %v", err)
	}
	err = writer.Commit(context.Background())
	if err != nil {
		t.Fatalf("writer.Commit: unexpected error: %v", err)
	}
	err = writer.Close()
	if err != nil {
		t.Fatalf("writer.Close: unexpected error: %v", err)
	}
	if writer.Size() != int64(len(contents)) {
		t.Fatalf("writer.Size: %d != %d", writer.Size(), len(contents))
	}
	readContents, err := driver.GetContent(ctx, filename)
	if err != nil {
		t.Fatalf("driver.GetContent: unexpected error: %v", err)
	}
	if len(readContents) != len(contents) {
		t.Fatalf("len(driver.GetContent(..)): %d != %d", len(readContents), len(contents))
	}
}

func TestRetry(t *testing.T) {
	skipCheck(t)

	assertError := func(expected string, observed error) {
		observedMsg := "<nil>"
		if observed != nil {
			observedMsg = observed.Error()
		}
		if observedMsg != expected {
			t.Fatalf("expected %v, observed %v\n", expected, observedMsg)
		}
	}

	err := retry(func() error {
		return &googleapi.Error{
			Code:    503,
			Message: "google api error",
		}
	})
	assertError("googleapi: Error 503: google api error", err)

	err = retry(func() error {
		return &googleapi.Error{
			Code:    404,
			Message: "google api error",
		}
	})
	assertError("googleapi: Error 404: google api error", err)

	err = retry(func() error {
		return fmt.Errorf("error")
	})
	assertError("error", err)
}

func TestEmptyRootList(t *testing.T) {
	skipCheck(t)

	validRoot := t.TempDir()

	rootedDriver, err := gcsDriverConstructor(validRoot)
	if err != nil {
		t.Fatalf("unexpected error creating rooted driver: %v", err)
	}

	emptyRootDriver, err := gcsDriverConstructor("")
	if err != nil {
		t.Fatalf("unexpected error creating empty root driver: %v", err)
	}

	slashRootDriver, err := gcsDriverConstructor("/")
	if err != nil {
		t.Fatalf("unexpected error creating slash root driver: %v", err)
	}

	filename := "/test"
	contents := []byte("contents")
	ctx := dcontext.Background()
	err = rootedDriver.PutContent(ctx, filename, contents)
	if err != nil {
		t.Fatalf("unexpected error creating content: %v", err)
	}
	defer func() {
		err := rootedDriver.Delete(ctx, filename)
		if err != nil {
			t.Fatalf("failed to remove %v due to %v\n", filename, err)
		}
	}()
	keys, err := emptyRootDriver.List(ctx, "/")
	if err != nil {
		t.Fatalf("unexpected error listing empty root content: %v", err)
	}
	for _, path := range keys {
		if !storagedriver.PathRegexp.MatchString(path) {
			t.Fatalf("unexpected string in path: %q != %q", path, storagedriver.PathRegexp)
		}
	}

	keys, err = slashRootDriver.List(ctx, "/")
	if err != nil {
		t.Fatalf("unexpected error listing slash root content: %v", err)
	}
	for _, path := range keys {
		if !storagedriver.PathRegexp.MatchString(path) {
			t.Fatalf("unexpected string in path: %q != %q", path, storagedriver.PathRegexp)
		}
	}
}

// TestMoveDirectory checks that moving a directory returns an error.
func TestMoveDirectory(t *testing.T) {
	skipCheck(t)

	validRoot := t.TempDir()

	driver, err := gcsDriverConstructor(validRoot)
	if err != nil {
		t.Fatalf("unexpected error creating rooted driver: %v", err)
	}

	ctx := dcontext.Background()
	contents := []byte("contents")
	// Create a regular file.
	err = driver.PutContent(ctx, "/parent/dir/foo", contents)
	if err != nil {
		t.Fatalf("unexpected error creating content: %v", err)
	}
	defer func() {
		err := driver.Delete(ctx, "/parent")
		if err != nil {
			t.Fatalf("failed to remove /parent due to %v\n", err)
		}
	}()

	err = driver.Move(ctx, "/parent/dir", "/parent/other")
	if err == nil {
		t.Fatal("Moving directory /parent/dir /parent/other should have return a non-nil error")
	}
}
