// Copyright 2026 Harness, Inc.
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
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// infGCSFake is a tiny in-process emulation of the parts of the GCS API the store uses.
type infGCSFake struct {
	mx       sync.Mutex
	objects  map[string]string
	requests []string
}

func infNewGCSFake() *infGCSFake {
	return &infGCSFake{objects: map[string]string{}}
}

func (f *infGCSFake) requestLog() []string {
	f.mx.Lock()
	defer f.mx.Unlock()
	return append([]string(nil), f.requests...)
}

func (f *infGCSFake) put(bucket, name, data string) {
	f.mx.Lock()
	defer f.mx.Unlock()
	f.objects[bucket+"/"+name] = data
}

func (f *infGCSFake) get(bucket, name string) (string, bool) {
	f.mx.Lock()
	defer f.mx.Unlock()
	v, ok := f.objects[bucket+"/"+name]
	return v, ok
}

func infJSONError(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = fmt.Fprintf(w, `{"error":{"code":%d,"message":"fake error"}}`, code)
}

func (f *infGCSFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mx.Lock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	f.mx.Unlock()

	path := r.URL.EscapedPath()
	switch {
	// upload: POST /upload/storage/v1/b/{bucket}/o?uploadType=multipart&name={name}
	case r.Method == http.MethodPost && strings.HasPrefix(path, "/upload/storage/v1/b/"):
		bucket := strings.Split(strings.TrimPrefix(path, "/upload/storage/v1/b/"), "/")[0]
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
			infJSONError(w, http.StatusBadRequest)
			return
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		var meta, content string
		for i := 0; ; i++ {
			part, partErr := mr.NextPart()
			if errors.Is(partErr, io.EOF) {
				break
			}
			if partErr != nil {
				infJSONError(w, http.StatusBadRequest)
				return
			}
			data, _ := io.ReadAll(part)
			if i == 0 {
				meta = string(data)
			} else {
				content = string(data)
			}
		}
		name := infExtractName(meta)
		f.put(bucket, name, content)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"name":%q,"bucket":%q}`, name, bucket)

	// rewrite: POST /storage/v1/b/{b}/o/{src}/rewriteTo/b/{b}/o/{dst}
	case r.Method == http.MethodPost && strings.Contains(path, "/rewriteTo/"):
		parts := strings.Split(strings.TrimPrefix(path, "/storage/v1/b/"), "/")
		srcBucket, src := parts[0], infUnescape(parts[2])
		dstBucket, dst := parts[5], infUnescape(parts[7])
		data, ok := f.get(srcBucket, src)
		if !ok {
			infJSONError(w, http.StatusNotFound)
			return
		}
		f.put(dstBucket, dst, data)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"done":true,"resource":{"name":%q,"bucket":%q}}`, dst, dstBucket)

	// delete: DELETE /storage/v1/b/{bucket}/o/{name}
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/storage/v1/b/"):
		parts := strings.Split(strings.TrimPrefix(path, "/storage/v1/b/"), "/")
		bucket, name := parts[0], infUnescape(parts[2])
		f.mx.Lock()
		_, ok := f.objects[bucket+"/"+name]
		delete(f.objects, bucket+"/"+name)
		f.mx.Unlock()
		if !ok {
			infJSONError(w, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	// download (XML API style): GET /{bucket}/{name}
	case r.Method == http.MethodGet:
		parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
		if len(parts) != 2 {
			infJSONError(w, http.StatusNotFound)
			return
		}
		data, ok := f.get(parts[0], infUnescape(parts[1]))
		if !ok {
			infJSONError(w, http.StatusNotFound)
			return
		}
		_, _ = io.WriteString(w, data)

	default:
		infJSONError(w, http.StatusNotImplemented)
	}
}

func infUnescape(s string) string {
	u, err := url.PathUnescape(s)
	if err != nil {
		return s
	}
	return u
}

func infExtractName(meta string) string {
	const marker = `"name":"`
	i := strings.Index(meta, marker)
	if i < 0 {
		return ""
	}
	rest := meta[i+len(marker):]
	before, _, _ := strings.Cut(rest, `"`)
	return before
}

func infNewGCSStore(t *testing.T) (*GCSStore, *infGCSFake) {
	t.Helper()
	fake := infNewGCSFake()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	t.Setenv("STORAGE_EMULATOR_HOST", strings.TrimPrefix(srv.URL, "http://"))

	store, err := NewGCSStore(context.Background(), Config{Provider: ProviderGCS, Bucket: "inf-bucket"})
	require.NoError(t, err)
	gcs, ok := store.(*GCSStore)
	require.True(t, ok)
	return gcs, fake
}

func TestInfGCSStore_NewWithDefaultCredentials(t *testing.T) {
	gcs, _ := infNewGCSStore(t)
	require.NotNil(t, gcs.cachedClient)
	require.True(t, gcs.tokenExpirationTime.IsZero(), "default credentials never expire")

	client, err := gcs.getClient(context.Background())
	require.NoError(t, err)
	require.Same(t, gcs.cachedClient, client)
}

func TestInfGCSStore_UploadDownloadRoundTrip(t *testing.T) {
	ctx := context.Background()
	gcs, fake := infNewGCSStore(t)

	require.NoError(t, gcs.Upload(ctx, strings.NewReader("hello gcs"), "dir/file.txt"))
	got, ok := fake.get("inf-bucket", "dir/file.txt")
	require.True(t, ok)
	require.Equal(t, "hello gcs", got)

	rc, err := gcs.Download(ctx, "dir/file.txt")
	require.NoError(t, err)
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	require.Equal(t, "hello gcs", string(data))
}

func TestInfGCSStore_DownloadNotFound(t *testing.T) {
	gcs, _ := infNewGCSStore(t)
	_, err := gcs.Download(context.Background(), "missing.txt")
	require.ErrorIs(t, err, ErrNotFound)
}

type infFailingReader struct{}

func (infFailingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestInfGCSStore_UploadReadFailureIsReported(t *testing.T) {
	gcs, fake := infNewGCSStore(t)
	err := gcs.Upload(context.Background(), infFailingReader{}, "broken.txt")
	require.ErrorContains(t, err, "failed to write file to GCS")
	require.Contains(t, fake.requestLog(), "DELETE /storage/v1/b/inf-bucket/o/broken.txt",
		"a best-effort cleanup of the partial object is attempted")
}

func TestInfGCSStore_Move(t *testing.T) {
	ctx := context.Background()
	gcs, fake := infNewGCSStore(t)
	fake.put("inf-bucket", "src.txt", "payload")

	require.NoError(t, gcs.Move(ctx, "src.txt", "dst.txt"))
	moved, ok := fake.get("inf-bucket", "dst.txt")
	require.True(t, ok)
	require.Equal(t, "payload", moved)
	_, ok = fake.get("inf-bucket", "src.txt")
	require.False(t, ok, "the source is removed after the copy")

	require.ErrorIs(t, gcs.Move(ctx, "missing.txt", "other.txt"), ErrNotFound)
}

func TestInfGCSStore_Delete(t *testing.T) {
	ctx := context.Background()
	gcs, fake := infNewGCSStore(t)
	fake.put("inf-bucket", "gone.txt", "x")

	require.NoError(t, gcs.Delete(ctx, "gone.txt"))
	_, ok := fake.get("inf-bucket", "gone.txt")
	require.False(t, ok)
	require.ErrorIs(t, gcs.Delete(ctx, "gone.txt"), ErrNotFound)
}

func TestInfGCSStore_GetSignedURLWithoutSigningCredentials(t *testing.T) {
	gcs, _ := infNewGCSStore(t)
	_, err := gcs.GetSignedURL(context.Background(), "file.txt", time.Now().Add(time.Hour),
		SignWithMethod(http.MethodPut), SignWithContentType("text/plain"))
	require.ErrorContains(t, err, `failed to create signed URL for file "file.txt"`)
}

func TestInfGCSStore_TokenRefreshFailures(t *testing.T) {
	ctx := context.Background()
	gcs, _ := infNewGCSStore(t)

	// an expired token on a store without impersonation target can't be refreshed.
	expired := &GCSStore{
		cachedClient:        gcs.cachedClient,
		config:              Config{Bucket: "inf-bucket", KeyPath: "/some/key.json", ImpersonationLifetime: time.Hour},
		tokenExpirationTime: time.Now().Add(-time.Minute),
	}
	_, err := expired.getClient(ctx)
	require.ErrorContains(t, err, "failed to refresh token")

	require.ErrorContains(t, expired.Upload(ctx, strings.NewReader("x"), "f"), "failed to retrieve latest client")
	_, err = expired.GetSignedURL(ctx, "f", time.Now().Add(time.Hour))
	require.ErrorContains(t, err, "failed to retrieve latest client")
	_, err = expired.Download(ctx, "f")
	require.ErrorContains(t, err, "failed to retrieve latest client")
	require.ErrorContains(t, expired.Move(ctx, "a", "b"), "failed to retrieve latest client")
	require.ErrorContains(t, expired.Delete(ctx, "f"), "failed to retrieve latest client")

	// a token that is still valid keeps the cached client.
	valid := &GCSStore{
		cachedClient:        gcs.cachedClient,
		config:              Config{Bucket: "inf-bucket", KeyPath: "/some/key.json"},
		tokenExpirationTime: time.Now().Add(time.Hour),
	}
	client, err := valid.getClient(ctx)
	require.NoError(t, err)
	require.Same(t, gcs.cachedClient, client)
}

func TestInfNewGCSStore_Validation(t *testing.T) {
	_, err := NewGCSStore(context.Background(), Config{Provider: ProviderGCS})
	require.ErrorContains(t, err, "bucket name is required")

	_, err = NewGCSStore(context.Background(), Config{
		Provider: ProviderGCS, Bucket: "b", KeyPath: "/definitely/not/there.json",
	})
	require.ErrorContains(t, err, "failed to create GCS client with service account key")
}

func TestInfCreateNewImpersonatedClient_RequiresTarget(t *testing.T) {
	_, err := createNewImpersonatedClient(context.Background(), Config{ImpersonationLifetime: time.Hour})
	require.ErrorContains(t, err, "failed to impersonate the client service account")
}
