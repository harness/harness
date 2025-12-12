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
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/driver/base"
	"github.com/harness/gitness/registry/app/driver/factory"

	"cloud.google.com/go/storage"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const (
	driverName     = "gcs"
	dummyProjectID = "<unknown>"

	minChunkSize          = 256 * 1024
	defaultChunkSize      = 16 * 1024 * 1024
	defaultMaxConcurrency = 50
	minConcurrency        = 25

	uploadSessionContentType = "application/x-docker-upload-session"
	blobContentType          = "application/octet-stream"

	maxTries = 5
)

var rangeHeader = regexp.MustCompile(`^bytes=([0-9])+-([0-9]+)$`)

var _ storagedriver.FileWriter = &writer{}

// driverParameters is a struct that encapsulates all of the driver parameters after all values have been set.
type driverParameters struct {
	bucket        string
	email         string
	privateKey    []byte
	client        *http.Client
	rootDirectory string
	chunkSize     int
	gcs           *storage.Client

	// maxConcurrency limits the number of concurrent driver operations
	// to GCS, which ultimately increases reliability of many simultaneous
	// pushes by ensuring we aren't DoSing our own server with many
	// connections.
	maxConcurrency uint64
}

func init() {
	factory.Register(driverName, &gcsDriverFactory{})
}

// gcsDriverFactory implements the factory.StorageDriverFactory interface.
type gcsDriverFactory struct{}

// Create StorageDriver from parameters.
func (factory *gcsDriverFactory) Create(
	ctx context.Context,
	parameters map[string]interface{},
) (storagedriver.StorageDriver, error) {
	return FromParameters(ctx, parameters)
}

var _ storagedriver.StorageDriver = &driver{}

// driver is a storagedriver.StorageDriver implementation backed by GCS
// Objects are stored at absolute keys in the provided bucket.
type driver struct {
	client        *http.Client
	bucket        *storage.BucketHandle
	email         string
	privateKey    []byte
	rootDirectory string
	chunkSize     int
}

// Wrapper wraps `driver` with a throttler, ensuring that no more than N
// GCS actions can occur concurrently. The default limit is 75.
type Wrapper struct {
	baseEmbed
}

type baseEmbed struct {
	base.Base
}

// FromParameters constructs a new Driver with a given parameters map.
// Required parameters:
// - bucket.
func FromParameters(ctx context.Context, parameters map[string]interface{}) (storagedriver.StorageDriver, error) {
	bucket, ok := parameters["bucket"]
	if !ok || fmt.Sprint(bucket) == "" {
		return nil, fmt.Errorf("no bucket parameter provided")
	}

	rootDirectory, ok := parameters["rootdirectory"]
	if !ok {
		rootDirectory = ""
	}

	chunkSize := defaultChunkSize
	chunkSizeParam, ok := parameters["chunksize"]
	if ok {
		switch v := chunkSizeParam.(type) {
		case string:
			vv, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("chunksize must be an integer, %v invalid", chunkSizeParam)
			}
			chunkSize = vv
		case int, uint, int32, uint32, uint64, int64:
			chunkSize = int(reflect.ValueOf(v).Convert(reflect.TypeOf(chunkSize)).Int())
		default:
			return nil, fmt.Errorf("invalid valud for chunksize: %#v", chunkSizeParam)
		}

		if chunkSize < minChunkSize {
			return nil, fmt.Errorf("chunksize %#v must be larger than or equal to %d", chunkSize, minChunkSize)
		}

		if chunkSize%minChunkSize != 0 {
			return nil, fmt.Errorf("chunksize should be a multiple of %d", minChunkSize)
		}
	}

	var ts oauth2.TokenSource
	jwtConf := new(jwt.Config)
	var err error
	var gcs *storage.Client
	var options []option.ClientOption
	//nolint:nestif
	if keyfile, ok := parameters["keyfile"]; ok {
		jsonKey, err := os.ReadFile(fmt.Sprint(keyfile))
		if err != nil {
			return nil, err
		}
		jwtConf, err = google.JWTConfigFromJSON(jsonKey, storage.ScopeFullControl)
		if err != nil {
			return nil, err
		}
		ts = jwtConf.TokenSource(ctx)
		options = append(options, option.WithCredentialsFile(fmt.Sprint(keyfile)))
	} else if credentials, ok := parameters["credentials"]; ok {
		credentialMap, ok := credentials.(map[interface{}]interface{})
		if !ok {
			return nil, fmt.Errorf("the credentials were not specified in the correct format")
		}

		stringMap := map[string]interface{}{}
		for k, v := range credentialMap {
			key, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("one of the credential keys was not a string: %s", fmt.Sprint(k))
			}
			stringMap[key] = v
		}

		data, err := json.Marshal(stringMap)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal gcs credentials to json")
		}

		jwtConf, err = google.JWTConfigFromJSON(data, storage.ScopeFullControl)
		if err != nil {
			return nil, err
		}
		ts = jwtConf.TokenSource(ctx)
		options = append(options, option.WithCredentialsJSON(data))
	} else {
		var err error
		// DefaultTokenSource is a convenience method. It first calls FindDefaultCredentials,
		// then uses the credentials to construct an http.Client or an oauth2.TokenSource.
		// https://pkg.go.dev/golang.org/x/oauth2/google#hdr-Credentials
		ts, err = google.DefaultTokenSource(ctx, storage.ScopeFullControl)
		if err != nil {
			return nil, err
		}
	}

	if userAgent, ok := parameters["useragent"]; ok {
		if ua, ok := userAgent.(string); ok && ua != "" {
			options = append(options, option.WithUserAgent(ua))
		}
	}

	gcs, err = storage.NewClient(ctx, options...)
	if err != nil {
		return nil, err
	}

	maxConcurrency, err := base.GetLimitFromParameter(parameters["maxconcurrency"], minConcurrency,
		defaultMaxConcurrency)
	if err != nil {
		return nil, fmt.Errorf("maxconcurrency config error: %w", err)
	}

	params := driverParameters{
		bucket:         fmt.Sprint(bucket),
		rootDirectory:  fmt.Sprint(rootDirectory),
		email:          jwtConf.Email,
		privateKey:     jwtConf.PrivateKey,
		client:         oauth2.NewClient(ctx, ts),
		chunkSize:      chunkSize,
		maxConcurrency: maxConcurrency,
		gcs:            gcs,
	}

	return New(ctx, params)
}

// New constructs a new driver.
func New(_ context.Context, params driverParameters) (storagedriver.StorageDriver, error) {
	rootDirectory := strings.Trim(params.rootDirectory, "/")
	if rootDirectory != "" {
		rootDirectory += "/"
	}
	if params.chunkSize <= 0 || params.chunkSize%minChunkSize != 0 {
		return nil, fmt.Errorf("invalid chunksize: %d is not a positive multiple of %d", params.chunkSize, minChunkSize)
	}
	d := &driver{
		bucket:        params.gcs.Bucket(params.bucket),
		rootDirectory: rootDirectory,
		email:         params.email,
		privateKey:    params.privateKey,
		client:        params.client,
		chunkSize:     params.chunkSize,
	}

	return &Wrapper{
		baseEmbed: baseEmbed{
			Base: base.Base{
				StorageDriver: base.NewRegulator(d, params.maxConcurrency),
			},
		},
	}, nil
}

// Implement the storagedriver.StorageDriver interface

func (d *driver) Name() string {
	return driverName
}

// GetContent retrieves the content stored at "path" as a []byte.
// This should primarily be used for small objects.
func (d *driver) GetContent(ctx context.Context, path string) ([]byte, error) {
	r, err := d.bucket.Object(d.pathToKey(path)).NewReader(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, storagedriver.PathNotFoundError{Path: path}
		}
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}

// PutContent stores the []byte content at a location designated by "path".
// This should primarily be used for small objects.
func (d *driver) PutContent(ctx context.Context, path string, contents []byte) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	object := d.bucket.Object(d.pathToKey(path))
	err := d.putContent(ctx, object, contents, blobContentType, nil)
	if err != nil {
		return err
	}
	return nil
}

// Reader retrieves an io.ReadCloser for the content stored at "path"
// with a given byte offset.
// May be used to resume reading a stream by providing a nonzero offset.
func (d *driver) Reader(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	obj := d.bucket.Object(d.pathToKey(path))
	// NOTE(milosgajdos): If length is negative, the object is read until the end
	// See: https://pkg.go.dev/cloud.google.com/go/storage#ObjectHandle.NewRangeReader
	r, err := obj.NewRangeReader(ctx, offset, -1)
	if err != nil { //nolint:nestif
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, storagedriver.PathNotFoundError{Path: path}
		}
		var status *googleapi.Error
		if errors.As(err, &status) {
			switch status.Code {
			case http.StatusNotFound:
				return nil, storagedriver.PathNotFoundError{Path: path}
			case http.StatusRequestedRangeNotSatisfiable:
				attrs, err := obj.Attrs(ctx)
				if err != nil {
					return nil, err
				}
				if offset == attrs.Size {
					return io.NopCloser(bytes.NewReader([]byte{})), nil
				}
				return nil, storagedriver.InvalidOffsetError{Path: path, Offset: offset}
			}
		}
		return nil, err
	}
	if r.Attrs.ContentType == uploadSessionContentType {
		r.Close()
		return nil, storagedriver.PathNotFoundError{Path: path}
	}
	return r, nil
}

// Writer returns a FileWriter which will store the content written to it
// at the location designated by "path" after the call to Commit.
func (d *driver) Writer(ctx context.Context, path string, appendMode bool) (storagedriver.FileWriter, error) {
	w := &writer{
		ctx:    ctx,
		driver: d,
		object: d.bucket.Object(d.pathToKey(path)),
		buffer: make([]byte, d.chunkSize),
	}

	if appendMode {
		err := w.init(ctx)
		if err != nil {
			return nil, err
		}
	}
	return w, nil
}

type writer struct {
	ctx        context.Context
	object     *storage.ObjectHandle
	driver     *driver
	size       int64
	offset     int64
	closed     bool
	cancelled  bool
	committed  bool
	sessionURI string
	buffer     []byte
	buffSize   int
}

// Cancel removes any written content from this FileWriter.
func (w *writer) Cancel(ctx context.Context) error {
	w.closed = true
	w.cancelled = true

	err := w.object.Delete(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			err = nil
		}
	}
	return err
}

func (w *writer) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	err := w.writeChunk(w.ctx)
	if err != nil {
		return err
	}

	// Copy the remaining bytes from the buffer to the upload session
	// Normally buffSize will be smaller than minChunkSize. However, in the
	// unlikely event that the upload session failed to start, this number could be higher.
	// In this case we can safely clip the remaining bytes to the minChunkSize
	if w.buffSize > minChunkSize {
		w.buffSize = minChunkSize
	}

	// commit the writes by updating the upload session
	metadata := map[string]string{
		"Session-URI": w.sessionURI,
		"Offset":      strconv.FormatInt(w.offset, 10),
	}
	err = retry(func() error {
		err := w.driver.putContent(w.ctx, w.object, w.buffer[0:w.buffSize], uploadSessionContentType, metadata)
		if err != nil {
			return err
		}
		w.size = w.offset + int64(w.buffSize)
		w.buffSize = 0
		return nil
	})
	return err
}

func (d *driver) putContent(
	ctx context.Context,
	obj *storage.ObjectHandle,
	content []byte,
	contentType string,
	metadata map[string]string,
) error {
	wc := obj.NewWriter(ctx)
	wc.Metadata = metadata
	wc.ContentType = contentType
	wc.ChunkSize = d.chunkSize

	if _, err := bytes.NewReader(content).WriteTo(wc); err != nil {
		return err
	}
	// NOTE(milosgajdos): Apparently it's posisble to to upload 0-byte content to GCS.
	// Setting MD5 on the Writer helps to prevent presisting that data.
	// If set, the uploaded data is rejected if its MD5 hash does not match this field.
	// See: https://pkg.go.dev/cloud.google.com/go/storage#ObjectAttrs
	h := md5.New() //nolint:gosec
	h.Write(content)
	wc.MD5 = h.Sum(nil)

	return wc.Close()
}

// Commit flushes all content written to this FileWriter and makes it
// available for future calls to StorageDriver.GetContent and
// StorageDriver.Reader.
func (w *writer) Commit(ctx context.Context) error {
	if w.closed {
		return fmt.Errorf("already closed")
	}
	w.closed = true

	// no session started yet just perform a simple upload
	if w.sessionURI == "" {
		err := retry(func() error {
			err := w.driver.putContent(ctx, w.object, w.buffer[0:w.buffSize], blobContentType, nil)
			if err != nil {
				return err
			}
			w.committed = true
			w.size = w.offset + int64(w.buffSize)
			w.buffSize = 0
			return nil
		})
		return err
	}
	size := w.offset + int64(w.buffSize)
	var written int
	// loop must be performed at least once to ensure the file is committed even when
	// the buffer is empty
	for {
		n, err := w.putChunk(ctx, w.sessionURI, w.buffer[written:w.buffSize], w.offset, size)
		written += int(n)
		w.offset += n
		w.size = w.offset
		if err != nil {
			w.buffSize = copy(w.buffer, w.buffer[written:w.buffSize])
			return err
		}
		if written == w.buffSize {
			break
		}
	}
	w.committed = true
	w.buffSize = 0
	return nil
}

func (w *writer) writeChunk(ctx context.Context) error {
	var err error
	// chunks can be uploaded only in multiples of minChunkSize
	// chunkSize is a multiple of minChunkSize less than or equal to buffSize
	chunkSize := w.buffSize - (w.buffSize % minChunkSize)
	if chunkSize == 0 {
		return nil
	}
	// if their is no sessionURI yet, obtain one by starting the session
	if w.sessionURI == "" {
		w.sessionURI, err = w.newSession()
	}
	if err != nil {
		return err
	}
	n, err := w.putChunk(ctx, w.sessionURI, w.buffer[0:chunkSize], w.offset, -1)
	w.offset += n
	if w.offset > w.size {
		w.size = w.offset
	}
	// shift the remaining bytes to the start of the buffer
	w.buffSize = copy(w.buffer, w.buffer[int(n):w.buffSize])

	return err
}

func (w *writer) Write(p []byte) (int, error) {
	if w.closed {
		return 0, fmt.Errorf("already closed")
	} else if w.cancelled {
		return 0, fmt.Errorf("already cancelled")
	}

	var (
		written int
		err     error
	)

	for written < len(p) {
		n := copy(w.buffer[w.buffSize:], p[written:])
		w.buffSize += n
		if w.buffSize == cap(w.buffer) {
			err = w.writeChunk(w.ctx)
			if err != nil {
				break
			}
		}
		written += n
	}
	w.size = w.offset + int64(w.buffSize)
	return written, err
}

// Size returns the number of bytes written to this FileWriter.
func (w *writer) Size() int64 {
	return w.size
}

func (w *writer) init(ctx context.Context) error {
	attrs, err := w.object.Attrs(ctx)
	if err != nil {
		return err
	}

	// NOTE(milosgajdos): when PUSH abruptly finishes by
	// calling a single commit and then closes the stream
	// attrs.ContentType ends up being set to application/octet-stream
	// We must handle this case so the upload can resume.
	if attrs.ContentType != uploadSessionContentType &&
		attrs.ContentType != blobContentType {
		return storagedriver.PathNotFoundError{Path: w.object.ObjectName()}
	}

	offset := int64(0)
	// NOTE(milosgajdos): if a client creates an empty blob, then
	// closes the stream and then attempts to append to it, the offset
	// will be empty, in which case strconv.ParseInt will return error
	// See: https://pkg.go.dev/strconv#ParseInt
	if attrs.Metadata["Offset"] != "" {
		offset, err = strconv.ParseInt(attrs.Metadata["Offset"], 10, 64)
		if err != nil {
			return err
		}
	}

	r, err := w.object.NewReader(ctx)
	if err != nil {
		return err
	}
	defer r.Close()

	for err == nil && w.buffSize < len(w.buffer) {
		var n int
		n, err = r.Read(w.buffer[w.buffSize:])
		w.buffSize += n
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	// NOTE(milosgajdos): if a client closes an existing session and then attempts
	// to append to an existing blob, the session will be empty; recreate it
	if w.sessionURI = attrs.Metadata["Session-URI"]; w.sessionURI == "" {
		w.sessionURI, err = w.newSession()
		if err != nil {
			return err
		}
	}
	w.offset = offset
	w.size = offset + int64(w.buffSize)
	return nil
}

type request func() error

func retry(req request) error {
	backoff := time.Second
	var err error
	for i := 0; i < maxTries; i++ {
		err = req()
		if err == nil {
			return nil
		}

		status, ok := err.(*googleapi.Error) //nolint:errorlint
		if !ok || (status.Code != http.StatusTooManyRequests && status.Code < http.StatusInternalServerError) {
			return err
		}

		time.Sleep(backoff - time.Second + (time.Duration(rand.Int31n(1000)) * time.Millisecond)) //nolint:gosec
		if i <= 4 {
			backoff *= 2
		}
	}
	return err
}

// Stat retrieves the FileInfo for the given path, including the current
// size in bytes and the creation time.
func (d *driver) Stat(ctx context.Context, path string) (storagedriver.FileInfo, error) {
	var fi storagedriver.FileInfoFields
	// try to get as file
	obj, err := d.bucket.Object(d.pathToKey(path)).Attrs(ctx)
	if err == nil {
		if obj.ContentType == uploadSessionContentType {
			return nil, storagedriver.PathNotFoundError{Path: path}
		}
		fi = storagedriver.FileInfoFields{
			Path:    path,
			Size:    obj.Size,
			ModTime: obj.Updated,
			IsDir:   false,
		}
		return storagedriver.FileInfoInternal{FileInfoFields: fi}, nil
	}
	// try to get as folder
	dirpath := d.pathToDirKey(path)

	query := &storage.Query{
		Prefix: dirpath,
	}

	obj, err = d.bucket.Objects(ctx, query).Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, storagedriver.PathNotFoundError{Path: path}
		}
		return nil, err
	}

	fi = storagedriver.FileInfoFields{
		Path:  path,
		IsDir: true,
	}

	if obj.Name == dirpath {
		fi.Size = obj.Size
		fi.ModTime = obj.Updated
	}
	return storagedriver.FileInfoInternal{FileInfoFields: fi}, nil
}

// List returns a list of the objects that are direct descendants of the
// given path.
func (d *driver) List(ctx context.Context, path string) ([]string, error) {
	query := &storage.Query{
		Delimiter: "/",
		Prefix:    d.pathToDirKey(path),
	}
	objects := d.bucket.Objects(ctx, query)

	list := make([]string, 0, 64)
	for {
		object, err := objects.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}
		// GCS does not guarantee strong consistency between
		// DELETE and LIST operations. Check that the object is not deleted,
		// and filter out any objects with a non-zero time-deleted
		if object.Deleted.IsZero() && object.ContentType != uploadSessionContentType && object.Name != "" {
			list = append(list, d.keyToPath(object.Name))
		}

		if object.Name == "" && object.Prefix != "" {
			subpath := d.keyToPath(object.Prefix)
			list = append(list, subpath)
		}
	}

	if path != "/" && len(list) == 0 {
		// Treat empty response as missing directory, since we don't actually
		// have directories in Google Cloud Storage.
		return nil, storagedriver.PathNotFoundError{Path: path}
	}
	return list, nil
}

// Move moves an object stored at sourcePath to destPath, removing the
// original object.
func (d *driver) Move(ctx context.Context, sourcePath string, destPath string) error {
	srcKey, dstKey := d.pathToKey(sourcePath), d.pathToKey(destPath)
	src := d.bucket.Object(srcKey)
	_, err := d.bucket.Object(dstKey).CopierFrom(src).Run(ctx)
	if err != nil {
		var status *googleapi.Error
		if errors.As(err, &status) {
			if status.Code == http.StatusNotFound {
				return storagedriver.PathNotFoundError{Path: srcKey}
			}
		}
		return fmt.Errorf("move %q to %q: %w", srcKey, dstKey, err)
	}
	err = src.Delete(ctx)
	// if deleting the file fails, log the error, but do not fail; the file was successfully copied,
	// and the original should eventually be cleaned when purging the uploads folder.
	if err != nil {
		log.Info().Ctx(ctx).Msgf("error deleting %v: %v", sourcePath, err)
	}
	return nil
}

// listAll recursively lists all names of objects stored at "prefix" and its subpaths.
func (d *driver) listAll(ctx context.Context, prefix string) ([]string, error) {
	objects := d.bucket.Objects(ctx, &storage.Query{
		Prefix:   prefix,
		Versions: false,
	})

	list := make([]string, 0, 64)
	for {
		object, err := objects.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}
		// GCS does not guarantee strong consistency between
		// DELETE and LIST operations. Check that the object is not deleted,
		// and filter out any objects with a non-zero time-deleted
		if object.Deleted.IsZero() {
			list = append(list, object.Name)
		}
	}

	return list, nil
}

// Delete recursively deletes all objects stored at "path" and its subpaths.
func (d *driver) Delete(ctx context.Context, path string) error {
	prefix := d.pathToDirKey(path)
	keys, err := d.listAll(ctx, prefix)
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		// NOTE(milosgajdos): d.listAll calls (BucketHandle).Objects
		// See: https://pkg.go.dev/cloud.google.com/go/storage#BucketHandle.Objects
		// docs: Objects will be iterated over lexicographically by name.
		// This means we don't have to reverse order the slice; we can
		// range over the keys slice in reverse order
		for i := len(keys) - 1; i >= 0; i-- {
			key := keys[i]
			err := d.bucket.Object(key).Delete(ctx)
			// GCS only guarantees eventual consistency, so listAll might return
			// paths that no longer exist. If this happens, just ignore any not
			// found error
			if status, ok := err.(*googleapi.Error); ok { //nolint:errorlint
				if status.Code == http.StatusNotFound {
					err = nil
				}
			}
			if err != nil {
				return err
			}
		}
		return nil
	}
	err = d.bucket.Object(d.pathToKey(path)).Delete(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return storagedriver.PathNotFoundError{Path: path}
	}
	return err
}

// RedirectURL returns a URL which may be used to retrieve the content stored at
// the given path, possibly using the given options.
func (d *driver) RedirectURL(_ context.Context, method string, path string, filename string) (string, error) {
	if method != http.MethodGet && method != http.MethodHead {
		return "", nil
	}

	opts := &storage.SignedURLOptions{
		GoogleAccessID: d.email,
		PrivateKey:     d.privateKey,
		Method:         method,
		Expires:        time.Now().Add(20 * time.Minute),
	}

	if filename != "" {
		opts.QueryParameters = url.Values{
			"response-content-disposition": {fmt.Sprintf("attachment; filename=\"%s\"", filename)},
		}
	}

	return d.bucket.SignedURL(d.pathToKey(path), opts)
}

// Walk traverses a filesystem defined within driver, starting
// from the given path, calling f on each file.
func (d *driver) Walk(
	ctx context.Context,
	path string,
	f storagedriver.WalkFn,
	options ...func(*storagedriver.WalkOptions),
) error {
	return storagedriver.WalkFallback(ctx, d, path, f, options...)
}

func (w *writer) newSession() (uri string, err error) {
	u := &url.URL{
		Scheme:   "https",
		Host:     "www.googleapis.com",
		Path:     fmt.Sprintf("/upload/storage/v1/b/%v/o", w.object.BucketName()),
		RawQuery: fmt.Sprintf("uploadType=resumable&name=%v", w.object.ObjectName()),
	}
	req, err := http.NewRequestWithContext(w.ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Upload-Content-Type", blobContentType)
	req.Header.Set("Content-Length", "0")

	err = retry(func() error {
		resp, err := w.driver.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		err = googleapi.CheckMediaResponse(resp)
		if err != nil {
			return err
		}
		uri = resp.Header.Get("Location")
		return nil
	})
	return uri, err
}

func (w *writer) putChunk(ctx context.Context, sessionURI string, chunk []byte, from int64, totalSize int64) (
	int64,
	error,
) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, sessionURI, bytes.NewReader(chunk))
	if err != nil {
		return 0, err
	}
	length := int64(len(chunk))
	to := from + length - 1
	size := "*"
	if totalSize >= 0 {
		size = strconv.FormatInt(totalSize, 10)
	}
	req.Header.Set("Content-Type", blobContentType)
	if from == to+1 {
		req.Header.Set("Content-Range", fmt.Sprintf("bytes */%s", size))
	} else {
		req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%s", from, to, size))
	}
	req.Header.Set("Content-Length", strconv.FormatInt(length, 10))

	bytesPut := int64(0)
	err = retry(func() error {
		resp, err := w.driver.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if totalSize < 0 && resp.StatusCode == http.StatusPermanentRedirect {
			groups := rangeHeader.FindStringSubmatch(resp.Header.Get("Range"))
			end, err := strconv.ParseInt(groups[2], 10, 64)
			if err != nil {
				return err
			}
			bytesPut = end - from + 1
			return nil
		}
		err = googleapi.CheckMediaResponse(resp)
		if err != nil {
			return err
		}
		bytesPut = to - from + 1
		return nil
	})
	return bytesPut, err
}

func (d *driver) pathToKey(path string) string {
	return strings.TrimSpace(strings.TrimRight(d.rootDirectory+strings.TrimLeft(path, "/"), "/"))
}

func (d *driver) pathToDirKey(path string) string {
	return d.pathToKey(path) + "/"
}

func (d *driver) keyToPath(key string) string {
	return "/" + strings.Trim(strings.TrimPrefix(key, d.rootDirectory), "/")
}

func (d *driver) CopyObject(_ context.Context, _, _, _ string) error {
	return fmt.Errorf("not yet implemented")
}
