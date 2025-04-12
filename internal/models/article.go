package models

type Article struct {
	ID                       int64  `gorm:"primaryKey" json:"id"`
	Image                    string `gorm:"type:varchar(255)" json:"image"`
	Slug                     string `gorm:"type:varchar(255);uniqueIndex" json:"slug"`
	Title                    string `gorm:"type:varchar(255)" json:"title"`
	Content                  string `gorm:"type:text" json:"content"`
	Author                   int64  `gorm:"index" json:"author"`
	CreatedAt                int64  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt                int64  `gorm:"autoUpdateTime" json:"updated_at"`
	IsDraft                  bool   `gorm:"default:true" json:"is_draft"`
	Embedding                []byte `gorm:"type:bytea" json:"-"`
	ImageGenerationRequestID string `gorm:"type:varchar(255)" json:"image_generation_request_id"`
	PublishedAt              *int64 `gorm:"index" json:"published_at,omitempty"`
	Tags                     []Tag  `gorm:"many2many:article_tags;" json:"-"`
}

type Tag struct {
	ID       int64     `gorm:"primaryKey" json:"id"`
	Name     string    `gorm:"type:varchar(255);uniqueIndex" json:"name"`
	Articles []Article `gorm:"many2many:article_tags;" json:"-"`
}

type ArticleTag struct {
	ArticleID int64   `gorm:"primaryKey" json:"article_id"`
	TagID     int64   `gorm:"primaryKey" json:"tag_id"`
	Article   Article `gorm:"foreignKey:ArticleID;constraint:OnDelete:CASCADE"`
	Tag       Tag     `gorm:"foreignKey:TagID;constraint:OnDelete:CASCADE"`
}
