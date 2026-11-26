//  Copyright 2023 Harness, Inc.
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

package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/driver/filesystem"
	"github.com/harness/gitness/registry/app/manifest"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

var (
	errRb5Boom = errors.New("rb5 boom")
	errRb5Pass = errors.New("rb5 pass through")
)

const (
	rb5Sha       = "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
	rb5Payload   = "rb5 payload"
	rb5BadDigest = digest.Digest("sha256:zz")
	rb5Prefix    = "acct"
	rb5Repo      = "repo"
)

// rb5Driver wraps a real filesystem driver and injects failures by method name.
type rb5Driver struct {
	driver.StorageDriver
	mu          sync.Mutex
	seq         map[string][]error
	fail        map[string]error
	redirectURL string
	dirStat     bool
	writerFault rb5WriterFault
}

type rb5WriterFault struct {
	write  error
	closeE error
	commit error
	cancel error
}

func rb5NewDriver(t *testing.T) *rb5Driver {
	t.Helper()
	fs, err := filesystem.FromParameters(map[string]any{
		"rootdirectory": t.TempDir(),
		"maxthreads":    uint64(25),
	})
	require.NoError(t, err)
	return &rb5Driver{StorageDriver: fs, seq: map[string][]error{}, fail: map[string]error{}}
}

// inject returns the error to inject for the next call of the named method.
func (d *rb5Driver) inject(name string) (error, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if q := d.seq[name]; len(q) > 0 {
		d.seq[name] = q[1:]
		if errors.Is(q[0], errRb5Pass) {
			return nil, false
		}
		return q[0], true
	}
	if err, ok := d.fail[name]; ok {
		return err, true
	}
	return nil, false
}

func rb5NotFound() error { return driver.PathNotFoundError{Path: "/missing", DriverName: "rb5"} }

func (d *rb5Driver) Stat(ctx context.Context, path string) (driver.FileInfo, error) {
	if err, ok := d.inject("Stat"); ok {
		return nil, err
	}
	if d.dirStat {
		return driver.FileInfoInternal{FileInfoFields: driver.FileInfoFields{Path: path, IsDir: true}}, nil
	}
	return d.StorageDriver.Stat(ctx, path)
}

func (d *rb5Driver) Move(ctx context.Context, src, dst string) error {
	if err, ok := d.inject("Move"); ok {
		return err
	}
	return d.StorageDriver.Move(ctx, src, dst)
}

func (d *rb5Driver) Delete(ctx context.Context, path string) error {
	if err, ok := d.inject("Delete"); ok {
		return err
	}
	return d.StorageDriver.Delete(ctx, path)
}

func (d *rb5Driver) PutContent(ctx context.Context, path string, content []byte) error {
	if err, ok := d.inject("PutContent"); ok {
		return err
	}
	return d.StorageDriver.PutContent(ctx, path, content)
}

func (d *rb5Driver) GetContent(ctx context.Context, path string) ([]byte, error) {
	if err, ok := d.inject("GetContent"); ok {
		return nil, err
	}
	return d.StorageDriver.GetContent(ctx, path)
}

func (d *rb5Driver) Reader(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	if err, ok := d.inject("Reader"); ok {
		return nil, err
	}
	return d.StorageDriver.Reader(ctx, path, offset)
}

func (d *rb5Driver) List(ctx context.Context, path string) ([]string, error) {
	if err, ok := d.inject("List"); ok {
		return nil, err
	}
	return d.StorageDriver.List(ctx, path)
}

func (d *rb5Driver) RedirectURL(ctx context.Context, method, path, filename string) (string, error) {
	if err, ok := d.inject("RedirectURL"); ok {
		return "", err
	}
	if d.redirectURL != "" {
		return d.redirectURL, nil
	}
	return d.StorageDriver.RedirectURL(ctx, method, path, filename)
}

func (d *rb5Driver) Writer(ctx context.Context, path string, appendMode bool) (driver.FileWriter, error) {
	if err, ok := d.inject("Writer"); ok {
		return nil, err
	}
	fw, err := d.StorageDriver.Writer(ctx, path, appendMode)
	if err != nil {
		return nil, err
	}
	return &rb5Writer{FileWriter: fw, fault: &d.writerFault}, nil
}

type rb5Writer struct {
	driver.FileWriter
	fault *rb5WriterFault
}

func (w *rb5Writer) Write(p []byte) (int, error) {
	if w.fault.write != nil {
		return 0, w.fault.write
	}
	return w.FileWriter.Write(p)
}

func (w *rb5Writer) Close() error {
	if w.fault.closeE != nil {
		return w.fault.closeE
	}
	return w.FileWriter.Close()
}

func (w *rb5Writer) Commit(ctx context.Context) error {
	if w.fault.commit != nil {
		return w.fault.commit
	}
	return w.FileWriter.Commit(ctx)
}

func (w *rb5Writer) Cancel(ctx context.Context) error {
	if w.fault.cancel != nil {
		return w.fault.cancel
	}
	return w.FileWriter.Cancel(ctx)
}

type rb5ErrReader struct{}

func (rb5ErrReader) Read([]byte) (int, error) { return 0, errRb5Boom }

func rb5GlobalStore(fd *rb5Driver) *globalBlobStore {
	return &globalBlobStore{bucketKey: "rb5-bucket", driver: fd, ctx: context.Background()}
}

func rb5OciStore(fd *rb5Driver) *ociBlobStore {
	return &ociBlobStore{
		repoKey: rb5Repo, driver: fd, ctx: context.Background(), pathFn: PathFn, rootParentRef: rb5Prefix,
	}
}

func rb5Dgst() digest.Digest { return digest.FromBytes([]byte(rb5Payload)) }

// rb5BlobReader is the read side shared by the OCI and global blob stores.
type rb5BlobReader interface {
	ServeBlobInternal(
		ctx context.Context, pathPrefix string, dgst digest.Digest, headers map[string]string, method string,
	) (*FileReader, string, int64, error)
	GetBlobInternal(ctx context.Context, pathPrefix string, dgst digest.Digest) (*FileReader, int64, error)
	Stat(ctx context.Context, pathPrefix string, dgst digest.Digest) (manifest.Descriptor, error)
	Get(ctx context.Context, pathPrefix string, dgst digest.Digest) ([]byte, error)
	Open(ctx context.Context, pathPrefix string, dgst digest.Digest) (io.ReadSeekCloser, error)
	Put(ctx context.Context, pathPrefix string, p []byte) (manifest.Descriptor, error)
}

func rb5ReaderStores(fd *rb5Driver) map[string]rb5BlobReader {
	return map[string]rb5BlobReader{"oci": rb5OciStore(fd), "global": rb5GlobalStore(fd)}
}

func TestRb5BlobStoreReadErrors(t *testing.T) {
	ctx := context.Background()
	dgst := rb5Dgst()

	for _, kind := range []string{"oci", "global"} {
		// run builds a fresh driver and store, stores the payload and applies the failure.
		run := func(t *testing.T, setup func(fd *rb5Driver)) (rb5BlobReader, *rb5Driver) {
			t.Helper()
			fd := rb5NewDriver(t)
			store := rb5ReaderStores(fd)[kind]
			_, err := store.Put(ctx, rb5Prefix, []byte(rb5Payload))
			require.NoError(t, err)
			setup(fd)
			return store, fd
		}

		t.Run(kind+" invalid digest", func(t *testing.T) {
			store, _ := run(t, func(*rb5Driver) {})
			_, err := store.Stat(ctx, rb5Prefix, rb5BadDigest)
			require.Error(t, err)
			_, err = store.Get(ctx, rb5Prefix, rb5BadDigest)
			require.Error(t, err)
			_, err = store.Open(ctx, rb5Prefix, rb5BadDigest)
			require.Error(t, err)
			_, _, err = store.GetBlobInternal(ctx, rb5Prefix, rb5BadDigest)
			require.Error(t, err)
			_, _, _, err = store.ServeBlobInternal(ctx, rb5Prefix, rb5BadDigest, map[string]string{}, http.MethodGet)
			require.Error(t, err)
		})

		t.Run(kind+" stat failure", func(t *testing.T) {
			store, _ := run(t, func(fd *rb5Driver) { fd.fail["Stat"] = errRb5Boom })
			_, err := store.Stat(ctx, rb5Prefix, dgst)
			require.ErrorIs(t, err, errRb5Boom)
			_, err = store.Get(ctx, rb5Prefix, dgst)
			require.ErrorIs(t, err, errRb5Boom)
			_, err = store.Open(ctx, rb5Prefix, dgst)
			require.ErrorIs(t, err, errRb5Boom)
			_, _, err = store.GetBlobInternal(ctx, rb5Prefix, dgst)
			require.ErrorIs(t, err, errRb5Boom)
			_, _, _, err = store.ServeBlobInternal(ctx, rb5Prefix, dgst, map[string]string{}, http.MethodGet)
			require.ErrorIs(t, err, errRb5Boom)
			_, err = store.Put(ctx, rb5Prefix, []byte(rb5Payload))
			require.ErrorIs(t, err, errRb5Boom)
		})

		t.Run(kind+" stat not found and directory", func(t *testing.T) {
			store, _ := run(t, func(fd *rb5Driver) { fd.fail["Stat"] = rb5NotFound() })
			_, err := store.Stat(ctx, rb5Prefix, dgst)
			require.ErrorIs(t, err, ErrBlobUnknown)

			store, _ = run(t, func(fd *rb5Driver) { fd.dirStat = true })
			_, err = store.Stat(ctx, rb5Prefix, dgst)
			require.ErrorIs(t, err, ErrBlobUnknown)
		})

		t.Run(kind+" get content not found", func(t *testing.T) {
			store, _ := run(t, func(fd *rb5Driver) { fd.fail["Reader"] = rb5NotFound() })
			_, err := store.Get(ctx, rb5Prefix, dgst)
			require.ErrorIs(t, err, ErrBlobUnknown)
		})

		t.Run(kind+" get content error", func(t *testing.T) {
			store, _ := run(t, func(fd *rb5Driver) { fd.fail["Reader"] = errRb5Boom })
			_, err := store.Get(ctx, rb5Prefix, dgst)
			require.ErrorIs(t, err, errRb5Boom)
		})

		t.Run(kind+" put content error", func(t *testing.T) {
			fd := rb5NewDriver(t)
			fd.seq["Stat"] = []error{rb5NotFound()}
			fd.fail["PutContent"] = errRb5Boom
			_, err := rb5ReaderStores(fd)[kind].Put(ctx, rb5Prefix, []byte(rb5Payload))
			require.ErrorIs(t, err, errRb5Boom)
		})

		t.Run(kind+" serve redirect", func(t *testing.T) {
			store, fd := run(t, func(*rb5Driver) {})
			switch s := store.(type) {
			case *ociBlobStore:
				s.redirect = true
			case *globalBlobStore:
				s.redirect = true
			}

			fd.fail["RedirectURL"] = errRb5Boom
			_, _, _, err := store.ServeBlobInternal(ctx, rb5Prefix, dgst, map[string]string{}, http.MethodGet)
			require.ErrorIs(t, err, errRb5Boom)
			delete(fd.fail, "RedirectURL")

			fd.redirectURL = "https://example.com/blob"
			reader, url, size, err := store.ServeBlobInternal(ctx, rb5Prefix, dgst, map[string]string{}, http.MethodGet)
			require.NoError(t, err)
			require.Nil(t, reader)
			require.Equal(t, "https://example.com/blob", url)
			require.Equal(t, int64(len(rb5Payload)), size)

			// The filesystem driver cannot redirect, so serving falls back to the content.
			fd.redirectURL = ""
			fd.fail["RedirectURL"] = nil
			delete(fd.fail, "RedirectURL")
			headers := map[string]string{}
			reader, url, _, err = store.ServeBlobInternal(ctx, rb5Prefix, dgst, headers, http.MethodGet)
			require.NoError(t, err)
			require.Empty(t, url)
			require.NotNil(t, reader)
			require.Equal(t, "application/octet-stream", headers[HeaderContentType])
			require.Equal(t, dgst.String(), headers[HeaderDockerContentDigest])
			require.Equal(t, `"`+dgst.String()+`"`, headers[HeaderEtag])
			require.Equal(t, "11", headers[HeaderContentLength])
			_ = reader.Close()
		})
	}
}

func TestRb5GlobalStoreGenericReads(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid digest", func(t *testing.T) {
		bs := rb5GlobalStore(rb5NewDriver(t))
		_, err := bs.GetV2NoRedirect(ctx, "", "not-a-sha", 1)
		require.Error(t, err)
		_, _, err = bs.GetGeneric(ctx, 1, "f", "", "not-a-sha")
		require.Error(t, err)
		_, err = bs.StatByDigest(ctx, "", "not-a-sha")
		require.Error(t, err)
		require.Error(t, bs.move(ctx, "id", "not-a-sha"))
	})

	t.Run("stat by digest", func(t *testing.T) {
		fd := rb5NewDriver(t)
		bs := rb5GlobalStore(fd)
		_, err := bs.StatByDigest(ctx, "", rb5Sha)
		require.Error(t, err)

		path, err := GlobalPathFn(digest.NewDigestFromEncoded(digest.SHA256, rb5Sha))
		require.NoError(t, err)
		require.NoError(t, fd.PutContent(ctx, path, []byte("abcd")))
		size, err := bs.StatByDigest(ctx, "", rb5Sha)
		require.NoError(t, err)
		require.Equal(t, int64(4), size)
	})

	t.Run("generic redirect", func(t *testing.T) {
		fd := rb5NewDriver(t)
		bs := rb5GlobalStore(fd)
		bs.redirect = true

		fd.fail["RedirectURL"] = errRb5Boom
		_, _, err := bs.GetGeneric(ctx, 1, "f", "", rb5Sha)
		require.ErrorIs(t, err, errRb5Boom)
		delete(fd.fail, "RedirectURL")

		fd.redirectURL = "https://example.com/generic"
		reader, url, err := bs.GetGeneric(ctx, 1, "f", "", rb5Sha)
		require.NoError(t, err)
		require.Nil(t, reader)
		require.Equal(t, "https://example.com/generic", url)

		fd.redirectURL = ""
		reader, url, err = bs.GetGeneric(ctx, 1, "f", "", rb5Sha)
		require.NoError(t, err)
		require.Empty(t, url)
		require.NotNil(t, reader)
	})

	t.Run("no redirect", func(t *testing.T) {
		bs := rb5GlobalStore(rb5NewDriver(t))
		reader, err := bs.GetV2NoRedirect(ctx, "", "sha256:"+rb5Sha, 1)
		require.NoError(t, err)
		require.NotNil(t, reader)
	})

	t.Run("write copy failure", func(t *testing.T) {
		bs := rb5GlobalStore(rb5NewDriver(t))
		w, err := bs.CreateGeneric(ctx, "root")
		require.NoError(t, err)
		_, err = bs.Write(ctx, w, nil, rb5ErrReader{})
		require.ErrorIs(t, err, errRb5Boom)

		info, err := bs.Write(ctx, w, nil, strings.NewReader("abc"))
		require.NoError(t, err)
		require.Equal(t, int64(3), info.Size)
	})

	t.Run("accessors", func(t *testing.T) {
		bs := rb5GlobalStore(rb5NewDriver(t))
		require.Equal(t, "rb5-bucket", string(bs.BucketKey()))
		require.Empty(t, bs.Path())
	})
}

func TestRb5BlobStoreWriterCreationErrors(t *testing.T) {
	ctx := context.Background()

	fd := rb5NewDriver(t)
	fd.fail["Writer"] = errRb5Boom
	gs := rb5GlobalStore(fd)
	_, err := gs.Create(ctx)
	require.ErrorIs(t, err, errRb5Boom)
	_, err = gs.CreateGeneric(ctx, "root")
	require.ErrorIs(t, err, errRb5Boom)
	_, err = gs.Resume(ctx, "upload-id")
	require.ErrorIs(t, err, errRb5Boom)

	os := rb5OciStore(fd)
	_, err = os.Create(ctx)
	require.ErrorIs(t, err, errRb5Boom)
	_, err = os.Resume(ctx, "upload-id")
	require.ErrorIs(t, err, errRb5Boom)

	gen := &genericBlobStore{driver: fd, rootParentRef: rb5Prefix}
	_, err = gen.CreateGeneric(ctx, "root")
	require.ErrorIs(t, err, errRb5Boom)

	// Invalid upload ids cannot be turned into paths.
	delete(fd.fail, "Writer")
	_, err = os.Resume(ctx, "bad/../id")
	require.NoError(t, err)
}

func TestRb5GenericBlobStoreErrors(t *testing.T) {
	ctx := context.Background()
	fd := rb5NewDriver(t)
	bs := &genericBlobStore{driver: fd, rootParentRef: rb5Prefix}

	_, err := bs.StatByDigest(ctx, "root", rb5Sha)
	require.Error(t, err)

	fd.fail["Move"] = errRb5Boom
	require.ErrorIs(t, bs.move(ctx, "root", "id", rb5Sha), errRb5Boom)
	delete(fd.fail, "Move")

	fd.fail["Delete"] = errRb5Boom
	require.ErrorIs(t, bs.Delete(ctx, "/some/path"), errRb5Boom)
	delete(fd.fail, "Delete")
	require.NoError(t, fd.PutContent(ctx, "/some/path", []byte("x")))
	require.NoError(t, bs.Delete(ctx, "/some/path"))

	bs.redirect = true
	fd.fail["RedirectURL"] = errRb5Boom
	_, _, err = bs.GetGeneric(ctx, 1, "f", "root", rb5Sha)
	require.ErrorIs(t, err, errRb5Boom)
	delete(fd.fail, "RedirectURL")
	fd.redirectURL = "https://example.com/g"
	_, url, err := bs.GetGeneric(ctx, 1, "f", "root", rb5Sha)
	require.NoError(t, err)
	require.Equal(t, "https://example.com/g", url)
	fd.redirectURL = ""
	reader, url, err := bs.GetGeneric(ctx, 1, "f", "root", rb5Sha)
	require.NoError(t, err)
	require.Empty(t, url)
	require.NotNil(t, reader)

	w, err := bs.CreateGeneric(ctx, "root")
	require.NoError(t, err)
	_, err = bs.Write(ctx, w, nil, rb5ErrReader{})
	require.Error(t, err)
}

func TestRb5FileReader(t *testing.T) {
	ctx := context.Background()
	fd := rb5NewDriver(t)
	require.NoError(t, fd.PutContent(ctx, "/rb5/file", []byte("0123456789")))

	t.Run("missing file reads as empty", func(t *testing.T) {
		fr, err := NewFileReader(ctx, fd, "/rb5/missing", 5)
		require.NoError(t, err)
		_, err = fr.Read(make([]byte, 4))
		require.Error(t, err)
	})

	t.Run("reader error is returned", func(t *testing.T) {
		fd.fail["Reader"] = errRb5Boom
		defer delete(fd.fail, "Reader")
		fr, err := NewFileReader(ctx, fd, "/rb5/file", 10)
		require.NoError(t, err)
		_, err = fr.Read(make([]byte, 4))
		require.ErrorIs(t, err, errRb5Boom)
	})

	t.Run("seek variants and close", func(t *testing.T) {
		fr, err := NewFileReader(ctx, fd, "/rb5/file", 10)
		require.NoError(t, err)

		buf := make([]byte, 4)
		n, err := fr.Read(buf)
		require.NoError(t, err)
		require.Equal(t, "0123", string(buf[:n]))

		pos, err := fr.Seek(2, io.SeekCurrent)
		require.NoError(t, err)
		require.Equal(t, int64(6), pos)
		n, err = fr.Read(buf)
		require.True(t, err == nil || errors.Is(err, io.EOF))
		require.Equal(t, "6789", string(buf[:n]))

		pos, err = fr.Seek(-3, io.SeekEnd)
		require.NoError(t, err)
		require.Equal(t, int64(7), pos)

		pos, err = fr.Seek(1, io.SeekStart)
		require.NoError(t, err)
		require.Equal(t, int64(1), pos)

		_, err = fr.Seek(-5, io.SeekStart)
		require.Error(t, err)

		_ = fr.Close()
		require.Error(t, fr.Close())
		_, err = fr.Read(buf)
		require.Error(t, err)
		_, err = fr.Seek(0, io.SeekStart)
		require.Error(t, err)
	})
}
