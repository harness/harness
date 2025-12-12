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

package testsuites

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path"
	"sort"
	"sync"
	"testing"
	"time"

	storagedriver "github.com/harness/gitness/registry/app/driver"

	"github.com/stretchr/testify/suite"
)

// randomBytes pre-allocates all of the memory sizes needed for the test. If
// anything panics while accessing randomBytes, just make this number bigger.
var randomBytes = make([]byte, 128<<20)

func init() {
	_, _ = crand.Read(randomBytes) // always returns len(randomBytes) and nil error
}

// DriverConstructor is a function which returns a new
// storagedriver.StorageDriver.
type DriverConstructor func() (storagedriver.StorageDriver, error)

// DriverTeardown is a function which cleans up a suite's
// storagedriver.StorageDriver.
type DriverTeardown func() error

// DriverSuite is a [suite.Suite] test suite designed to test a
// storagedriver.StorageDriver.
type DriverSuite struct {
	suite.Suite
	Constructor DriverConstructor
	Teardown    DriverTeardown
	storagedriver.StorageDriver
	ctx        context.Context
	skipVerify bool
}

// Driver runs [DriverSuite] for the given [DriverConstructor].
func Driver(t *testing.T, driverConstructor DriverConstructor, skipVerify bool) {
	suite.Run(t, &DriverSuite{
		Constructor: driverConstructor,
		ctx:         context.Background(),
		skipVerify:  skipVerify,
	})
}

// SetupSuite implements [suite.SetupAllSuite] interface.
func (suite *DriverSuite) SetupSuite() {
	d, err := suite.Constructor()
	suite.Require().NoError(err)
	suite.StorageDriver = d
}

// TearDownSuite implements [suite.TearDownAllSuite].
func (suite *DriverSuite) TearDownSuite() {
	if suite.Teardown != nil {
		suite.Require().NoError(suite.Teardown())
	}
}

// TearDownTest implements [suite.TearDownTestSuite].
// This causes the suite to abort if any files are left around in the storage
// driver.
func (suite *DriverSuite) TearDownTest() {
	files, _ := suite.StorageDriver.List(suite.ctx, "/")
	if len(files) > 0 {
		suite.T().Fatalf("Storage driver did not clean up properly. Offending files: %#v", files)
	}
}

// TestRootExists ensures that all storage drivers have a root path by default.
func (suite *DriverSuite) TestRootExists() {
	_, err := suite.StorageDriver.List(suite.ctx, "/")
	if err != nil {
		suite.T().Fatalf(`the root path "/" should always exist: %v`, err)
	}
}

// TestValidPaths checks that various valid file paths are accepted by the
// storage driver.
func (suite *DriverSuite) TestValidPaths() {
	contents := randomContents(64)
	validFiles := []string{
		"/a",
		"/2",
		"/aa",
		"/a.a",
		"/0-9/abcdefg",
		"/abcdefg/z.75",
		"/abc/1.2.3.4.5-6_zyx/123.z/4",
		"/docker/docker-registry",
		"/123.abc",
		"/abc./abc",
		"/.abc",
		"/a--b",
		"/a-.b",
		"/_.abc",
		"/Docker/docker-registry",
		"/Abc/Cba",
	}

	for _, filename := range validFiles {
		err := suite.StorageDriver.PutContent(suite.ctx, filename, contents)
		defer suite.deletePath(firstPart(filename))
		suite.Require().NoError(err)

		received, err := suite.StorageDriver.GetContent(suite.ctx, filename)
		suite.Require().NoError(err)
		suite.Require().Equal(contents, received)
	}
}

func (suite *DriverSuite) deletePath(path string) {
	for tries := 2; tries > 0; tries-- {
		err := suite.StorageDriver.Delete(suite.ctx, path)
		if _, ok := err.(storagedriver.PathNotFoundError); ok { //nolint:errorlint
			err = nil
		}
		suite.Require().NoError(err)
		paths, _ := suite.StorageDriver.List(suite.ctx, path)
		if len(paths) == 0 {
			break
		}
		time.Sleep(time.Second * 2)
	}
}

// TestInvalidPaths checks that various invalid file paths are rejected by the
// storage driver.
func (suite *DriverSuite) TestInvalidPaths() {
	contents := randomContents(64)
	invalidFiles := []string{
		"",
		"/",
		"abc",
		"123.abc",
		"//bcd",
		"/abc_123/",
	}

	for _, filename := range invalidFiles {
		err := suite.StorageDriver.PutContent(suite.ctx, filename, contents)
		// only delete if file was successfully written
		if err == nil {
			defer suite.deletePath(firstPart(filename))
		}
		suite.Require().Error(err)
		suite.Require().IsType(err, storagedriver.InvalidPathError{})
		suite.Require().Contains(err.Error(), suite.Name())

		_, err = suite.StorageDriver.GetContent(suite.ctx, filename)
		suite.Require().Error(err)
		suite.Require().IsType(err, storagedriver.InvalidPathError{})
		suite.Require().Contains(err.Error(), suite.Name())
	}
}

// TestWriteRead1 tests a simple write-read workflow.
func (suite *DriverSuite) TestWriteRead1() {
	filename := randomPath(32)
	contents := []byte("a")
	suite.writeReadCompare(filename, contents)
}

// TestWriteRead2 tests a simple write-read workflow with unicode data.
func (suite *DriverSuite) TestWriteRead2() {
	filename := randomPath(32)
	contents := []byte("\xc3\x9f")
	suite.writeReadCompare(filename, contents)
}

// TestWriteRead3 tests a simple write-read workflow with a small string.
func (suite *DriverSuite) TestWriteRead3() {
	filename := randomPath(32)
	contents := randomContents(32)
	suite.writeReadCompare(filename, contents)
}

// TestWriteRead4 tests a simple write-read workflow with 1MB of data.
func (suite *DriverSuite) TestWriteRead4() {
	filename := randomPath(32)
	contents := randomContents(1024 * 1024)
	suite.writeReadCompare(filename, contents)
}

// TestWriteReadNonUTF8 tests that non-utf8 data may be written to the storage
// driver safely.
func (suite *DriverSuite) TestWriteReadNonUTF8() {
	filename := randomPath(32)
	contents := []byte{0x80, 0x80, 0x80, 0x80}
	suite.writeReadCompare(filename, contents)
}

// TestTruncate tests that putting smaller contents than an original file does
// remove the excess contents.
func (suite *DriverSuite) TestTruncate() {
	filename := randomPath(32)
	contents := randomContents(1024 * 1024)
	suite.writeReadCompare(filename, contents)

	contents = randomContents(1024)
	suite.writeReadCompare(filename, contents)
}

// TestReadNonexistent tests reading content from an empty path.
func (suite *DriverSuite) TestReadNonexistent() {
	filename := randomPath(32)
	_, err := suite.StorageDriver.GetContent(suite.ctx, filename)
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())
}

// TestWriteReadStreams1 tests a simple write-read streaming workflow.
func (suite *DriverSuite) TestWriteReadStreams1() {
	filename := randomPath(32)
	contents := []byte("a")
	suite.writeReadCompareStreams(filename, contents)
}

// TestWriteReadStreams2 tests a simple write-read streaming workflow with
// unicode data.
func (suite *DriverSuite) TestWriteReadStreams2() {
	filename := randomPath(32)
	contents := []byte("\xc3\x9f")
	suite.writeReadCompareStreams(filename, contents)
}

// TestWriteReadStreams3 tests a simple write-read streaming workflow with a
// small amount of data.
func (suite *DriverSuite) TestWriteReadStreams3() {
	filename := randomPath(32)
	contents := randomContents(32)
	suite.writeReadCompareStreams(filename, contents)
}

// TestWriteReadStreams4 tests a simple write-read streaming workflow with 1MB
// of data.
func (suite *DriverSuite) TestWriteReadStreams4() {
	filename := randomPath(32)
	contents := randomContents(1024 * 1024)
	suite.writeReadCompareStreams(filename, contents)
}

// TestWriteReadStreamsNonUTF8 tests that non-utf8 data may be written to the
// storage driver safely.
func (suite *DriverSuite) TestWriteReadStreamsNonUTF8() {
	filename := randomPath(32)
	contents := []byte{0x80, 0x80, 0x80, 0x80}
	suite.writeReadCompareStreams(filename, contents)
}

// TestWriteReadLargeStreams tests that a 5GB file may be written to the storage
// driver safely.
func (suite *DriverSuite) TestWriteReadLargeStreams() {
	if testing.Short() {
		suite.T().Skip("Skipping test in short mode")
	}

	filename := randomPath(32)
	defer suite.deletePath(firstPart(filename))

	checksum := sha256.New()
	var fileSize int64 = 5 * 1024 * 1024 * 1024

	contents := newRandReader(fileSize)

	writer, err := suite.StorageDriver.Writer(suite.ctx, filename, false)
	suite.Require().NoError(err)
	written, err := io.Copy(writer, io.TeeReader(contents, checksum))
	suite.Require().NoError(err)
	suite.Require().Equal(fileSize, written)

	err = writer.Commit(context.Background())
	suite.Require().NoError(err)
	err = writer.Close()
	suite.Require().NoError(err)

	reader, err := suite.StorageDriver.Reader(suite.ctx, filename, 0)
	suite.Require().NoError(err)
	defer reader.Close()

	writtenChecksum := sha256.New()
	if _, err := io.Copy(writtenChecksum, reader); err != nil {
		suite.Require().NoError(err)
	}

	suite.Require().Equal(checksum.Sum(nil), writtenChecksum.Sum(nil))
}

// TestReaderWithOffset tests that the appropriate data is streamed when
// reading with a given offset.
func (suite *DriverSuite) TestReaderWithOffset() {
	filename := randomPath(32)
	defer suite.deletePath(firstPart(filename))

	chunkSize := int64(32)

	contentsChunk1 := randomContents(chunkSize)
	contentsChunk2 := randomContents(chunkSize)
	contentsChunk3 := randomContents(chunkSize)

	err := suite.StorageDriver.PutContent(suite.ctx, filename,
		append(append(contentsChunk1, contentsChunk2...), contentsChunk3...))
	suite.Require().NoError(err)

	reader, err := suite.StorageDriver.Reader(suite.ctx, filename, 0)
	suite.Require().NoError(err)
	defer reader.Close()

	readContents, err := io.ReadAll(reader)
	suite.Require().NoError(err)

	suite.Require().Equal(append(append(contentsChunk1, contentsChunk2...), contentsChunk3...), readContents)

	reader, err = suite.StorageDriver.Reader(suite.ctx, filename, chunkSize)
	suite.Require().NoError(err)
	defer reader.Close()

	readContents, err = io.ReadAll(reader)
	suite.Require().NoError(err)

	suite.Require().Equal(append(contentsChunk2, contentsChunk3...), readContents)

	reader, err = suite.StorageDriver.Reader(suite.ctx, filename, chunkSize*2)
	suite.Require().NoError(err)
	defer reader.Close()

	readContents, err = io.ReadAll(reader)
	suite.Require().NoError(err)
	suite.Require().Equal(contentsChunk3, readContents)

	// Ensure we get invalid offset for negative offsets.
	reader, err = suite.StorageDriver.Reader(suite.ctx, filename, -1)
	suite.Require().IsType(err, storagedriver.InvalidOffsetError{})
	suite.Require().Equal(int64(-1), err.(storagedriver.InvalidOffsetError).Offset) //nolint:errorlint,errcheck
	suite.Require().Equal(filename, err.(storagedriver.InvalidOffsetError).Path)    //nolint:errorlint,errcheck
	suite.Require().Nil(reader)
	suite.Require().Contains(err.Error(), suite.Name())

	// Read past the end of the content and make sure we get a reader that
	// returns 0 bytes and io.EOF
	reader, err = suite.StorageDriver.Reader(suite.ctx, filename, chunkSize*3)
	suite.Require().NoError(err)
	defer reader.Close()

	buf := make([]byte, chunkSize)
	n, err := reader.Read(buf)
	suite.Require().ErrorIs(err, io.EOF)
	suite.Require().Equal(0, n)

	// Check the N-1 boundary condition, ensuring we get 1 byte then io.EOF.
	reader, err = suite.StorageDriver.Reader(suite.ctx, filename, chunkSize*3-1)
	suite.Require().NoError(err)
	defer reader.Close()

	n, err = reader.Read(buf)
	suite.Require().Equal(1, n)

	// We don't care whether the io.EOF comes on the this read or the first
	// zero read, but the only error acceptable here is io.EOF.
	if err != nil {
		suite.Require().ErrorIs(err, io.EOF)
	}

	// Any more reads should result in zero bytes and io.EOF
	n, err = reader.Read(buf)
	suite.Require().Equal(0, n)
	suite.Require().ErrorIs(err, io.EOF)
}

// TestContinueStreamAppendLarge tests that a stream write can be appended to without
// corrupting the data with a large chunk size.
func (suite *DriverSuite) TestContinueStreamAppendLarge() {
	chunkSize := int64(10 * 1024 * 1024)
	if suite.Name() == "azure" {
		chunkSize = int64(4 * 1024 * 1024)
	}
	suite.testContinueStreamAppend(chunkSize)
}

// TestContinueStreamAppendSmall is the same as TestContinueStreamAppendLarge, but only
// with a tiny chunk size in order to test corner cases for some cloud storage drivers.
func (suite *DriverSuite) TestContinueStreamAppendSmall() {
	suite.testContinueStreamAppend(int64(32))
}

func (suite *DriverSuite) testContinueStreamAppend(chunkSize int64) {
	filename := randomPath(32)
	defer suite.deletePath(firstPart(filename))

	var fullContents bytes.Buffer
	contents := io.TeeReader(newRandReader(chunkSize*3), &fullContents)

	writer, err := suite.StorageDriver.Writer(suite.ctx, filename, false)
	suite.Require().NoError(err)
	nn, err := io.CopyN(writer, contents, chunkSize)
	suite.Require().NoError(err)
	suite.Require().Equal(chunkSize, nn)

	err = writer.Close()
	suite.Require().NoError(err)

	curSize := writer.Size()
	suite.Require().Equal(chunkSize, curSize)

	writer, err = suite.StorageDriver.Writer(suite.ctx, filename, true)
	suite.Require().NoError(err)
	suite.Require().Equal(curSize, writer.Size())

	nn, err = io.CopyN(writer, contents, chunkSize)
	suite.Require().NoError(err)
	suite.Require().Equal(chunkSize, nn)

	err = writer.Close()
	suite.Require().NoError(err)

	curSize = writer.Size()
	suite.Require().Equal(2*chunkSize, curSize)

	writer, err = suite.StorageDriver.Writer(suite.ctx, filename, true)
	suite.Require().NoError(err)
	suite.Require().Equal(curSize, writer.Size())

	nn, err = io.CopyN(writer, contents, chunkSize)
	suite.Require().NoError(err)
	suite.Require().Equal(chunkSize, nn)

	err = writer.Commit(context.Background())
	suite.Require().NoError(err)
	err = writer.Close()
	suite.Require().NoError(err)

	received, err := suite.StorageDriver.GetContent(suite.ctx, filename)
	suite.Require().NoError(err)
	suite.Require().Equal(fullContents.Bytes(), received)
}

// TestReadNonexistentStream tests that reading a stream for a nonexistent path
// fails.
func (suite *DriverSuite) TestReadNonexistentStream() {
	filename := randomPath(32)

	_, err := suite.StorageDriver.Reader(suite.ctx, filename, 0)
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())

	_, err = suite.StorageDriver.Reader(suite.ctx, filename, 64)
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())
}

// TestWriteZeroByteStreamThenAppend tests if zero byte file handling works for append to a Stream.
func (suite *DriverSuite) TestWriteZeroByteStreamThenAppend() {
	filename := randomPath(32)
	defer suite.deletePath(firstPart(filename))
	chunkSize := int64(32)
	contentsChunk1 := randomContents(chunkSize)

	// Open a Writer
	writer, err := suite.StorageDriver.Writer(suite.ctx, filename, false)
	suite.Require().NoError(err)

	// Close the Writer
	err = writer.Commit(context.Background())
	suite.Require().NoError(err)
	err = writer.Close()
	suite.Require().NoError(err)
	curSize := writer.Size()
	suite.Require().Equal(int64(0), curSize)

	// Open a Reader
	reader, err := suite.StorageDriver.Reader(suite.ctx, filename, 0)
	suite.Require().NoError(err)
	defer reader.Close()

	// Check the file is empty
	buf := make([]byte, chunkSize)
	n, err := reader.Read(buf)
	suite.Require().ErrorIs(err, io.EOF)
	suite.Require().Equal(0, n)

	// Open a Writer for Append
	awriter, err := suite.StorageDriver.Writer(suite.ctx, filename, true)
	suite.Require().NoError(err)

	// Write small bytes to AppendWriter
	nn, err := io.Copy(awriter, bytes.NewReader(contentsChunk1))
	suite.Require().NoError(err)
	suite.Require().Equal(int64(len(contentsChunk1)), nn)

	// Close the AppendWriter
	err = awriter.Commit(context.Background())
	suite.Require().NoError(err)
	err = awriter.Close()
	suite.Require().NoError(err)
	appendSize := awriter.Size()
	suite.Require().Equal(int64(len(contentsChunk1)), appendSize)

	// Open a Reader
	reader, err = suite.StorageDriver.Reader(suite.ctx, filename, 0)
	suite.Require().NoError(err)
	defer reader.Close()

	// Read small bytes from Reader
	readContents, err := io.ReadAll(reader)
	suite.Require().NoError(err)
	suite.Require().Equal(contentsChunk1, readContents)
}

// TestWriteZeroByteContentThenAppend tests if zero byte file handling works for append to PutContent.
func (suite *DriverSuite) TestWriteZeroByteContentThenAppend() {
	filename := randomPath(32)
	defer suite.deletePath(firstPart(filename))
	chunkSize := int64(32)
	contentsChunk1 := randomContents(chunkSize)

	err := suite.StorageDriver.PutContent(suite.ctx, filename, nil)
	suite.Require().NoError(err)

	// Open a Reader
	reader, err := suite.StorageDriver.Reader(suite.ctx, filename, 0)
	suite.Require().NoError(err)
	defer reader.Close()

	// Check the file is empty
	buf := make([]byte, chunkSize)
	n, err := reader.Read(buf)
	suite.Require().ErrorIs(err, io.EOF)
	suite.Require().Equal(0, n)

	// Open a Writer for Append
	awriter, err := suite.StorageDriver.Writer(suite.ctx, filename, true)
	suite.Require().NoError(err)

	// Write small bytes to AppendWriter
	nn, err := io.Copy(awriter, bytes.NewReader(contentsChunk1))
	suite.Require().NoError(err)
	suite.Require().Equal(int64(len(contentsChunk1)), nn)

	// Close the AppendWriter
	err = awriter.Commit(context.Background())
	suite.Require().NoError(err)
	err = awriter.Close()
	suite.Require().NoError(err)
	appendSize := awriter.Size()
	suite.Require().Equal(int64(len(contentsChunk1)), appendSize)

	// Open a Reader
	reader, err = suite.StorageDriver.Reader(suite.ctx, filename, 0)
	suite.Require().NoError(err)
	defer reader.Close()

	// Read small bytes from Reader
	readContents, err := io.ReadAll(reader)
	suite.Require().NoError(err)
	suite.Require().Equal(contentsChunk1, readContents)
}

// TestList checks the returned list of keys after populating a directory tree.
func (suite *DriverSuite) TestList() {
	rootDirectory := "/" + randomFilename(int64(8+rand.Intn(8))) //nolint:gosec
	defer suite.deletePath(rootDirectory)

	doesnotexist := path.Join(rootDirectory, "nonexistent")
	_, err := suite.StorageDriver.List(suite.ctx, doesnotexist)
	suite.Require().Equal(err, storagedriver.PathNotFoundError{
		Path:       doesnotexist,
		DriverName: suite.StorageDriver.Name(),
	})

	parentDirectory := rootDirectory + "/" + randomFilename(int64(8+rand.Intn(8))) //nolint:gosec
	childFiles := make([]string, 50)
	for i := 0; i < len(childFiles); i++ {
		childFile := parentDirectory + "/" + randomFilename(int64(8+rand.Intn(8))) //nolint:gosec
		childFiles[i] = childFile
		err := suite.StorageDriver.PutContent(suite.ctx, childFile, randomContents(32))
		suite.Require().NoError(err)
	}
	sort.Strings(childFiles)

	keys, err := suite.StorageDriver.List(suite.ctx, "/")
	suite.Require().NoError(err)
	suite.Require().Equal([]string{rootDirectory}, keys)

	keys, err = suite.StorageDriver.List(suite.ctx, rootDirectory)
	suite.Require().NoError(err)
	suite.Require().Equal([]string{parentDirectory}, keys)

	keys, err = suite.StorageDriver.List(suite.ctx, parentDirectory)
	suite.Require().NoError(err)

	sort.Strings(keys)
	suite.Require().Equal(childFiles, keys)

	// A few checks to add here (check out #819 for more discussion on this):
	// 1. Ensure that all paths are absolute.
	// 2. Ensure that listings only include direct children.
	// 3. Ensure that we only respond to directory listings that end with a slash (maybe?).
}

// TestMove checks that a moved object no longer exists at the source path and
// does exist at the destination.
func (suite *DriverSuite) TestMove() {
	contents := randomContents(32)
	sourcePath := randomPath(32)
	destPath := randomPath(32)

	defer suite.deletePath(firstPart(sourcePath))
	defer suite.deletePath(firstPart(destPath))

	err := suite.StorageDriver.PutContent(suite.ctx, sourcePath, contents)
	suite.Require().NoError(err)

	err = suite.StorageDriver.Move(suite.ctx, sourcePath, destPath)
	suite.Require().NoError(err)

	received, err := suite.StorageDriver.GetContent(suite.ctx, destPath)
	suite.Require().NoError(err)
	suite.Require().Equal(contents, received)

	_, err = suite.StorageDriver.GetContent(suite.ctx, sourcePath)
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())
}

// TestMoveOverwrite checks that a moved object no longer exists at the source
// path and overwrites the contents at the destination.
func (suite *DriverSuite) TestMoveOverwrite() {
	sourcePath := randomPath(32)
	destPath := randomPath(32)
	sourceContents := randomContents(32)
	destContents := randomContents(64)

	defer suite.deletePath(firstPart(sourcePath))
	defer suite.deletePath(firstPart(destPath))

	err := suite.StorageDriver.PutContent(suite.ctx, sourcePath, sourceContents)
	suite.Require().NoError(err)

	err = suite.StorageDriver.PutContent(suite.ctx, destPath, destContents)
	suite.Require().NoError(err)

	err = suite.StorageDriver.Move(suite.ctx, sourcePath, destPath)
	suite.Require().NoError(err)

	received, err := suite.StorageDriver.GetContent(suite.ctx, destPath)
	suite.Require().NoError(err)
	suite.Require().Equal(sourceContents, received)

	_, err = suite.StorageDriver.GetContent(suite.ctx, sourcePath)
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())
}

// TestMoveNonexistent checks that moving a nonexistent key fails and does not
// delete the data at the destination path.
func (suite *DriverSuite) TestMoveNonexistent() {
	contents := randomContents(32)
	sourcePath := randomPath(32)
	destPath := randomPath(32)

	defer suite.deletePath(firstPart(destPath))

	err := suite.StorageDriver.PutContent(suite.ctx, destPath, contents)
	suite.Require().NoError(err)

	err = suite.StorageDriver.Move(suite.ctx, sourcePath, destPath)
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())

	received, err := suite.StorageDriver.GetContent(suite.ctx, destPath)
	suite.Require().NoError(err)
	suite.Require().Equal(contents, received)
}

// TestMoveInvalid provides various checks for invalid moves.
func (suite *DriverSuite) TestMoveInvalid() {
	contents := randomContents(32)

	// Create a regular file.
	err := suite.StorageDriver.PutContent(suite.ctx, "/notadir", contents)
	suite.Require().NoError(err)
	defer suite.deletePath("/notadir")

	// Now try to move a non-existent file under it.
	err = suite.StorageDriver.Move(suite.ctx, "/notadir/foo", "/notadir/bar")
	suite.Require().Error(err) // non-nil error
}

// TestDelete checks that the delete operation removes data from the storage
// driver.
func (suite *DriverSuite) TestDelete() {
	filename := randomPath(32)
	contents := randomContents(32)

	defer suite.deletePath(firstPart(filename))

	err := suite.StorageDriver.PutContent(suite.ctx, filename, contents)
	suite.Require().NoError(err)

	err = suite.StorageDriver.Delete(suite.ctx, filename)
	suite.Require().NoError(err)

	_, err = suite.StorageDriver.GetContent(suite.ctx, filename)
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())
}

// TestRedirectURL checks that the RedirectURL method functions properly,
// but only if it is implemented.
func (suite *DriverSuite) TestRedirectURL() {
	filename := randomPath(32)
	contents := randomContents(32)

	defer suite.deletePath(firstPart(filename))

	err := suite.StorageDriver.PutContent(suite.ctx, filename, contents)
	suite.Require().NoError(err)

	url, err := suite.StorageDriver.RedirectURL(suite.ctx, http.MethodGet, filename, filename)
	if url == "" && err == nil {
		return
	}
	suite.Require().NoError(err)

	client := http.DefaultClient
	if suite.skipVerify {
		httpTransport := http.DefaultTransport.(*http.Transport).Clone()      //nolint:errcheck
		httpTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
		client = &http.Client{
			Transport: httpTransport,
		}
	}

	req, err := http.NewRequestWithContext(suite.ctx, http.MethodGet, url, nil)
	suite.Require().NoError(err)

	response, err := client.Do(req)
	suite.Require().NoError(err)
	defer response.Body.Close()

	read, err := io.ReadAll(response.Body)
	suite.Require().NoError(err)
	suite.Require().Equal(contents, read)

	url, err = suite.StorageDriver.RedirectURL(suite.ctx, http.MethodHead, filename, filename)
	if url == "" && err == nil {
		return
	}
	suite.Require().NoError(err)
	req, _ = http.NewRequestWithContext(suite.ctx, http.MethodHead, url, nil)

	response, err = client.Do(req)
	suite.Require().NoError(err)
	defer response.Body.Close()
	suite.Require().Equal(200, response.StatusCode)
	suite.Require().Equal(int64(32), response.ContentLength)
}

// TestDeleteNonexistent checks that removing a nonexistent key fails.
func (suite *DriverSuite) TestDeleteNonexistent() {
	filename := randomPath(32)
	err := suite.StorageDriver.Delete(suite.ctx, filename)
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())
}

// TestDeleteFolder checks that deleting a folder removes all child elements.
func (suite *DriverSuite) TestDeleteFolder() {
	dirname := randomPath(32)
	filename1 := randomPath(32)
	filename2 := randomPath(32)
	filename3 := randomPath(32)
	contents := randomContents(32)

	defer suite.deletePath(firstPart(dirname))

	err := suite.StorageDriver.PutContent(suite.ctx, path.Join(dirname, filename1), contents)
	suite.Require().NoError(err)

	err = suite.StorageDriver.PutContent(suite.ctx, path.Join(dirname, filename2), contents)
	suite.Require().NoError(err)

	err = suite.StorageDriver.PutContent(suite.ctx, path.Join(dirname, filename3), contents)
	suite.Require().NoError(err)

	err = suite.StorageDriver.Delete(suite.ctx, path.Join(dirname, filename1))
	suite.Require().NoError(err)

	_, err = suite.StorageDriver.GetContent(suite.ctx, path.Join(dirname, filename1))
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())

	_, err = suite.StorageDriver.GetContent(suite.ctx, path.Join(dirname, filename2))
	suite.Require().NoError(err)

	_, err = suite.StorageDriver.GetContent(suite.ctx, path.Join(dirname, filename3))
	suite.Require().NoError(err)

	err = suite.StorageDriver.Delete(suite.ctx, dirname)
	suite.Require().NoError(err)

	_, err = suite.StorageDriver.GetContent(suite.ctx, path.Join(dirname, filename1))
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())

	_, err = suite.StorageDriver.GetContent(suite.ctx, path.Join(dirname, filename2))
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())

	_, err = suite.StorageDriver.GetContent(suite.ctx, path.Join(dirname, filename3))
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())
}

// TestDeleteOnlyDeletesSubpaths checks that deleting path A does not
// delete path B when A is a prefix of B but B is not a subpath of A (so that
// deleting "/a" does not delete "/ab").  This matters for services like S3 that
// do not implement directories.
func (suite *DriverSuite) TestDeleteOnlyDeletesSubpaths() {
	dirname := randomPath(32)
	filename := randomPath(32)
	contents := randomContents(32)

	defer suite.deletePath(firstPart(dirname))

	err := suite.StorageDriver.PutContent(suite.ctx, path.Join(dirname, filename), contents)
	suite.Require().NoError(err)

	err = suite.StorageDriver.PutContent(suite.ctx, path.Join(dirname, filename+"suffix"), contents)
	suite.Require().NoError(err)

	err = suite.StorageDriver.PutContent(suite.ctx, path.Join(dirname, dirname, filename), contents)
	suite.Require().NoError(err)

	err = suite.StorageDriver.PutContent(suite.ctx, path.Join(dirname, dirname+"suffix", filename), contents)
	suite.Require().NoError(err)

	err = suite.StorageDriver.Delete(suite.ctx, path.Join(dirname, filename))
	suite.Require().NoError(err)

	_, err = suite.StorageDriver.GetContent(suite.ctx, path.Join(dirname, filename))
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())

	_, err = suite.StorageDriver.GetContent(suite.ctx, path.Join(dirname, filename+"suffix"))
	suite.Require().NoError(err)

	err = suite.StorageDriver.Delete(suite.ctx, path.Join(dirname, dirname))
	suite.Require().NoError(err)

	_, err = suite.StorageDriver.GetContent(suite.ctx, path.Join(dirname, dirname, filename))
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())

	_, err = suite.StorageDriver.GetContent(suite.ctx, path.Join(dirname, dirname+"suffix", filename))
	suite.Require().NoError(err)
}

// TestStatCall runs verifies the implementation of the storagedriver's Stat call.
func (suite *DriverSuite) TestStatCall() {
	content := randomContents(4096)
	dirPath := randomPath(32)
	fileName := randomFilename(32)
	filePath := path.Join(dirPath, fileName)

	defer suite.deletePath(firstPart(dirPath))

	// Call on non-existent file/dir, check error.
	fi, err := suite.StorageDriver.Stat(suite.ctx, dirPath)
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())
	suite.Require().Nil(fi)

	fi, err = suite.StorageDriver.Stat(suite.ctx, filePath)
	suite.Require().Error(err)
	suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	suite.Require().Contains(err.Error(), suite.Name())
	suite.Require().Nil(fi)

	err = suite.StorageDriver.PutContent(suite.ctx, filePath, content)
	suite.Require().NoError(err)

	// Call on regular file, check results
	fi, err = suite.StorageDriver.Stat(suite.ctx, filePath)
	suite.Require().NoError(err)
	suite.Require().NotNil(fi)
	suite.Require().Equal(filePath, fi.Path())
	suite.Require().Equal(int64(len(content)), fi.Size())
	suite.Require().False(fi.IsDir())
	createdTime := fi.ModTime()

	// Sleep and modify the file
	time.Sleep(time.Second * 10)
	content = randomContents(4096)
	err = suite.StorageDriver.PutContent(suite.ctx, filePath, content)
	suite.Require().NoError(err)
	fi, err = suite.StorageDriver.Stat(suite.ctx, filePath)
	suite.Require().NoError(err)
	suite.Require().NotNil(fi)
	time.Sleep(time.Second * 5) // allow changes to propagate (eventual consistency)

	// Check if the modification time is after the creation time.
	// In case of cloud storage services, storage frontend nodes might have
	// time drift between them, however that should be solved with sleeping
	// before update.
	modTime := fi.ModTime()
	if !modTime.After(createdTime) {
		suite.T().Errorf("modtime (%s) is before the creation time (%s)", modTime, createdTime)
	}

	// Call on directory (do not check ModTime as dirs don't need to support it)
	fi, err = suite.StorageDriver.Stat(suite.ctx, dirPath)
	suite.Require().NoError(err)
	suite.Require().NotNil(fi)
	suite.Require().Equal(dirPath, fi.Path())
	suite.Require().Equal(int64(0), fi.Size())
	suite.Require().True(fi.IsDir())

	// The storage healthcheck performs this exact call to Stat.
	// PathNotFoundErrors are not considered health check failures.
	_, err = suite.StorageDriver.Stat(suite.ctx, "/")
	// Some drivers will return a not found here, while others will not
	// return an error at all. If we get an error, ensure it's a not found.
	if err != nil {
		suite.Require().IsType(err, storagedriver.PathNotFoundError{})
	}
}

// TestPutContentMultipleTimes checks that if storage driver can overwrite the content
// in the subsequent puts. Validates that PutContent does not have to work
// with an offset like Writer does and overwrites the file entirely
// rather than writing the data to the [0,len(data)) of the file.
func (suite *DriverSuite) TestPutContentMultipleTimes() {
	filename := randomPath(32)
	contents := randomContents(4096)

	defer suite.deletePath(firstPart(filename))
	err := suite.StorageDriver.PutContent(suite.ctx, filename, contents)
	suite.Require().NoError(err)

	contents = randomContents(2048) // upload a different, smaller file
	err = suite.StorageDriver.PutContent(suite.ctx, filename, contents)
	suite.Require().NoError(err)

	readContents, err := suite.StorageDriver.GetContent(suite.ctx, filename)
	suite.Require().NoError(err)
	suite.Require().Equal(contents, readContents)
}

// TestConcurrentStreamReads checks that multiple clients can safely read from
// the same file simultaneously with various offsets.
func (suite *DriverSuite) TestConcurrentStreamReads() {
	var filesize int64 = 128 * 1024 * 1024

	if testing.Short() {
		filesize = 10 * 1024 * 1024
		suite.T().Log("Reducing file size to 10MB for short mode")
	}

	filename := randomPath(32)
	contents := randomContents(filesize)

	defer suite.deletePath(firstPart(filename))

	err := suite.StorageDriver.PutContent(suite.ctx, filename, contents)
	suite.Require().NoError(err)

	var wg sync.WaitGroup

	readContents := func() {
		defer wg.Done()
		offset := rand.Int63n(int64(len(contents))) //nolint:gosec
		reader, err := suite.StorageDriver.Reader(suite.ctx, filename, offset)
		suite.Require().NoError(err)

		readContents, err := io.ReadAll(reader)
		suite.Require().NoError(err)
		suite.Require().Equal(contents[offset:], readContents)
	}

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go readContents()
	}
	wg.Wait()
}

// TestConcurrentFileStreams checks that multiple *os.File objects can be passed
// in to Writer concurrently without hanging.
func (suite *DriverSuite) TestConcurrentFileStreams() {
	numStreams := 32

	if testing.Short() {
		numStreams = 8
		suite.T().Log("Reducing number of streams to 8 for short mode")
	}

	var wg sync.WaitGroup

	testStream := func(size int64) {
		defer wg.Done()
		suite.testFileStreams(size)
	}

	wg.Add(numStreams)
	for i := numStreams; i > 0; i-- {
		go testStream(int64(numStreams) * 1024 * 1024)
	}

	wg.Wait()
}

type DriverBenchmarkSuite struct {
	DriverSuite
}

func BenchDriver(b *testing.B, driverConstructor DriverConstructor) {
	benchsuite := &DriverBenchmarkSuite{
		DriverSuite{
			Constructor: driverConstructor,
			ctx:         context.Background(),
		},
	}
	benchsuite.SetupSuite()
	b.Cleanup(benchsuite.TearDownSuite)

	b.Run("PutGetEmptyFiles", benchsuite.BenchmarkPutGetEmptyFiles)
	b.Run("PutGet1KBFiles", benchsuite.BenchmarkPutGet1KBFiles)
	b.Run("PutGet1MBFiles", benchsuite.BenchmarkPutGet1MBFiles)
	b.Run("PutGet1GBFiles", benchsuite.BenchmarkPutGet1GBFiles)
	b.Run("StreamEmptyFiles", benchsuite.BenchmarkStreamEmptyFiles)
	b.Run("Stream1KBFiles", benchsuite.BenchmarkStream1KBFiles)
	b.Run("Stream1MBFiles", benchsuite.BenchmarkStream1MBFiles)
	b.Run("Stream1GBFiles", benchsuite.BenchmarkStream1GBFiles)
	b.Run("List5Files", benchsuite.BenchmarkList5Files)
	b.Run("List50Files", benchsuite.BenchmarkList50Files)
	b.Run("Delete5Files", benchsuite.BenchmarkDelete5Files)
	b.Run("Delete50Files", benchsuite.BenchmarkDelete50Files)
}

// BenchmarkPutGetEmptyFiles benchmarks PutContent/GetContent for 0B files.
func (s *DriverBenchmarkSuite) BenchmarkPutGetEmptyFiles(b *testing.B) {
	s.benchmarkPutGetFiles(b, 0)
}

// BenchmarkPutGet1KBFiles benchmarks PutContent/GetContent for 1KB files.
func (s *DriverBenchmarkSuite) BenchmarkPutGet1KBFiles(b *testing.B) {
	s.benchmarkPutGetFiles(b, 1024)
}

// BenchmarkPutGet1MBFiles benchmarks PutContent/GetContent for 1MB files.
func (s *DriverBenchmarkSuite) BenchmarkPutGet1MBFiles(b *testing.B) {
	s.benchmarkPutGetFiles(b, 1024*1024)
}

// BenchmarkPutGet1GBFiles benchmarks PutContent/GetContent for 1GB files.
func (s *DriverBenchmarkSuite) BenchmarkPutGet1GBFiles(b *testing.B) {
	s.benchmarkPutGetFiles(b, 1024*1024*1024)
}

func (s *DriverBenchmarkSuite) benchmarkPutGetFiles(b *testing.B, size int64) {
	b.SetBytes(size)
	parentDir := randomPath(8)
	defer func() {
		b.StopTimer()
		// nolint:errcheck
		s.StorageDriver.Delete(s.ctx, firstPart(parentDir))
	}()

	for b.Loop() {
		filename := path.Join(parentDir, randomPath(32))
		err := s.StorageDriver.PutContent(s.ctx, filename, randomContents(size))
		s.Suite.Require().NoError(err)

		_, err = s.StorageDriver.GetContent(s.ctx, filename)
		s.Suite.Require().NoError(err)
	}
}

// BenchmarkStreamEmptyFiles benchmarks Writer/Reader for 0B files.
func (s *DriverBenchmarkSuite) BenchmarkStreamEmptyFiles(b *testing.B) {
	s.benchmarkStreamFiles(b, 0)
}

// BenchmarkStream1KBFiles benchmarks Writer/Reader for 1KB files.
func (s *DriverBenchmarkSuite) BenchmarkStream1KBFiles(b *testing.B) {
	s.benchmarkStreamFiles(b, 1024)
}

// BenchmarkStream1MBFiles benchmarks Writer/Reader for 1MB files.
func (s *DriverBenchmarkSuite) BenchmarkStream1MBFiles(b *testing.B) {
	s.benchmarkStreamFiles(b, 1024*1024)
}

// BenchmarkStream1GBFiles benchmarks Writer/Reader for 1GB files.
func (s *DriverBenchmarkSuite) BenchmarkStream1GBFiles(b *testing.B) {
	s.benchmarkStreamFiles(b, 1024*1024*1024)
}

func (s *DriverBenchmarkSuite) benchmarkStreamFiles(b *testing.B, size int64) {
	b.SetBytes(size)
	parentDir := randomPath(8)
	defer func() {
		b.StopTimer()
		// nolint:errcheck
		s.StorageDriver.Delete(s.ctx, firstPart(parentDir))
	}()

	for b.Loop() {
		filename := path.Join(parentDir, randomPath(32))
		writer, err := s.StorageDriver.Writer(s.ctx, filename, false)
		s.Suite.Require().NoError(err)
		written, err := io.Copy(writer, bytes.NewReader(randomContents(size)))
		s.Suite.Require().NoError(err)
		s.Suite.Require().Equal(size, written)

		err = writer.Commit(context.Background())
		s.Suite.Require().NoError(err)
		err = writer.Close()
		s.Suite.Require().NoError(err)

		rc, err := s.StorageDriver.Reader(s.ctx, filename, 0)
		s.Suite.Require().NoError(err)
		rc.Close()
	}
}

// BenchmarkList5Files benchmarks List for 5 small files.
func (s *DriverBenchmarkSuite) BenchmarkList5Files(b *testing.B) {
	s.benchmarkListFiles(b, 5)
}

// BenchmarkList50Files benchmarks List for 50 small files.
func (s *DriverBenchmarkSuite) BenchmarkList50Files(b *testing.B) {
	s.benchmarkListFiles(b, 50)
}

func (s *DriverBenchmarkSuite) benchmarkListFiles(b *testing.B, numFiles int64) {
	parentDir := randomPath(8)
	defer func() {
		b.StopTimer()
		// nolint:errcheck
		s.StorageDriver.Delete(s.ctx, firstPart(parentDir))
	}()

	for range numFiles {
		err := s.StorageDriver.PutContent(s.ctx, path.Join(parentDir, randomPath(32)), nil)
		s.Suite.Require().NoError(err)
	}

	b.ResetTimer()
	for b.Loop() {
		files, err := s.StorageDriver.List(s.ctx, parentDir)
		s.Suite.Require().NoError(err)
		s.Suite.Require().Equal(numFiles, int64(len(files)))
	}
}

// BenchmarkDelete5Files benchmarks Delete for 5 small files.
func (s *DriverBenchmarkSuite) BenchmarkDelete5Files(b *testing.B) {
	s.benchmarkDeleteFiles(b, 5)
}

// BenchmarkDelete50Files benchmarks Delete for 50 small files.
func (s *DriverBenchmarkSuite) BenchmarkDelete50Files(b *testing.B) {
	s.benchmarkDeleteFiles(b, 50)
}

func (s *DriverBenchmarkSuite) benchmarkDeleteFiles(b *testing.B, numFiles int64) {
	for i := 0; i < b.N; i++ {
		parentDir := randomPath(8)
		defer s.deletePath(firstPart(parentDir))

		b.StopTimer()
		for j := int64(0); j < numFiles; j++ {
			err := s.StorageDriver.PutContent(s.ctx, path.Join(parentDir, randomPath(32)), nil)
			s.Suite.Require().NoError(err)
		}
		b.StartTimer()

		// This is the operation we're benchmarking
		err := s.StorageDriver.Delete(s.ctx, firstPart(parentDir))
		s.Suite.Require().NoError(err)
	}
}

func (suite *DriverSuite) testFileStreams(size int64) {
	tf, err := os.CreateTemp("", "tf")
	suite.Require().NoError(err)
	defer os.Remove(tf.Name())
	defer tf.Close()

	filename := randomPath(32)
	defer suite.deletePath(firstPart(filename))

	contents := randomContents(size)

	_, err = tf.Write(contents)
	suite.Require().NoError(err)

	err = tf.Sync()
	suite.Require().NoError(err)
	_, err = tf.Seek(0, io.SeekStart)
	suite.Require().NoError(err)

	writer, err := suite.StorageDriver.Writer(suite.ctx, filename, false)
	suite.Require().NoError(err)
	nn, err := io.Copy(writer, tf)
	suite.Require().NoError(err)
	suite.Require().Equal(size, nn)

	err = writer.Commit(context.Background())
	suite.Require().NoError(err)
	err = writer.Close()
	suite.Require().NoError(err)

	reader, err := suite.StorageDriver.Reader(suite.ctx, filename, 0)
	suite.Require().NoError(err)
	defer reader.Close()

	readContents, err := io.ReadAll(reader)
	suite.Require().NoError(err)

	suite.Require().Equal(contents, readContents)
}

func (suite *DriverSuite) writeReadCompare(filename string, contents []byte) {
	defer suite.deletePath(firstPart(filename))

	err := suite.StorageDriver.PutContent(suite.ctx, filename, contents)
	suite.Require().NoError(err)

	readContents, err := suite.StorageDriver.GetContent(suite.ctx, filename)
	suite.Require().NoError(err)

	suite.Require().Equal(contents, readContents)
}

func (suite *DriverSuite) writeReadCompareStreams(filename string, contents []byte) {
	defer suite.deletePath(firstPart(filename))

	writer, err := suite.StorageDriver.Writer(suite.ctx, filename, false)
	suite.Require().NoError(err)
	nn, err := io.Copy(writer, bytes.NewReader(contents))
	suite.Require().NoError(err)
	suite.Require().Equal(int64(len(contents)), nn)

	err = writer.Commit(context.Background())
	suite.Require().NoError(err)
	err = writer.Close()
	suite.Require().NoError(err)

	reader, err := suite.StorageDriver.Reader(suite.ctx, filename, 0)
	suite.Require().NoError(err)
	defer reader.Close()

	readContents, err := io.ReadAll(reader)
	suite.Require().NoError(err)

	suite.Require().Equal(contents, readContents)
}

var (
	filenameChars  = []byte("abcdefghijklmnopqrstuvwxyz0123456789")
	separatorChars = []byte("-")
)

func randomPath(length int64) string {
	path := "/"
	for int64(len(path)) < length {
		chunkLength := rand.Int63n(length-int64(len(path))) + 1 //nolint:gosec
		chunk := randomFilename(chunkLength)
		path += chunk
		remaining := length - int64(len(path))
		if remaining == 1 {
			path += randomFilename(1)
		} else if remaining > 1 {
			path += "/"
		}
	}
	return path
}

func randomFilename(length int64) string {
	b := make([]byte, length)
	wasSeparator := true
	for i := range b {
		if !wasSeparator && i < len(b)-1 && rand.Intn(4) == 0 { //nolint:gosec
			b[i] = separatorChars[rand.Intn(len(separatorChars))] //nolint:gosec
			wasSeparator = true
		} else {
			b[i] = filenameChars[rand.Intn(len(filenameChars))] //nolint:gosec
			wasSeparator = false
		}
	}
	return string(b)
}

func randomContents(length int64) []byte {
	return randomBytes[:length]
}

type randReader struct {
	r int64
	m sync.Mutex
}

func (rr *randReader) Read(p []byte) (n int, err error) {
	rr.m.Lock()
	defer rr.m.Unlock()

	toread := int64(len(p))
	if toread > rr.r {
		toread = rr.r
	}
	n = copy(p, randomContents(toread))
	rr.r -= int64(n)

	if rr.r <= 0 {
		err = io.EOF
	}

	return
}

func newRandReader(n int64) *randReader {
	return &randReader{r: n}
}

func firstPart(filePath string) string {
	if filePath == "" {
		return "/"
	}
	for {
		if filePath[len(filePath)-1] == '/' {
			filePath = filePath[:len(filePath)-1]
		}

		dir, file := path.Split(filePath)
		if dir == "" && file == "" {
			return "/"
		}
		if dir == "/" || dir == "" {
			return "/" + file
		}
		if file == "" {
			return dir
		}
		filePath = dir
	}
}
