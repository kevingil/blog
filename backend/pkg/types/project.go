package types

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a portfolio project
type Project struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	TagIDs      []int64   `json:"tag_ids,omitempty"`
	ImageURL    string    `json:"image_url,omitempty"`
	URL         string    `json:"url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProjectListOptions represents options for listing projects
type ProjectListOptions struct {
	Page    int
	PerPage int
}
