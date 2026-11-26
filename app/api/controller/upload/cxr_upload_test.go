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

package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/blob"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const cxrRepoRef = "root/repo"

var (
	errCxrBlob = errors.New("blob failure")
	// cxrPNG is the minimal prefix needed for the content type detection to report a PNG image.
	cxrPNG = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
)

// cxrBlobStore is an in-memory blob.Store.
type cxrBlobStore struct {
	blob.Store
	files     map[string][]byte
	signedURL string
	signedErr error
	uploadErr error
}

func (s *cxrBlobStore) Upload(_ context.Context, file io.Reader, filePath string) error {
	if s.uploadErr != nil {
		return s.uploadErr
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	s.files[filePath] = data
	return nil
}

func (s *cxrBlobStore) GetSignedURL(context.Context, string, time.Time, ...blob.SignURLOption) (string, error) {
	return s.signedURL, s.signedErr
}

func (s *cxrBlobStore) Download(_ context.Context, filePath string) (io.ReadCloser, error) {
	data, ok := s.files[filePath]
	if !ok {
		return nil, blob.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

type cxrEnv struct {
	*cxrBase
	repo  *types.Repository
	blobs *cxrBlobStore
}

func cxrNewEnv(ctx context.Context, t *testing.T) *cxrEnv {
	t.Helper()
	b := cxrNewBase(ctx, t)
	repo := b.createRepo(ctx, t, b.createSpace(ctx, t, "root"), "repo")
	return &cxrEnv{cxrBase: b, repo: repo, blobs: &cxrBlobStore{files: map[string][]byte{}}}
}

func (e *cxrEnv) controller(mode cxrAuthMode) *Controller {
	cfg := &types.Config{}
	cfg.BlobStore.MaxFileSize = 1234
	return NewController(cxrAuthorizer{mode: mode}, e.repoFinder, e.blobs, cfg)
}

func TestCxrUploadMaxFileSize(t *testing.T) {
	require.EqualValues(t, 1234, cxrNewEnv(t.Context(), t).controller(cxrAllow).GetMaxFileSize())
}

func TestCxrUploadAndDownload(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)

	res, err := c.Upload(ctx, e.session(), cxrRepoRef, bytes.NewReader(cxrPNG))
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(res.FilePath, ".png"), res.FilePath)
	require.Contains(t, e.blobs.files, getFileBucketPath(e.repo.ID, res.FilePath))

	// Without signed URL support the file is streamed from the store.
	e.blobs.signedErr = blob.ErrNotSupported
	url, file, err := c.Download(ctx, e.session(), cxrRepoRef, res.FilePath)
	require.NoError(t, err)
	require.Empty(t, url)
	data, err := io.ReadAll(file)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	require.Equal(t, cxrPNG, data)

	// With signed URL support only the URL is returned.
	e.blobs.signedErr = nil
	e.blobs.signedURL = "https://blobs.example.com/file"
	url, file, err = c.Download(ctx, e.session(), cxrRepoRef, res.FilePath)
	require.NoError(t, err)
	require.Equal(t, "https://blobs.example.com/file", url)
	require.Nil(t, file)
}

func TestCxrUploadErrors(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)

	_, err := c.Upload(ctx, e.session(), "", bytes.NewReader(cxrPNG))
	require.ErrorContains(t, err, "failed to acquire access to repo")
	_, err = c.Upload(ctx, e.session(), "root/nope", bytes.NewReader(cxrPNG))
	require.ErrorContains(t, err, "failed to acquire access to repo")
	_, err = e.controller(cxrDeny).Upload(ctx, e.session(), cxrRepoRef, bytes.NewReader(cxrPNG))
	require.ErrorContains(t, err, "failed to acquire access to repo")
	_, err = c.Upload(ctx, e.session(), cxrRepoRef, nil)
	require.ErrorContains(t, err, "No file provided")
	_, err = c.Upload(ctx, e.session(), cxrRepoRef, strings.NewReader("just some text"))
	require.ErrorContains(t, err, "only image and video files are supported")

	e.blobs.uploadErr = errCxrBlob
	_, err = c.Upload(ctx, e.session(), cxrRepoRef, bytes.NewReader(cxrPNG))
	require.ErrorIs(t, err, errCxrBlob)
}

func TestCxrUploadRepoStateBlocksUpload(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	_, err := e.db.ExecContext(ctx, "UPDATE repositories SET repo_state = $1 WHERE repo_id = $2",
		int(enum.RepoStateGitImport), e.repo.ID)
	require.NoError(t, err)

	_, err = e.controller(cxrAllow).Upload(ctx, e.session(), cxrRepoRef, bytes.NewReader(cxrPNG))
	require.Error(t, err)
}

func TestCxrDownloadErrors(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)

	_, _, err := c.Download(ctx, e.session(), "root/nope", "x.png")
	require.ErrorContains(t, err, "failed to acquire access to repo")

	e.blobs.signedErr = errCxrBlob
	_, _, err = c.Download(ctx, e.session(), cxrRepoRef, "x.png")
	require.ErrorIs(t, err, errCxrBlob)

	e.blobs.signedErr = blob.ErrNotSupported
	_, _, err = c.Download(ctx, e.session(), cxrRepoRef, "missing.png")
	require.ErrorContains(t, err, "failed to download file from blobstore")
}
