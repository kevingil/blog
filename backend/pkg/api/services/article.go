package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"regexp"
	"strings"
	"time"

	"backend/pkg/core"
	"backend/pkg/core/ml"
	"backend/pkg/database"
	"backend/pkg/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

type ArticleService struct {
	db               database.Service
	writerAgent      *WriterAgent
	tagService       *TagService
	embeddingService *ml.EmbeddingService
}

func NewArticleService(db database.Service, writerAgent *WriterAgent) *ArticleService {
	return &ArticleService{
		db:               db,
		writerAgent:      writerAgent,
		tagService:       NewTagService(db),
		embeddingService: ml.NewEmbeddingService(),
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
	Title       string   `json:"title" validate:"required,min=3,max=200"`
	Content     string   `json:"content" validate:"required,min=10"`
	ImageURL    string   `json:"image_url" validate:"omitempty,url"`
	Tags        []string `json:"tags" validate:"max=10,dive,min=2,max=30"`
	PublishedAt *int64   `json:"published_at"`
}

type ArticleListResponse struct {
	Articles      []ArticleListItem `json:"articles"`
	TotalPages    int               `json:"total_pages"`
	IncludeDrafts bool              `json:"include_drafts"`
}

type ArticleVersionResponse struct {
	ID            uuid.UUID `json:"id"`
	ArticleID     uuid.UUID `json:"article_id"`
	VersionNumber int       `json:"version_number"`
	Status        string    `json:"status"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	ImageURL      string    `json:"image_url"`
	CreatedAt     string    `json:"created_at"`
}

type ArticleVersionListResponse struct {
	Versions []ArticleVersionResponse `json:"versions"`
	Total    int                      `json:"total"`
}

const ITEMS_PER_PAGE = 6

func (s *ArticleService) GenerateArticle(ctx context.Context, prompt string, title string, authorID uuid.UUID, publish bool) (*models.Article, error) {
	article, err := s.writerAgent.GenerateArticle(ctx, prompt, title, authorID)
	if err != nil {
		return nil, fmt.Errorf("error generating article: %w", err)
	}

	// Generate slug from title
	slug := generateSlug(title)

	gormArticle := &models.Article{
		DraftImageURL:   article.DraftImageURL,
		Slug:            slug,
		DraftTitle:      article.DraftTitle,
		DraftContent:    article.DraftContent,
		AuthorID:        authorID,
		DraftEmbedding:  article.DraftEmbedding,
		ImagenRequestID: article.ImagenRequestID,
		SessionMemory:   article.SessionMemory,
	}

	// If publish is requested, copy draft to published
	if publish {
		now := time.Now()
		gormArticle.PublishedTitle = &article.DraftTitle
		gormArticle.PublishedContent = &article.DraftContent
		gormArticle.PublishedImageURL = &article.DraftImageURL
		gormArticle.PublishedEmbedding = gormArticle.DraftEmbedding
		gormArticle.PublishedAt = &now
	}

	db := s.db.GetDB()
	result := db.Create(gormArticle)
	if result.Error != nil {
		return nil, result.Error
	}

	// Create initial version asynchronously
	status := "draft"
	if publish {
		status = "published"
	}
	go s.createVersionAsync(gormArticle.ID, article.DraftTitle, article.DraftContent, article.DraftImageURL, article.DraftEmbedding.Slice(), status, authorID)

	return gormArticle, nil
}

func (s *ArticleService) UpdateArticle(ctx context.Context, articleID uuid.UUID, req ArticleUpdateRequest) (*ArticleListItem, error) {
	db := s.db.GetDB()

	var article models.Article
	result := db.First(&article, articleID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find article: %w", result.Error)
	}

	// Store old title to check if it changed
	oldTitle := article.DraftTitle

	// Process tags using tag service
	tagIDs, err := s.tagService.EnsureTagsExist(req.Tags)
	if err != nil {
		return nil, err
	}

	// Update draft fields
	article.DraftTitle = req.Title
	article.DraftContent = req.Content
	article.DraftImageURL = req.ImageURL

	// Regenerate slug if title changed
	if oldTitle != req.Title {
		article.Slug = s.generateUniqueSlug(req.Title, &articleID)
		fmt.Printf("\n\n- Title changed from '%s' to '%s', new slug: %s\n", oldTitle, req.Title, article.Slug)
	}

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

	// Update article draft fields
	updateFields := map[string]interface{}{
		"draft_title":     article.DraftTitle,
		"slug":            article.Slug,
		"draft_content":   article.DraftContent,
		"draft_image_url": article.DraftImageURL,
		"published_at":    article.PublishedAt,
		"tag_ids":         article.TagIDs,
		"updated_at":      time.Now(),
	}

	fmt.Println("\n\n- Updating article", articleID)
	result = db.Model(&article).Updates(updateFields)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update article: %w", result.Error)
	}

	// Create version record asynchronously
	go s.createVersionAsync(articleID, req.Title, req.Content, req.ImageURL, nil, "draft", article.AuthorID)

	// Generate embeddings in the background after successful update
	go func() {
		fmt.Println("\n\n- Regenerating embedding for article", articleID)
		ctx := context.Background()
		if err := s.regenerateArticleEmbedding(ctx, articleID, req.Content); err != nil {
			// Log error but don't fail the update
			fmt.Printf("Warning: failed to regenerate embedding for article %s: %v\n", articleID, err)
		}
	}()

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

	article.DraftContent = updatedContent

	result = db.Model(&article).Update("draft_content", updatedContent)
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
		if result.Error == gorm.ErrRecordNotFound {
			return uuid.UUID{}, core.NotFoundError("Article")
		}
		return uuid.UUID{}, core.InternalError("Failed to fetch article by slug")
	}

	return article.ID, nil
}

// getAuthorData fetches author information for an article
func (s *ArticleService) getAuthorData(authorID uuid.UUID) (AuthorData, error) {
	db := s.db.GetDB()
	var account models.Account

	if err := db.First(&account, "id = ?", authorID).Error; err != nil {
		// Return empty author data if not found, rather than failing
		return AuthorData{
			ID:   authorID,
			Name: "",
		}, nil
	}

	return AuthorData{
		ID:   authorID,
		Name: account.Name,
	}, nil
}

// getTagsData fetches tag information for an article
func (s *ArticleService) getTagsData(articleID uuid.UUID, tagIDs []int64) ([]TagData, error) {
	if len(tagIDs) == 0 {
		return []TagData{}, nil
	}

	tags, err := s.tagService.GetTagsByIDs(tagIDs)
	if err != nil {
		return nil, err
	}

	var tagData []TagData
	for _, t := range tags {
		tagData = append(tagData, TagData{
			ArticleID: articleID,
			TagID:     int(t.ID),
			TagName:   t.Name,
		})
	}

	return tagData, nil
}

// enrichArticleWithMetadata adds author and tag metadata to an article
func (s *ArticleService) enrichArticleWithMetadata(article models.Article) (*ArticleListItem, error) {
	author, err := s.getAuthorData(article.AuthorID)
	if err != nil {
		return nil, err
	}

	tags, err := s.getTagsData(article.ID, article.TagIDs)
	if err != nil {
		return nil, err
	}

	return &ArticleListItem{
		Article: article,
		Author:  author,
		Tags:    tags,
	}, nil
}

func (s *ArticleService) GetArticle(id uuid.UUID) (*ArticleListItem, error) {
	db := s.db.GetDB()
	var article models.Article
	if err := db.First(&article, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Article")
		}
		return nil, core.InternalError("Failed to fetch article")
	}

	return s.enrichArticleWithMetadata(article)
}

func (s *ArticleService) GetArticles(page int, tag string, status string, articlesPerPage int, sortBy string, sortOrder string) (*ArticleListResponse, error) {
	db := s.db.GetDB()
	var articles []models.Article
	var totalCount int64

	if articlesPerPage <= 0 {
		articlesPerPage = ITEMS_PER_PAGE
	}

	query := db.Model(&models.Article{})

	switch status {
	case "published":
		query = query.Where("published_at IS NOT NULL")
	case "drafts":
		query = query.Where("published_at IS NULL")
	case "all":
		// No filter
	default:
		query = query.Where("published_at IS NOT NULL")
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

	// Apply sorting
	orderClause := buildOrderClause(sortBy, sortOrder)
	result := query.Order(orderClause).Offset(offset).Limit(articlesPerPage).Find(&articles)
	if result.Error != nil {
		return nil, result.Error
	}

	articleItems := make([]ArticleListItem, 0)
	for _, article := range articles {
		enriched, err := s.enrichArticleWithMetadata(article)
		if err != nil {
			return nil, err
		}
		articleItems = append(articleItems, *enriched)
	}

	return &ArticleListResponse{
		Articles:      articleItems,
		TotalPages:    totalPages,
		IncludeDrafts: status == "all" || status == "drafts",
	}, nil
}

func (s *ArticleService) SearchArticles(query string, page int, tag string, status string, sortBy string, sortOrder string) (*ArticleListResponse, error) {
	db := s.db.GetDB()
	var articles []models.Article
	var totalCount int64

	searchQuery := db.Model(&models.Article{}).
		Where("draft_title ILIKE ? OR draft_content ILIKE ? OR published_title ILIKE ? OR published_content ILIKE ? OR EXISTS (SELECT 1 FROM tag WHERE tag.id = ANY(tag_ids) AND tag.name ILIKE ?)",
			"%"+query+"%", "%"+query+"%", "%"+query+"%", "%"+query+"%", "%"+query+"%")

	// Apply status filter
	switch status {
	case "published":
		searchQuery = searchQuery.Where("published_at IS NOT NULL")
	case "drafts":
		searchQuery = searchQuery.Where("published_at IS NULL")
	case "all":
		// No filter
	default:
		searchQuery = searchQuery.Where("published_at IS NOT NULL")
	}

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

	// Apply sorting
	orderClause := buildOrderClause(sortBy, sortOrder)
	result := searchQuery.Order(orderClause).Offset(offset).Limit(ITEMS_PER_PAGE).Find(&articles)
	if result.Error != nil {
		return nil, result.Error
	}

	articleItems := make([]ArticleListItem, 0)
	for _, article := range articles {
		enriched, err := s.enrichArticleWithMetadata(article)
		if err != nil {
			return nil, err
		}
		articleItems = append(articleItems, *enriched)
	}

	return &ArticleListResponse{
		Articles:      articleItems,
		TotalPages:    totalPages,
		IncludeDrafts: status == "all" || status == "drafts",
	}, nil
}

func (s *ArticleService) GetPopularTags() ([]string, error) {
	db := s.db.GetDB()
	var results []struct {
		TagName string
		Count   int
	}
	sql := `
SELECT tag.name AS tag_name, COUNT(*) AS count
FROM article, unnest(tag_ids) AS tag_id
JOIN tag ON tag.id = tag_id
WHERE article.published_at IS NOT NULL
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
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Article")
		}
		return nil, core.InternalError("Failed to fetch article")
	}

	author, err := s.getAuthorData(article.AuthorID)
	if err != nil {
		return nil, err
	}

	tags, err := s.getTagsData(article.ID, article.TagIDs)
	if err != nil {
		return nil, err
	}

	return &ArticleData{
		Article: article,
		Author:  author,
		Tags:    tags,
	}, nil
}

func (s *ArticleService) GetRecommendedArticles(currentArticleID uuid.UUID) ([]RecommendedArticle, error) {
	db := s.db.GetDB()
	var articles []models.Article

	result := db.Where("id != ? AND published_at IS NOT NULL", currentArticleID).
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
		if article.PublishedImageURL != nil && *article.PublishedImageURL != "" {
			image = article.PublishedImageURL
		} else if article.DraftImageURL != "" {
			image = &article.DraftImageURL
		}

		// Use published title if available, otherwise draft
		title := article.DraftTitle
		if article.PublishedTitle != nil {
			title = *article.PublishedTitle
		}

		recommended = append(recommended, RecommendedArticle{
			ID:          article.ID,
			Title:       title,
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
	result := db.Delete(&models.Article{}, id)
	if result.Error != nil {
		return core.InternalError("Failed to delete article")
	}
	if result.RowsAffected == 0 {
		return core.NotFoundError("Article")
	}
	return nil
}

type ArticleCreateRequest struct {
	Title    string    `json:"title" validate:"required,min=3,max=200"`
	Content  string    `json:"content" validate:"required,min=10"`
	ImageURL string    `json:"image_url" validate:"omitempty,url"`
	Tags     []string  `json:"tags" validate:"max=10,dive,min=2,max=30"`
	Publish  bool      `json:"publish"`
	AuthorID uuid.UUID `json:"authorId" validate:"required"`
}

func (s *ArticleService) CreateArticle(ctx context.Context, req ArticleCreateRequest) (*ArticleListItem, error) {
	db := s.db.GetDB()

	// Process tags using tag service
	tagIDs, err := s.tagService.EnsureTagsExist(req.Tags)
	if err != nil {
		return nil, err
	}

	// Create article with draft content
	article := models.Article{
		DraftTitle:    req.Title,
		DraftContent:  req.Content,
		DraftImageURL: req.ImageURL,
		AuthorID:      req.AuthorID,
		Slug:          s.generateUniqueSlug(req.Title, nil),
	}

	// If publish is requested, also set published content
	if req.Publish {
		now := time.Now()
		article.PublishedTitle = &req.Title
		article.PublishedContent = &req.Content
		article.PublishedImageURL = &req.ImageURL
		article.PublishedAt = &now
	}

	if len(tagIDs) > 0 {
		article.TagIDs = tagIDs
	}

	result := db.Create(&article)
	if result.Error != nil {
		return nil, result.Error
	}

	// Create initial version asynchronously
	status := "draft"
	if req.Publish {
		status = "published"
	}
	go s.createVersionAsync(article.ID, req.Title, req.Content, req.ImageURL, nil, status, req.AuthorID)

	return s.GetArticle(article.ID)
}

// PublishArticle publishes the current draft
func (s *ArticleService) PublishArticle(ctx context.Context, articleID uuid.UUID) (*ArticleListItem, error) {
	db := s.db.GetDB()

	var article models.Article
	if err := db.First(&article, articleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Article")
		}
		return nil, core.InternalError("Failed to fetch article")
	}

	now := time.Now()

	// Copy draft to published
	updateFields := map[string]interface{}{
		"published_title":     article.DraftTitle,
		"published_content":   article.DraftContent,
		"published_image_url": article.DraftImageURL,
		"published_embedding": article.DraftEmbedding,
		"published_at":        now,
		"updated_at":          now,
	}

	if err := db.Model(&article).Updates(updateFields).Error; err != nil {
		return nil, fmt.Errorf("failed to publish article: %w", err)
	}

	// Create published version asynchronously
	go s.createVersionAsync(articleID, article.DraftTitle, article.DraftContent, article.DraftImageURL, article.DraftEmbedding.Slice(), "published", article.AuthorID)

	return s.GetArticle(article.ID)
}

// UnpublishArticle removes published status
func (s *ArticleService) UnpublishArticle(ctx context.Context, articleID uuid.UUID) (*ArticleListItem, error) {
	db := s.db.GetDB()

	var article models.Article
	if err := db.First(&article, articleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Article")
		}
		return nil, core.InternalError("Failed to fetch article")
	}

	if article.PublishedAt == nil {
		return nil, core.InvalidInputError("Article is not published")
	}

	updateFields := map[string]interface{}{
		"published_title":              nil,
		"published_content":            nil,
		"published_image_url":          nil,
		"published_embedding":          nil,
		"published_at":                 nil,
		"current_published_version_id": nil,
		"updated_at":                   time.Now(),
	}

	if err := db.Model(&article).Updates(updateFields).Error; err != nil {
		return nil, fmt.Errorf("failed to unpublish article: %w", err)
	}

	return s.GetArticle(article.ID)
}

// ListVersions returns all versions for an article
func (s *ArticleService) ListVersions(ctx context.Context, articleID uuid.UUID) (*ArticleVersionListResponse, error) {
	db := s.db.GetDB()

	// Verify article exists
	var article models.Article
	if err := db.First(&article, articleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Article")
		}
		return nil, core.InternalError("Failed to fetch article")
	}

	var versions []models.ArticleVersion
	if err := db.Where("article_id = ?", articleID).Order("version_number DESC").Find(&versions).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch versions: %w", err)
	}

	response := &ArticleVersionListResponse{
		Versions: make([]ArticleVersionResponse, len(versions)),
		Total:    len(versions),
	}

	for i, v := range versions {
		response.Versions[i] = ArticleVersionResponse{
			ID:            v.ID,
			ArticleID:     v.ArticleID,
			VersionNumber: v.VersionNumber,
			Status:        v.Status,
			Title:         v.Title,
			Content:       v.Content,
			ImageURL:      v.ImageURL,
			CreatedAt:     v.CreatedAt.UTC().Format(time.RFC3339),
		}
	}

	return response, nil
}

// GetVersion retrieves a specific version
func (s *ArticleService) GetVersion(ctx context.Context, versionID uuid.UUID) (*ArticleVersionResponse, error) {
	db := s.db.GetDB()

	var version models.ArticleVersion
	if err := db.First(&version, versionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Version")
		}
		return nil, core.InternalError("Failed to fetch version")
	}

	return &ArticleVersionResponse{
		ID:            version.ID,
		ArticleID:     version.ArticleID,
		VersionNumber: version.VersionNumber,
		Status:        version.Status,
		Title:         version.Title,
		Content:       version.Content,
		ImageURL:      version.ImageURL,
		CreatedAt:     version.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// RevertToVersion creates a new draft from a historical version
func (s *ArticleService) RevertToVersion(ctx context.Context, articleID, versionID uuid.UUID) (*ArticleListItem, error) {
	db := s.db.GetDB()

	// Verify article exists
	var article models.Article
	if err := db.First(&article, articleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Article")
		}
		return nil, core.InternalError("Failed to fetch article")
	}

	// Get the version to revert to
	var version models.ArticleVersion
	if err := db.First(&version, versionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Version")
		}
		return nil, core.InternalError("Failed to fetch version")
	}

	// Verify version belongs to this article
	if version.ArticleID != articleID {
		return nil, core.InvalidInputError("Version does not belong to this article")
	}

	now := time.Now()

	// Update draft with version content
	updateFields := map[string]interface{}{
		"draft_title":     version.Title,
		"draft_content":   version.Content,
		"draft_image_url": version.ImageURL,
		"draft_embedding": version.Embedding,
		"updated_at":      now,
	}

	if err := db.Model(&article).Updates(updateFields).Error; err != nil {
		return nil, fmt.Errorf("failed to revert article: %w", err)
	}

	// Create new draft version asynchronously (to record the revert)
	go s.createVersionAsync(articleID, version.Title, version.Content, version.ImageURL, version.Embedding.Slice(), "draft", article.AuthorID)

	return s.GetArticle(article.ID)
}

// createVersionAsync creates a version record asynchronously
func (s *ArticleService) createVersionAsync(articleID uuid.UUID, title, content, imageURL string, embedding []float32, status string, editedBy uuid.UUID) {
	fmt.Printf("\n[VERSION] Creating version for article %s, status=%s\n", articleID, status)
	
	db := s.db.GetDB()
	ctx := context.Background()

	// Get next version number
	var maxVersion int
	if err := db.WithContext(ctx).Model(&models.ArticleVersion{}).
		Where("article_id = ?", articleID).
		Select("COALESCE(MAX(version_number), 0)").
		Scan(&maxVersion).Error; err != nil {
		fmt.Printf("[VERSION] Error getting max version: %v\n", err)
		return
	}
	fmt.Printf("[VERSION] Max version number: %d, next will be %d\n", maxVersion, maxVersion+1)

	var editedByPtr *uuid.UUID
	if editedBy != uuid.Nil {
		editedByPtr = &editedBy
	}

	version := &models.ArticleVersion{
		ID:            uuid.New(),
		ArticleID:     articleID,
		VersionNumber: maxVersion + 1,
		Status:        status,
		Title:         title,
		Content:       content,
		ImageURL:      imageURL,
		EditedBy:      editedByPtr,
		CreatedAt:     time.Now(),
	}

	// Create version - omit embedding if empty (pgvector requires at least 1 dimension)
	var err error
	if len(embedding) > 0 {
		version.Embedding = pgvector.NewVector(embedding)
		err = db.WithContext(ctx).Create(version).Error
	} else {
		// Omit embedding field when it's empty to avoid pgvector error
		err = db.WithContext(ctx).Omit("Embedding").Create(version).Error
	}

	if err != nil {
		fmt.Printf("[VERSION] Failed to create version for article %s: %v\n", articleID, err)
		log.Printf("Failed to create version for article %s: %v", articleID, err)
		return
	}
	fmt.Printf("[VERSION] Successfully created version %s (v%d) for article %s\n", version.ID, version.VersionNumber, articleID)

	// Update version pointer on article
	pointerField := "current_draft_version_id"
	if status == "published" {
		pointerField = "current_published_version_id"
	}
	if err := db.Model(&models.Article{}).
		Where("id = ?", articleID).
		Update(pointerField, version.ID).Error; err != nil {
		fmt.Printf("[VERSION] Failed to update version pointer: %v\n", err)
	} else {
		fmt.Printf("[VERSION] Updated %s pointer to %s\n", pointerField, version.ID)
	}
}

// Helper function to generate slug from title
func generateSlug(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)

	// Replace spaces with dashes
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove special characters (keep only alphanumeric and dashes)
	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	slug = reg.ReplaceAllString(slug, "")

	// Replace multiple consecutive dashes with single dash
	multiDash := regexp.MustCompile(`-+`)
	slug = multiDash.ReplaceAllString(slug, "-")

	// Trim leading/trailing dashes
	slug = strings.Trim(slug, "-")

	// Ensure slug is not empty
	if slug == "" {
		slug = "untitled"
	}

	return slug
}

// generateUniqueSlug generates a unique slug for an article, appending a short UUID if needed
// excludeArticleID is optional - if provided, that article's slug won't cause a conflict
func (s *ArticleService) generateUniqueSlug(title string, excludeArticleID *uuid.UUID) string {
	db := s.db.GetDB()
	baseSlug := generateSlug(title)
	slug := baseSlug

	// Check if slug exists (excluding the current article if updating)
	var count int64
	query := db.Model(&models.Article{}).Where("slug = ?", slug)
	if excludeArticleID != nil {
		query = query.Where("id != ?", *excludeArticleID)
	}
	query.Count(&count)

	// If slug exists, append short UUID
	if count > 0 {
		shortUUID := uuid.New().String()[:8]
		slug = fmt.Sprintf("%s-%s", baseSlug, shortUUID)
	}

	return slug
}

// buildOrderClause constructs a safe ORDER BY clause for article queries
func buildOrderClause(sortBy string, sortOrder string) string {
	// Validate and sanitize sort order
	order := "DESC"
	if strings.ToUpper(sortOrder) == "ASC" {
		order = "ASC"
	}

	// Map sortBy to valid column names
	validColumns := map[string]string{
		"title":        "draft_title",
		"created_at":   "created_at",
		"published_at": "published_at",
		"status":       "published_at", // Sort by published status
	}

	column, exists := validColumns[sortBy]
	if !exists {
		// Default to created_at if invalid column
		column = "created_at"
	}

	return fmt.Sprintf("%s %s", column, order)
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

// regenerateArticleEmbedding generates and updates the embedding for an article
func (s *ArticleService) regenerateArticleEmbedding(ctx context.Context, articleID uuid.UUID, content string) error {
	// Generate embedding for the article content
	embedding, err := s.embeddingService.GenerateEmbedding(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Update the article with the new embedding
	db := s.db.GetDB()
	result := db.Model(&models.Article{}).Where("id = ?", articleID).Update("draft_embedding", embedding)
	if result.Error != nil {
		return fmt.Errorf("failed to update article embedding: %w", result.Error)
	}

	return nil
}
