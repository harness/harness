package types

import "github.com/harness/gitness/types/enum"

// Pagination stores pagination related params
type Pagination struct {
	Page  int        `json:"page"`
	Size  int        `json:"size"`
	Query string     `json:"query"`
	Sort  string     `json:"sort"`
	Order enum.Order `json:"order"`
}
