package models

type AboutPage struct {
	ID              uint   `json:"id" gorm:"primaryKey"`
	Title           string `json:"title" gorm:"not null"`
	Content         string `json:"content" gorm:"type:text"`
	ProfileImage    string `json:"profile_image"`
	MetaDescription string `json:"meta_description"`
	LastUpdated     string `json:"last_updated"`
}

func (AboutPage) TableName() string {
	return "about_page"
}

type ContactPage struct {
	ID              uint   `json:"id" gorm:"primaryKey"`
	Title           string `json:"title" gorm:"not null"`
	Content         string `json:"content" gorm:"type:text"`
	EmailAddress    string `json:"email_address" gorm:"not null"`
	SocialLinks     string `json:"social_links" gorm:"type:text"`
	MetaDescription string `json:"meta_description"`
	LastUpdated     string `json:"last_updated"`
}

func (ContactPage) TableName() string {
	return "contact_page"
}

type Project struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Title       string `json:"title" gorm:"not null"`
	Description string `json:"description" gorm:"type:text"`
	URL         string `json:"url"`
	Image       string `json:"image"`
}
