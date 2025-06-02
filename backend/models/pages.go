package models

type AboutPage struct {
	Title           string `json:"title"`
	Content         string `json:"content"`
	ProfileImage    string `json:"profile_image"`
	MetaDescription string `json:"meta_description"`
	LastUpdated     string `json:"last_updated"`
}

type ContactPage struct {
	Title           string `json:"title"`
	Content         string `json:"content"`
	EmailAddress    string `json:"email_address"`
	SocialLinks     string `json:"social_links"`
	MetaDescription string `json:"meta_description"`
	LastUpdated     string `json:"last_updated"`
}

type Project struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Image       string `json:"image"`
}
