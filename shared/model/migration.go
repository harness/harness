package model

type Migrate struct {
	Revision int64 `gorm:"primary_key:yes"`
}
