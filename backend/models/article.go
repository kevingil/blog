package models

import (
	"time"
)

type Article struct {
	ID                       uint       `json:"id" gorm:"primaryKey"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
	Image                    string     `json:"image"`
	Slug                     string     `json:"slug" gorm:"uniqueIndex;not null"`
	Title                    string     `json:"title" gorm:"not null"`
	Content                  string     `json:"content" gorm:"type:text"`
	AuthorID                 uint       `json:"author" gorm:"not null"`
	Author                   User       `json:"author_data" gorm:"foreignKey:AuthorID"`
	IsDraft                  bool       `json:"is_draft" gorm:"default:true"`
	Embedding                []byte     `json:"-" gorm:"type:blob"`
	ImageGenerationRequestID string     `json:"image_generation_request_id"`
	PublishedAt              *time.Time `json:"published_at,omitempty"`
	ChatHistory              []byte     `json:"chat_history" gorm:"type:blob"`
	Tags                     []Tag      `json:"tags" gorm:"many2many:article_tags;"`
}

type Tag struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"tag_name" gorm:"uniqueIndex;not null"`
	Articles  []Article `json:"articles" gorm:"many2many:article_tags;"`
}

type ArticleTag struct {
	ArticleID uint    `json:"article_id" gorm:"primaryKey"`
	TagID     uint    `json:"tag_id" gorm:"primaryKey"`
	Article   Article `gorm:"foreignKey:ArticleID"`
	Tag       Tag     `gorm:"foreignKey:TagID"`
}
