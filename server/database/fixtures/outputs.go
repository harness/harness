package fixtures

import (
	"database/sql"

	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

func LoadOutputs(db *sql.DB) {
	meddler.Insert(db, "output", &model.Output{
		CommitID: 1,
		Raw:      []byte("sample console output"),
	})

	meddler.Insert(db, "output", &model.Output{
		CommitID: 2,
		Raw:      []byte("sample console output"),
	})
}
