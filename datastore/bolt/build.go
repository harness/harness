package bolt

import (
	"bytes"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/drone/drone/common"
)

// GetBuild gets the specified build number for the
// named repository and build number
func (db *DB) GetBuild(repo string, build int) (*common.Build, error) {
	build_ := &common.Build{}
	key := []byte(repo + "/" + strconv.Itoa(build))
	err := get(db, bucketBuild, key, build_)
	return build_, err
}

// GetBuildList gets a list of recent builds for the
// named repository.
func (db *DB) GetBuildList(repo string) ([]*common.Build, error) {
	// get the last build sequence number (stored in key in `bucketBuildSeq`)
	// get all builds where build number > sequent-20
	// github.com/foo/bar/{number}
	return nil, nil
}

// GetBuildLast gets the last executed build for the
// named repository.
func (db *DB) GetBuildLast(repo string) (*common.Build, error) {
	// get the last build sequence number (stored in key in `bucketBuildSeq`)
	// return that build
	return nil, nil
}

// GetBuildStatus gets the named build status for the
// named repository and build number.
func (db *DB) GetBuildStatus(repo string, build int, status string) (*common.Status, error) {
	status_ := &common.Status{}
	key := []byte(repo + "/" + strconv.Itoa(build) + "/" + status)
	err := update(db, bucketBuildStatus, key, status)
	return status_, err
}

// GetBuildStatusList gets a list of all build statues for
// the named repository and build number.
func (db *DB) GetBuildStatusList(repo string, build int) ([]*common.Status, error) {
	// TODO (bradrydzewski) explore efficiency of cursor vs index

	statuses := []*common.Status{}
	err := db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketBuildStatus).Cursor()
		prefix := []byte(repo + "/" + strconv.Itoa(build) + "/")
		for k, v := c.Seek(prefix); bytes.HasPrefix(k, prefix); k, v = c.Next() {
			status := &common.Status{}
			if err := decode(v, status); err != nil {
				return err
			}
			statuses = append(statuses, status)
		}
		return nil
	})
	return statuses, err
}

// InsertBuild inserts a new build for the named repository
func (db *DB) InsertBuild(repo string, build *common.Build) error {
	// TODO(bradrydzewski) use the `bucketBuildSeq` to increment the
	//                     sequence for the build and set the build number.
	key := []byte(repo + "/" + strconv.Itoa(build.Number))
	return update(db, bucketBuild, key, build)
}

// InsertBuildStatus inserts a new build status for the
// named repository and build number. If the status already
// exists an error is returned.
func (db *DB) InsertBuildStatus(repo string, build int, status *common.Status) error {
	key := []byte(repo + "/" + strconv.Itoa(build) + "/" + status.Context)
	return update(db, bucketBuildStatus, key, status)
}

// UpdateBuild updates an existing build for the named
// repository. If the build already exists and error is
// returned.
func (db *DB) UpdateBuild(repo string, build *common.Build) error {
	key := []byte(repo + "/" + strconv.Itoa(build.Number))
	build.Updated = time.Now().UTC().Unix()
	return update(db, bucketBuild, key, build)
}
