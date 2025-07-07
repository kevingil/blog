package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"blog-agent-go/backend/database"
	"blog-agent-go/backend/models"

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

func (s *ArticleService) GenerateArticle(ctx context.Context, prompt string, title string, authorID uint, draft bool) (*models.Article, error) {
	article, err := s.writerAgent.GenerateArticle(ctx, prompt, title, int64(authorID))
	if err != nil {
		return nil, fmt.Errorf("error generating article: %w", err)
	}

	gormArticle := &models.Article{
		Image:                    article.Image,
		Slug:                     article.Slug,
		Title:                    article.Title,
		Content:                  article.Content,
		AuthorID:                 authorID,
		IsDraft:                  draft,
		Embedding:                article.Embedding,
		ImageGenerationRequestID: article.ImageGenerationRequestID,
		PublishedAt:              article.PublishedAt,
		ChatHistory:              article.ChatHistory,
	}

	db := s.db.GetDB()
	result := db.Create(gormArticle)
	if result.Error != nil {
		return nil, result.Error
	}

	return gormArticle, nil
}

func (s *ArticleService) GetArticleChatHistory(ctx context.Context, articleID uint) (*ArticleChatHistory, error) {
	db := s.db.GetDB()
	var article models.Article

	result := db.Select("chat_history").First(&article, articleID)
	if result.Error != nil {
		return nil, result.Error
	}

	if article.ChatHistory == nil {
		return nil, nil
	}

	var history ArticleChatHistory
	if err := json.Unmarshal(article.ChatHistory, &history); err != nil {
		return nil, err
	}

	return &history, nil
}

func (s *ArticleService) UpdateArticle(ctx context.Context, articleID uint, req ArticleUpdateRequest) (*ArticleListItem, error) {
	db := s.db.GetDB()

	// Start transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if tx.Error != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Get existing article
	var article models.Article
	result := tx.First(&article, articleID)
	if result.Error != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to find article: %w", result.Error)
	}

	fmt.Println("Article content", req.Content)

	// Update article fields
	updates := models.Article{
		Title:       req.Title,
		Content:     req.Content,
		Image:       req.Image,
		IsDraft:     req.IsDraft,
		PublishedAt: req.PublishedAt,
	}

	result = tx.Model(&article).Updates(updates)
	if result.Error != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update article: %w", result.Error)
	}

	// Clear existing tags
	err := tx.Model(&article).Association("Tags").Clear()
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to clear existing tags: %w", err)
	}

	// Process tags
	var processedTags []string
	var tags []models.Tag

	for _, tagName := range req.Tags {
		if tagName == "" {
			continue
		}

		// Normalize tag name to lowercase and trim whitespace
		normalizedTag := strings.ToLower(strings.TrimSpace(tagName))
		if normalizedTag == "" {
			continue
		}

		// Avoid duplicate tags in the same request
		isDuplicate := false
		for _, existing := range processedTags {
			if existing == normalizedTag {
				isDuplicate = true
				break
			}
		}
		if isDuplicate {
			continue
		}
		processedTags = append(processedTags, normalizedTag)

		// Check if tag exists (case-insensitive lookup)
		var tag models.Tag
		result = tx.Where("LOWER(tag_name) = ?", normalizedTag).First(&tag)

		if result.Error == gorm.ErrRecordNotFound {
			// Tag doesn't exist, create it
			tag = models.Tag{TagName: normalizedTag}
			result = tx.Create(&tag)
			if result.Error != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to create tag '%s': %w", normalizedTag, result.Error)
			}
		} else if result.Error != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to check tag existence: %w", result.Error)
		}

		tags = append(tags, tag)
	}

	// Associate tags with article
	if len(tags) > 0 {
		if err := tx.Model(&article).Association("Tags").Append(tags); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to associate tags: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Get updated article with tags
	updatedArticle, err := s.GetArticle(articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated article: %w", err)
	}

	return updatedArticle, nil
}

func (s *ArticleService) UpdateArticleWithContext(ctx context.Context, articleID uint) (*models.Article, error) {
	db := s.db.GetDB()
	var article models.Article

	result := db.First(&article, articleID)
	if result.Error != nil {
		return nil, result.Error
	}

	// Convert to old format for writer agent
	oldArticle := &models.Article{
		ID:                       article.ID,
		CreatedAt:                article.CreatedAt,
		UpdatedAt:                article.UpdatedAt,
		Image:                    article.Image,
		Slug:                     article.Slug,
		Title:                    article.Title,
		Content:                  article.Content,
		AuthorID:                 article.AuthorID,
		IsDraft:                  article.IsDraft,
		Embedding:                article.Embedding,
		ImageGenerationRequestID: article.ImageGenerationRequestID,
		PublishedAt:              article.PublishedAt,
		ChatHistory:              article.ChatHistory,
	}

	updatedContent, err := s.writerAgent.UpdateWithContext(ctx, oldArticle)
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

func (s *ArticleService) GetArticleIDBySlug(slug string) (uint, error) {
	db := s.db.GetDB()
	var article models.Article

	result := db.Select("id").Where("slug = ?", slug).First(&article)
	if result.Error != nil {
		return 0, result.Error
	}

	return article.ID, nil
}

func (s *ArticleService) GetArticle(id uint) (*ArticleListItem, error) {
	db := s.db.GetDB()
	var article models.Article

	result := db.Preload("Author").Preload("Tags").First(&article, id)
	if result.Error != nil {
		return nil, result.Error
	}

	// Convert tags
	var tags []TagData
	for _, tag := range article.Tags {
		tags = append(tags, TagData{
			ArticleID: article.ID,
			TagID:     tag.TagID,
			TagName:   tag.TagName,
		})
	}

	return &ArticleListItem{
		Article: article,
		Author: AuthorData{
			ID:   article.Author.ID,
			Name: article.Author.Name,
		},
		Tags: tags,
	}, nil
}

func (s *ArticleService) GetArticles(page int, tag string, status string, articlesPerPage int) (*ArticleListResponse, error) {
	db := s.db.GetDB()
	var articles []models.Article
	var totalCount int64

	// Use default if articlesPerPage is not provided or invalid
	if articlesPerPage <= 0 {
		articlesPerPage = ITEMS_PER_PAGE
	}

	// Build query
	query := db.Model(&models.Article{}).Preload("Author").Preload("Tags")

	// Apply status filter
	switch status {
	case "published":
		query = query.Where("is_draft = ?", false)
	case "drafts":
		query = query.Where("is_draft = ?", true)
	case "all":
		// No filter - include all articles
	default:
		// Default to published only for invalid status
		query = query.Where("is_draft = ?", false)
	}

	if tag != "" {
		query = query.Joins("JOIN article_tags ON articles.id = article_tags.article_id").
			Joins("JOIN tags ON article_tags.tag_id = tags.tag_id").
			Where("LOWER(tags.tag_name) = ?", strings.ToLower(tag))
	}

	// Get total count
	countQuery := query
	countQuery.Count(&totalCount)

	// Calculate pagination
	offset := (page - 1) * articlesPerPage
	totalPages := int(math.Ceil(float64(totalCount) / float64(articlesPerPage)))

	// Get articles with pagination
	result := query.Order("created_at DESC").Offset(offset).Limit(articlesPerPage).Find(&articles)
	if result.Error != nil {
		return nil, result.Error
	}

	// Convert to response format
	var articleItems []ArticleListItem
	for _, article := range articles {
		var tags []TagData
		for _, tag := range article.Tags {
			tags = append(tags, TagData{
				ArticleID: article.ID,
				TagID:     tag.TagID,
				TagName:   tag.TagName,
			})
		}

		articleItems = append(articleItems, ArticleListItem{
			Article: article,
			Author: AuthorData{
				ID:   article.Author.ID,
				Name: article.Author.Name,
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

	// Build search query
	searchQuery := db.Model(&models.Article{}).Preload("Author").Preload("Tags").
		Where("is_draft = ?", false).
		Where("title LIKE ? OR content LIKE ?", "%"+query+"%", "%"+query+"%")

	if tag != "" {
		searchQuery = searchQuery.Joins("JOIN article_tags ON articles.id = article_tags.article_id").
			Joins("JOIN tags ON article_tags.tag_id = tags.tag_id").
			Where("LOWER(tags.tag_name) = ?", strings.ToLower(tag))
	}

	// Get total count
	countQuery := searchQuery
	countQuery.Count(&totalCount)

	// Calculate pagination
	offset := (page - 1) * ITEMS_PER_PAGE
	totalPages := int(math.Ceil(float64(totalCount) / float64(ITEMS_PER_PAGE)))

	// Get articles with pagination
	result := searchQuery.Order("created_at DESC").Offset(offset).Limit(ITEMS_PER_PAGE).Find(&articles)
	if result.Error != nil {
		return nil, result.Error
	}

	// Convert to response format
	var articleItems []ArticleListItem
	for _, article := range articles {
		var tags []TagData
		for _, tag := range article.Tags {
			tags = append(tags, TagData{
				ArticleID: article.ID,
				TagID:     tag.TagID,
				TagName:   tag.TagName,
			})
		}

		articleItems = append(articleItems, ArticleListItem{
			Article: article,
			Author: AuthorData{
				ID:   article.Author.ID,
				Name: article.Author.Name,
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
	var results []struct {
		TagName string `db:"tag_name"`
		Count   int
	}

	err := db.Table("tags").
		Select("tags.tag_name, COUNT(article_tags.tag_id) as count").
		Joins("JOIN article_tags ON tags.tag_id = article_tags.tag_id").
		Joins("JOIN articles ON article_tags.article_id = articles.id").
		Where("articles.is_draft = ?", false).
		Group("tags.tag_id, tags.tag_name").
		Order("count DESC").
		Limit(10).
		Scan(&results).Error

	if err != nil {
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
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type TagData struct {
	ArticleID uint   `json:"article_id"`
	TagID     uint   `json:"tag_id"`
	TagName   string `json:"tag_name"`
}

type RecommendedArticle struct {
	ID          uint    `json:"id"`
	Title       string  `json:"title"`
	Slug        string  `json:"slug"`
	Image       *string `json:"image"`
	PublishedAt *int64  `json:"published_at"`
	CreatedAt   int64   `json:"created_at"`
	Author      *string `json:"author"`
}

type ArticleRow struct {
	ID          uint     `json:"id"`
	Title       *string  `json:"title"`
	Content     *string  `json:"content"`
	CreatedAt   int64    `json:"created_at"`
	PublishedAt *int64   `json:"published_at"`
	IsDraft     bool     `json:"is_draft"`
	Slug        *string  `json:"slug"`
	Tags        []string `json:"tags"`
	Image       *string  `json:"image"`
}

func (s *ArticleService) GetArticleData(slug string) (*ArticleData, error) {
	db := s.db.GetDB()
	var article models.Article

	result := db.Preload("Author").Preload("Tags").Where("slug = ?", slug).First(&article)
	if result.Error != nil {
		return nil, result.Error
	}

	// Convert tags
	var tags []TagData
	for _, tag := range article.Tags {
		tags = append(tags, TagData{
			ArticleID: article.ID,
			TagID:     tag.TagID,
			TagName:   tag.TagName,
		})
	}

	return &ArticleData{
		Article: article,
		Author: AuthorData{
			ID:   article.Author.ID,
			Name: article.Author.Name,
		},
		Tags: tags,
	}, nil
}

func (s *ArticleService) GetRecommendedArticles(currentArticleID uint) ([]RecommendedArticle, error) {
	db := s.db.GetDB()
	var articles []models.Article

	// Get articles from same tags (simplified recommendation)
	result := db.Preload("Author").
		Where("id != ? AND is_draft = ?", currentArticleID, false).
		Order("created_at DESC").
		Limit(3).
		Find(&articles)

	if result.Error != nil {
		return nil, result.Error
	}

	var recommended []RecommendedArticle
	for _, article := range articles {
		var authorName *string
		if article.Author.Name != "" {
			authorName = &article.Author.Name
		}

		var image *string
		if article.Image != "" {
			image = &article.Image
		}

		recommended = append(recommended, RecommendedArticle{
			ID:          article.ID,
			Title:       article.Title,
			Slug:        article.Slug,
			Image:       image,
			PublishedAt: article.PublishedAt,
			CreatedAt:   article.CreatedAt,
			Author:      authorName,
		})
	}

	return recommended, nil
}

func (s *ArticleService) DeleteArticle(id uint) error {
	db := s.db.GetDB()

	// Start transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	var article models.Article
	result := tx.First(&article, id)
	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}

	// Clear tag associations
	err := tx.Model(&article).Association("Tags").Clear()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to clear tag associations: %w", err)
	}

	// Delete article
	result = tx.Delete(&article)
	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}

	return tx.Commit().Error
}

type ArticleCreateRequest struct {
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Image    string   `json:"image"`
	Tags     []string `json:"tags"`
	IsDraft  bool     `json:"isDraft"`
	AuthorID uint     `json:"authorId"`
}

func (s *ArticleService) CreateArticle(ctx context.Context, req ArticleCreateRequest) (*ArticleListItem, error) {
	db := s.db.GetDB()

	// Create article
	article := models.Article{
		Title:    req.Title,
		Content:  req.Content,
		Image:    req.Image,
		IsDraft:  req.IsDraft,
		AuthorID: req.AuthorID,
		Slug:     generateSlug(req.Title), // You'll need to implement this function
	}

	// Start transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if tx.Error != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	result := tx.Create(&article)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}

	// Process tags
	var tags []models.Tag
	for _, tagName := range req.Tags {
		if tagName == "" {
			continue
		}

		normalizedTag := strings.ToLower(strings.TrimSpace(tagName))
		if normalizedTag == "" {
			continue
		}

		var tag models.Tag
		result = tx.Where("LOWER(tag_name) = ?", normalizedTag).First(&tag)

		if result.Error == gorm.ErrRecordNotFound {
			tag = models.Tag{TagName: normalizedTag}
			result = tx.Create(&tag)
			if result.Error != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to create tag '%s': %w", normalizedTag, result.Error)
			}
		} else if result.Error != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to check tag existence: %w", result.Error)
		}

		tags = append(tags, tag)
	}

	// Associate tags with article
	if len(tags) > 0 {
		err := tx.Model(&article).Association("Tags").Append(tags)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to associate tags: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Get created article with all relations
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
