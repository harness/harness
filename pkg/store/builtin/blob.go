package builtin

import (
	"bytes"
	"database/sql"
	"io"
	"io/ioutil"
)

type Blob struct {
	ID   int64
	Path string `sql:"unique:ux_blob_path"`
	Data []byte
}

type Blobstore struct {
	*sql.DB
}

// Del removes an object from the blobstore.
func (db *Blobstore) DelBlob(path string) error {
	blob, _ := getBlob(db, rebind(stmtBlobSelectBlobPath), path)
	if blob == nil {
		return nil
	}
	_, err := db.Exec(rebind(stmtBlobDelete), blob.ID)
	return err
}

// Get retrieves an object from the blobstore.
func (db *Blobstore) GetBlob(path string) ([]byte, error) {
	blob, err := getBlob(db, rebind(stmtBlobSelectBlobPath), path)
	if err != nil {
		return nil, nil
	}
	return blob.Data, nil
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
	blob, _ := getBlob(db, rebind(stmtBlobSelectBlobPath), path)
	if blob == nil {
		blob = &Blob{}
	}
	blob.Path = path
	blob.Data = data
	if blob.ID == 0 {
		return createBlob(db, rebind(stmtBlobInsert), blob)
	}
	return updateBlob(db, rebind(stmtBlobUpdate), blob)
}

// SetBlobReader inserts an object into the blobstore by
// consuming data from r until EOF.
func (db *Blobstore) SetBlobReader(path string, r io.Reader) error {
	var data, _ = ioutil.ReadAll(r)
	return db.SetBlob(path, data)
}

func NewBlobstore(db *sql.DB) *Blobstore {
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
