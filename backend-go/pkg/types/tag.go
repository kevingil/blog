package types

import (
	"time"
)

// Tag represents a tag for categorizing articles and projects
type Tag struct {
	ID        int
	Name      string
	CreatedAt time.Time
}
