package model

type Output struct {
	Id        int64 `gorm:"primary_key:yes"`
	CommitId  int64
	OutputRaw string
}
