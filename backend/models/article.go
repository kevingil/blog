package models

type Article struct {
	ID                       uint   `json:"id" gorm:"primaryKey"`
	CreatedAt                int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt                int64  `json:"updated_at" gorm:"autoUpdateTime"`
	Image                    string `json:"image"`
	Slug                     string `json:"slug" gorm:"uniqueIndex;not null"`
	Title                    string `json:"title" gorm:"not null"`
	Content                  string `json:"content" gorm:"type:text"`
	AuthorID                 uint   `json:"author" gorm:"not null;column:author"`
	Author                   User   `json:"author_data" gorm:"foreignKey:AuthorID"`
	IsDraft                  bool   `json:"is_draft" gorm:"default:true"`
	Embedding                []byte `json:"-" gorm:"type:blob"`
	ImageGenerationRequestID string `json:"image_generation_request_id"`
	PublishedAt              *int64 `json:"published_at,omitempty"`
	ChatHistory              []byte `json:"chat_history" gorm:"type:blob"`
	Tags                     []Tag  `json:"tags" gorm:"many2many:article_tags;joinForeignKey:article_id;joinReferences:tag_id"`
}

type Tag struct {
	TagID    uint      `json:"id" gorm:"primaryKey;column:tag_id"`
	TagName  string    `json:"tag_name" gorm:"uniqueIndex;not null;column:tag_name"`
	Articles []Article `json:"articles" gorm:"many2many:article_tags;joinForeignKey:tag_id;joinReferences:article_id"`
}

type ArticleTag struct {
	ArticleID uint    `json:"article_id" gorm:"primaryKey"`
	TagID     uint    `json:"tag_id" gorm:"primaryKey"`
	Article   Article `gorm:"foreignKey:ArticleID"`
	Tag       Tag     `gorm:"foreignKey:TagID;references:TagID"`
}
