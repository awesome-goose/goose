package web

// IndexDto contains query parameters for listing resources
type IndexDto struct {
	Page    string `query:"page"`
	PerPage string `query:"per_page"`
	From    string `query:"from"`
	To      string `query:"to"`
	Query   string `query:"query"`
}

// CreateDto is empty as the create form doesn't need parameters
type CreateDto struct{}

// StoreDto contains the body for creating a new resource
type StoreDto[T any] struct {
	Body T `form:"body,merge"`
}

// ShowDto contains parameters for viewing a single resource
type ShowDto struct {
	ID string `param:"id"`
}

// EditDto contains parameters for editing a resource
type EditDto struct {
	ID string `param:"id"`
}

// UpdateDto contains parameters for updating a resource
type UpdateDto[T any] struct {
	ID string `param:"id"`

	Body T `form:"body,merge"`
}

// DestroyDto contains parameters for deleting a resource
type DestroyDto struct {
	ID string `param:"id"`
}
