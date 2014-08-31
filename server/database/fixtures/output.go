package fixtures

import (
	"github.com/drone/drone/shared/model"
	"github.com/jinzhu/gorm"
)

func LoadOutput(db *gorm.DB) {
	db.Table("output").Create(model.Output{
		CommitId:  1,
		OutputRaw: "sample console output",
	})

	db.Table("output").Create(model.Output{
		CommitId:  2,
		OutputRaw: "sample console output",
	})
}
