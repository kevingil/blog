package models

type Article struct {
	ID                       int64  `json:"id"`
	Image                    string `json:"image"`
	Slug                     string `json:"slug"`
	Title                    string `json:"title"`
	Content                  string `json:"content"`
	Author                   int64  `json:"author"`
	CreatedAt                int64  `json:"created_at"`
	UpdatedAt                int64  `json:"updated_at"`
	IsDraft                  bool   `json:"is_draft"`
	Embedding                []byte `json:"-"`
	ImageGenerationRequestID string `json:"image_generation_request_id"`
	PublishedAt              *int64 `json:"published_at,omitempty"`
	ChatHistory              []byte `json:"chat_history"`
}

type Tag struct {
	ID   int64  `json:"tag_id"`
	Name string `json:"tag_name"`
}

type ArticleTag struct {
	ArticleID int64 `json:"article_id"`
	TagID     int64 `json:"tag_id"`
}
