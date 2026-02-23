package api

// ListDto contains query parameters for listing resources
type ListDto struct {
	Page    string `query:"page"`
	PerPage string `query:"per_page"`
	From    string `query:"from"`
	To      string `query:"to"`
	Query   string `query:"query"`
}

// GetDto contains parameters for fetching a single resource
type GetDto struct {
	ID string `param:"id"`
}

// UpdateDto contains parameters for updating a resource
type UpdateDto[T any] struct {
	ID string `param:"id"`

	Body T `json:"body,merge"`
}

// DeleteDto contains parameters for deleting a resource
type DeleteDto struct {
	ID string `param:"id"`
}
