package types

// ListQueryFilter has pagination related info and a query param
type ListQueryFilter struct {
	Pagination
	Query string `json:"query"`
}
