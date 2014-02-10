package migrate

import (
	"database/sql"
)

type Change_20150209 struct{}

func (Change_20150209) Up(tx *sql.Tx) error {
	return nil
}

func (Change_20150209) Down(tx *sql.Tx) error {
	return nil
}

func (Change_20150209) Revision() int64 {
	return 20150209
}
