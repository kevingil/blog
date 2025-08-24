package services

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"blog-agent-go/backend/internal/database"
	"blog-agent-go/backend/internal/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type ArticleService struct {
	db          database.Service
	writerAgent *WriterAgent
}

func NewArticleService(db database.Service, writerAgent *WriterAgent) *ArticleService {
	return &ArticleService{
		db:          db,
		writerAgent: writerAgent,
	}
}

type ArticleChatHistoryMessage struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"`
	Metadata  any    `json:"metadata"`
}

type ArticleChatHistory struct {
	Messages []ArticleChatHistoryMessage `json:"messages"`
	Metadata any                         `json:"metadata"`
}

type ArticleListItem struct {
	Article models.Article `json:"article"`
	Author  AuthorData     `json:"author"`
	Tags    []TagData      `json:"tags"`
}

type ArticleUpdateRequest struct {
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Image       string   `json:"image"`
	Tags        []string `json:"tags"`
	IsDraft     bool     `json:"is_draft"`
	PublishedAt *int64   `json:"published_at"`
}

type ArticleListResponse struct {
	Articles      []ArticleListItem `json:"articles"`
	TotalPages    int               `json:"total_pages"`
	IncludeDrafts bool              `json:"include_drafts"`
}

const ITEMS_PER_PAGE = 6

func (s *ArticleService) GenerateArticle(ctx context.Context, prompt string, title string, authorID uuid.UUID, draft bool) (*models.Article, error) {
	article, err := s.writerAgent.GenerateArticle(ctx, prompt, title, authorID)
	if err != nil {
		return nil, fmt.Errorf("error generating article: %w", err)
	}

	gormArticle := &models.Article{
		ImageURL:        article.ImageURL,
		Slug:            article.Slug,
		Title:           article.Title,
		Content:         article.Content,
		AuthorID:        authorID,
		IsDraft:         draft,
		Embedding:       article.Embedding,
		ImagenRequestID: article.ImagenRequestID,
		PublishedAt:     article.PublishedAt,
		SessionMemory:   article.SessionMemory,
	}

	db := s.db.GetDB()
	result := db.Create(gormArticle)
	if result.Error != nil {
		return nil, result.Error
	}

	return gormArticle, nil
}

func (s *ArticleService) UpdateArticle(ctx context.Context, articleID uuid.UUID, req ArticleUpdateRequest) (*ArticleListItem, error) {
	db := s.db.GetDB()

	var article models.Article
	result := db.First(&article, articleID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find article: %w", result.Error)
	}

	// Process tags: ensure all exist, collect their IDs
	var tagIDs []int64
	for _, tagName := range req.Tags {
		tagName = strings.ToLower(strings.TrimSpace(tagName))
		if tagName == "" {
			continue
		}
		var tag models.Tag
		result := db.Where("LOWER(name) = ?", tagName).First(&tag)
		if result.Error == gorm.ErrRecordNotFound {
			tag = models.Tag{Name: tagName}
			if err := db.Create(&tag).Error; err != nil {
				return nil, fmt.Errorf("failed to create tag '%s': %w", tagName, err)
			}
		} else if result.Error != nil {
			return nil, fmt.Errorf("failed to check tag existence: %w", result.Error)
		}
		tagIDs = append(tagIDs, int64(tag.ID))
	}

	// Update article fields directly
	article.Title = req.Title
	article.Content = req.Content
	article.ImageURL = req.Image
	article.IsDraft = req.IsDraft

	// Convert Unix timestamp to time.Time if provided
	if req.PublishedAt != nil {
		var t time.Time
		// Check if timestamp is in milliseconds (typical for JavaScript)
		if *req.PublishedAt > 1e12 {
			// Timestamp is in milliseconds
			t = time.Unix(0, *req.PublishedAt*int64(time.Millisecond))
		} else {
			// Timestamp is in seconds
			t = time.Unix(*req.PublishedAt, 0)
		}
		article.PublishedAt = &t
	}
	if len(tagIDs) > 0 {
		article.TagIDs = pq.Int64Array(tagIDs)
	}

	result = db.Save(&article)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update article: %w", result.Error)
	}

	return s.GetArticle(article.ID)
}

func (s *ArticleService) UpdateArticleWithContext(ctx context.Context, articleID uuid.UUID) (*models.Article, error) {
	db := s.db.GetDB()
	var article models.Article

	result := db.First(&article, articleID)
	if result.Error != nil {
		return nil, result.Error
	}

	// If writerAgent requires *models.Article, pass only the fields it needs
	updatedContent, err := s.writerAgent.UpdateWithContext(ctx, &article)
	if err != nil {
		return nil, fmt.Errorf("error updating article content: %w", err)
	}

	article.Content = updatedContent

	result = db.Model(&article).Update("content", updatedContent)
	if result.Error != nil {
		return nil, result.Error
	}

	return &article, nil
}

func (s *ArticleService) GetArticleIDBySlug(slug string) (uuid.UUID, error) {
	db := s.db.GetDB()
	var article models.Article

	result := db.Select("id").Where("slug = ?", slug).First(&article)
	if result.Error != nil {
		return uuid.UUID{}, result.Error
	}

	return article.ID, nil
}

// --- FULL REFACTOR FOR NEW SCHEMA ---
// All list/detail methods now use the new schema from 20250723064003_init.sql.
// Remove all legacy GORM relations, join table logic, and references to removed fields.
// For tag handling, use tag_ids integer array on Article and fetch tag names from tag table.
// For author, fetch from account table using author_id.
// Only use fields present in the new schema.

func (s *ArticleService) GetArticle(id uuid.UUID) (*ArticleListItem, error) {
	db := s.db.GetDB()
	var article models.Article
	if err := db.First(&article, id).Error; err != nil {
		return nil, err
	}

	// Parse tag IDs
	var tagIDs []int64
	tagIDs = article.TagIDs

	// Fetch tag names
	var tags []TagData
	if len(tagIDs) > 0 {
		var dbTags []models.Tag
		if err := db.Where("id IN ?", tagIDs).Find(&dbTags).Error; err == nil {
			for _, t := range dbTags {
				tags = append(tags, TagData{
					ArticleID: article.ID,
					TagID:     int(t.ID),
					TagName:   t.Name,
				})
			}
		}
	}

	// Fetch author
	var authorName string
	var account models.Account
	if err := db.First(&account, "id = ?", article.AuthorID).Error; err == nil {
		authorName = account.Name
	}

	return &ArticleListItem{
		Article: article,
		Author: AuthorData{
			ID:   article.AuthorID,
			Name: authorName,
		},
		Tags: tags,
	}, nil
}

func (s *ArticleService) GetArticles(page int, tag string, status string, articlesPerPage int) (*ArticleListResponse, error) {
	db := s.db.GetDB()
	var articles []models.Article
	var totalCount int64

	if articlesPerPage <= 0 {
		articlesPerPage = ITEMS_PER_PAGE
	}

	query := db.Model(&models.Article{})

	switch status {
	case "published":
		query = query.Where("is_draft = ?", false)
	case "drafts":
		query = query.Where("is_draft = ?", true)
	case "all":
		// No filter
	default:
		query = query.Where("is_draft = ?", false)
	}

	if tag != "" {
		// Find tag ID by name
		var tagModel models.Tag
		if err := db.Where("LOWER(name) = ?", strings.ToLower(tag)).First(&tagModel).Error; err == nil {
			query = query.Where("tag_ids @> ARRAY[?]::integer[]", tagModel.ID)
		}
	}

	countQuery := query
	countQuery.Count(&totalCount)

	offset := (page - 1) * articlesPerPage
	totalPages := int(math.Ceil(float64(totalCount) / float64(articlesPerPage)))

	result := query.Order("created_at DESC").Offset(offset).Limit(articlesPerPage).Find(&articles)
	if result.Error != nil {
		return nil, result.Error
	}

	articleItems := make([]ArticleListItem, 0)
	for _, article := range articles {
		// Parse tag IDs
		var tagIDs []int64
		tagIDs = article.TagIDs
		// Fetch tag names
		var tags []TagData
		if len(tagIDs) > 0 {
			var dbTags []models.Tag
			if err := db.Where("id IN ?", tagIDs).Find(&dbTags).Error; err == nil {
				for _, t := range dbTags {
					tags = append(tags, TagData{
						ArticleID: article.ID,
						TagID:     int(t.ID),
						TagName:   t.Name,
					})
				}
			}
		}
		// Fetch author
		var authorName string
		var account models.Account
		if err := db.First(&account, "id = ?", article.AuthorID).Error; err == nil {
			authorName = account.Name
		}
		articleItems = append(articleItems, ArticleListItem{
			Article: article,
			Author: AuthorData{
				ID:   article.AuthorID,
				Name: authorName,
			},
			Tags: tags,
		})
	}

	return &ArticleListResponse{
		Articles:      articleItems,
		TotalPages:    totalPages,
		IncludeDrafts: status == "all" || status == "drafts",
	}, nil
}

func (s *ArticleService) SearchArticles(query string, page int, tag string) (*ArticleListResponse, error) {
	db := s.db.GetDB()
	var articles []models.Article
	var totalCount int64

	searchQuery := db.Model(&models.Article{}).
		Where("is_draft = ?", false).
		Where("title LIKE ? OR content LIKE ? OR EXISTS (SELECT 1 FROM tag WHERE id = ANY(article.tag_ids) AND name ILIKE ?)", "%"+query+"%", "%"+query+"%", "%"+query+"%")

	if tag != "" {
		var tagModel models.Tag
		if err := db.Where("LOWER(name) = ?", strings.ToLower(tag)).First(&tagModel).Error; err == nil {
			searchQuery = searchQuery.Where("tag_ids @> ARRAY[?]::integer[]", tagModel.ID)
		}
	}

	countQuery := searchQuery
	countQuery.Count(&totalCount)

	offset := (page - 1) * ITEMS_PER_PAGE
	totalPages := int(math.Ceil(float64(totalCount) / float64(ITEMS_PER_PAGE)))

	result := searchQuery.Order("created_at DESC").Offset(offset).Limit(ITEMS_PER_PAGE).Find(&articles)
	if result.Error != nil {
		return nil, result.Error
	}

	articleItems := make([]ArticleListItem, 0)
	for _, article := range articles {
		var tagIDs []int64
		tagIDs = article.TagIDs
		var tags []TagData
		if len(tagIDs) > 0 {
			var dbTags []models.Tag
			if err := db.Where("id IN ?", tagIDs).Find(&dbTags).Error; err == nil {
				for _, t := range dbTags {
					tags = append(tags, TagData{
						ArticleID: article.ID,
						TagID:     int(t.ID),
						TagName:   t.Name,
					})
				}
			}
		}
		var authorName string
		var account models.Account
		if err := db.First(&account, "id = ?", article.AuthorID).Error; err == nil {
			authorName = account.Name
		}
		articleItems = append(articleItems, ArticleListItem{
			Article: article,
			Author: AuthorData{
				ID:   article.AuthorID,
				Name: authorName,
			},
			Tags: tags,
		})
	}

	return &ArticleListResponse{
		Articles:      articleItems,
		TotalPages:    totalPages,
		IncludeDrafts: false,
	}, nil
}

func (s *ArticleService) GetPopularTags() ([]string, error) {
	db := s.db.GetDB()
	// Use raw SQL to unnest tag_ids and count tag usage
	var results []struct {
		TagName string
		Count   int
	}
	sql := `
SELECT tag.name AS tag_name, COUNT(*) AS count
FROM article, unnest(tag_ids) AS tag_id
JOIN tag ON tag.id = tag_id
WHERE article.is_draft = false
GROUP BY tag.name
ORDER BY count DESC
LIMIT 10`
	if err := db.Raw(sql).Scan(&results).Error; err != nil {
		return nil, err
	}

	var tags []string
	for _, result := range results {
		tags = append(tags, result.TagName)
	}
	return tags, nil
}

type ArticleMetadata struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type ArticleData struct {
	Article models.Article `json:"article"`
	Tags    []TagData      `json:"tags"`
	Author  AuthorData     `json:"author"`
}

type AuthorData struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type TagData struct {
	ArticleID uuid.UUID `json:"article_id"`
	TagID     int       `json:"tag_id"`
	TagName   string    `json:"name"`
}

type RecommendedArticle struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	ImageURL    *string   `json:"image_url"`
	PublishedAt *string   `json:"published_at"`
	CreatedAt   string    `json:"created_at"`
	Author      *string   `json:"author"`
}

type ArticleRow struct {
	ID          uuid.UUID `json:"id"`
	Title       *string   `json:"title"`
	Content     *string   `json:"content"`
	CreatedAt   string    `json:"created_at"`
	PublishedAt *string   `json:"published_at"`
	IsDraft     bool      `json:"is_draft"`
	Slug        *string   `json:"slug"`
	TagIDs      []int     `json:"tag_ids"`
	ImageURL    *string   `json:"image_url"`
}

func (s *ArticleService) GetArticleData(slug string) (*ArticleData, error) {
	db := s.db.GetDB()
	var article models.Article
	if err := db.Where("slug = ?", slug).First(&article).Error; err != nil {
		return nil, err
	}

	var tagIDs []int64
	tagIDs = article.TagIDs

	var tags []TagData
	if len(tagIDs) > 0 {
		var dbTags []models.Tag
		if err := db.Where("id IN ?", tagIDs).Find(&dbTags).Error; err == nil {
			for _, t := range dbTags {
				tags = append(tags, TagData{
					ArticleID: article.ID,
					TagID:     int(t.ID),
					TagName:   t.Name,
				})
			}
		}
	}

	var authorName string
	var account models.Account
	if err := db.First(&account, "id = ?", article.AuthorID).Error; err == nil {
		authorName = account.Name
	}

	return &ArticleData{
		Article: article,
		Author: AuthorData{
			ID:   article.AuthorID,
			Name: authorName,
		},
		Tags: tags,
	}, nil
}

func (s *ArticleService) GetRecommendedArticles(currentArticleID uuid.UUID) ([]RecommendedArticle, error) {
	db := s.db.GetDB()
	var articles []models.Article

	result := db.Where("id != ? AND is_draft = ?", currentArticleID, false).
		Order("created_at DESC").
		Limit(3).
		Find(&articles)

	if result.Error != nil {
		return nil, result.Error
	}

	var recommended []RecommendedArticle
	for _, article := range articles {
		var authorName *string
		var account models.Account
		if err := db.First(&account, "id = ?", article.AuthorID).Error; err == nil {
			authorName = &account.Name
		}

		var image *string
		if article.ImageURL != "" {
			image = &article.ImageURL
		}

		recommended = append(recommended, RecommendedArticle{
			ID:          article.ID,
			Title:       article.Title,
			Slug:        article.Slug,
			ImageURL:    image,
			PublishedAt: safeTimeToString(article.PublishedAt),
			CreatedAt:   article.CreatedAt.UTC().Format(time.RFC3339),
			Author:      authorName,
		})
	}

	return recommended, nil
}

func (s *ArticleService) DeleteArticle(id uuid.UUID) error {
	db := s.db.GetDB()
	return db.Delete(&models.Article{}, id).Error
}

type ArticleCreateRequest struct {
	Title    string    `json:"title"`
	Content  string    `json:"content"`
	Image    string    `json:"image"`
	Tags     []string  `json:"tags"`
	IsDraft  bool      `json:"isDraft"`
	AuthorID uuid.UUID `json:"authorId"`
}

func (s *ArticleService) CreateArticle(ctx context.Context, req ArticleCreateRequest) (*ArticleListItem, error) {
	db := s.db.GetDB()

	// Process tags: ensure all exist, collect their IDs
	var tagIDs []int64
	for _, tagName := range req.Tags {
		tagName = strings.ToLower(strings.TrimSpace(tagName))
		if tagName == "" {
			continue
		}
		var tag models.Tag
		result := db.Where("LOWER(name) = ?", tagName).First(&tag)
		if result.Error == gorm.ErrRecordNotFound {
			tag = models.Tag{Name: tagName}
			if err := db.Create(&tag).Error; err != nil {
				return nil, fmt.Errorf("failed to create tag '%s': %w", tagName, err)
			}
		} else if result.Error != nil {
			return nil, fmt.Errorf("failed to check tag existence: %w", result.Error)
		}
		tagIDs = append(tagIDs, int64(tag.ID))
	}

	// Create article
	article := models.Article{
		Title:    req.Title,
		Content:  req.Content,
		ImageURL: req.Image,
		IsDraft:  req.IsDraft,
		AuthorID: req.AuthorID,
		Slug:     generateSlug(req.Title),
	}
	if len(tagIDs) > 0 {
		article.TagIDs = tagIDs
	}

	result := db.Create(&article)
	if result.Error != nil {
		return nil, result.Error
	}

	return s.GetArticle(article.ID)
}

// Helper function to generate slug from title
func generateSlug(title string) string {
	// Basic slug generation - you might want to use a proper slug library
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters (basic implementation)
	return slug
}

func safeTimeToString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	year := t.Year()
	if year < 0 || year > 9999 {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}
