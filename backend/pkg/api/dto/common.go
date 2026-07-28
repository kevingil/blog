package dto

// PaginationParams represents common pagination parameters
type PaginationParams struct {
	Page    int `json:"page" validate:"omitempty,min=1"`
	PerPage int `json:"per_page" validate:"omitempty,min=1,max=100"`
}

// IDParam represents a UUID path parameter
type IDParam struct {
	ID string `json:"id" validate:"required,uuid"`
}

// SlugParam represents a slug path parameter
type SlugParam struct {
	Slug string `json:"slug" validate:"required,min=1,max=200"`
}
