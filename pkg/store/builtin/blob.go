package builtin

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/russross/meddler"
)

type blob struct {
	ID   int64  `meddler:"blob_id,pk"`
	Path string `meddler:"blob_path"`
	Data string `meddler:"blob_data,gobgzip"`
}

type Blobstore struct {
	meddler.DB
}

// Del removes an object from the blobstore.
func (db *Blobstore) DelBlob(path string) error {
	var _, err = db.Exec(rebind(blobDeleteStmt), path)
	return err
}

// Get retrieves an object from the blobstore.
func (db *Blobstore) GetBlob(path string) ([]byte, error) {
	var blob = blob{}
	var err = meddler.QueryRow(db, &blob, rebind(blobQuery), path)
	return []byte(blob.Data), err
}

// GetBlobReader retrieves an object from the blobstore.
// It is the caller's responsibility to call Close on
// the ReadCloser when finished reading.
func (db *Blobstore) GetBlobReader(path string) (io.ReadCloser, error) {
	var blob, err = db.GetBlob(path)
	var buf = bytes.NewBuffer(blob)
	return ioutil.NopCloser(buf), err
}

// SetBlob inserts an object into the blobstore.
func (db *Blobstore) SetBlob(path string, data []byte) error {
	var blob = blob{}
	meddler.QueryRow(db, &blob, rebind(blobQuery), path)
	blob.Path = path
	blob.Data = string(data)
	return meddler.Save(db, blobTable, &blob)
}

// SetBlobReader inserts an object into the blobstore by
// consuming data from r until EOF.
func (db *Blobstore) SetBlobReader(path string, r io.Reader) error {
	var data, _ = ioutil.ReadAll(r)
	return db.SetBlob(path, data)
}

func NewBlobstore(db meddler.DB) *Blobstore {
	return &Blobstore{db}
}

// Blob table name in database.
const blobTable = "blobs"

const blobQuery = `
SELECT *
FROM blobs
WHERE blob_path = ?;
`

const blobDeleteStmt = `
DELETE FROM blobs
WHERE blob_path = ?;
`
