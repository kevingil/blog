package types

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a portfolio project
type Project struct {
	ID          uuid.UUID
	Title       string
	Description string
	Content     string
	TagIDs      []int64
	ImageURL    string
	URL         string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProjectListOptions represents options for listing projects
type ProjectListOptions struct {
	Page    int
	PerPage int
}
