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
	openai "github.com/openai/openai-go"
	"github.com/pgvector/pgvector-go"
)

type ArticleService struct {
	db          database.Service
	writerAgent *WriterAgent
	tagService  *TagService
}

func NewArticleService(db database.Service, writerAgent *WriterAgent) *ArticleService {
	return &ArticleService{
		db:          db,
		writerAgent: writerAgent,
		tagService:  NewTagService(db),
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
	ImageURL    string   `json:"image_url"`
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

	// Process tags using tag service
	tagIDs, err := s.tagService.EnsureTagsExist(req.Tags)
	if err != nil {
		return nil, err
	}

	// Update article fields directly
	article.Title = req.Title
	article.Content = req.Content
	article.ImageURL = req.ImageURL
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

	// Update article without embedding field to avoid "vector must have at least 1 dimension" error
	updateFields := map[string]interface{}{
		"title":        article.Title,
		"content":      article.Content,
		"image_url":    article.ImageURL,
		"is_draft":     article.IsDraft,
		"published_at": article.PublishedAt,
		"tag_ids":      article.TagIDs,
		"updated_at":   time.Now(),
	}

	fmt.Println("\n\n- Updating article", articleID)
	result = db.Model(&article).Updates(updateFields)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update article: %w", result.Error)
	}

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
		return nil, err
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
		Where("title ILIKE ? OR content ILIKE ? OR EXISTS (SELECT 1 FROM tag WHERE tag.id = ANY(tag_ids) AND tag.name ILIKE ?)", "%"+query+"%", "%"+query+"%", "%"+query+"%")

	// Apply status filter
	switch status {
	case "published":
		searchQuery = searchQuery.Where("is_draft = ?", false)
	case "drafts":
		searchQuery = searchQuery.Where("is_draft = ?", true)
	case "all":
		// No filter
	default:
		searchQuery = searchQuery.Where("is_draft = ?", false)
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
	ImageURL string    `json:"image_url"`
	Tags     []string  `json:"tags"`
	IsDraft  bool      `json:"isDraft"`
	AuthorID uuid.UUID `json:"authorId"`
}

func (s *ArticleService) CreateArticle(ctx context.Context, req ArticleCreateRequest) (*ArticleListItem, error) {
	db := s.db.GetDB()

	// Process tags using tag service
	tagIDs, err := s.tagService.EnsureTagsExist(req.Tags)
	if err != nil {
		return nil, err
	}

	// Create article
	article := models.Article{
		Title:    req.Title,
		Content:  req.Content,
		ImageURL: req.ImageURL,
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

// buildOrderClause constructs a safe ORDER BY clause for article queries
func buildOrderClause(sortBy string, sortOrder string) string {
	// Validate and sanitize sort order
	order := "DESC"
	if strings.ToUpper(sortOrder) == "ASC" {
		order = "ASC"
	}

	// Map sortBy to valid column names
	validColumns := map[string]string{
		"title":        "title",
		"created_at":   "created_at",
		"published_at": "published_at",
		"is_draft":     "is_draft",
		"status":       "is_draft", // Map status to is_draft
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

// generateEmbedding generates an embedding vector for the given text using OpenAI's API
func (s *ArticleService) generateEmbedding(ctx context.Context, text string) (pgvector.Vector, error) {
	if text == "" {
		return pgvector.Vector{}, fmt.Errorf("text cannot be empty")
	}

	// Truncate text if too long (OpenAI has token limits)
	// text-embedding-3-small supports up to 8192 tokens (~6000 characters)
	originalLength := len(text)
	if len(text) > 8000 {
		text = text[:8000]
	}

	// Create OpenAI client
	client := openai.NewClient()

	// Generate embedding using OpenAI's text-embedding-3-small model
	// This model produces 1536-dimensional embeddings
	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: []string{text},
		},
		Model: openai.EmbeddingModelTextEmbedding3Small,
		// Optionally set dimensions to 1536 explicitly (default for text-embedding-3-small)
		// Dimensions: param.Int(1536),
	})
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("failed to generate embedding from OpenAI (text length: %d): %w", originalLength, err)
	}

	if len(resp.Data) == 0 {
		return pgvector.Vector{}, fmt.Errorf("no embedding data returned from OpenAI")
	}

	// Validate embedding dimensions
	embeddingData := resp.Data[0].Embedding
	if len(embeddingData) != 1536 {
		return pgvector.Vector{}, fmt.Errorf("unexpected embedding dimensions: got %d, expected 1536", len(embeddingData))
	}

	// Convert []float64 to []float32 for pgvector compatibility
	embedding := make([]float32, len(embeddingData))
	for i, v := range embeddingData {
		embedding[i] = float32(v)
	}

	return pgvector.NewVector(embedding), nil
}

// regenerateArticleEmbedding generates and updates the embedding for an article
func (s *ArticleService) regenerateArticleEmbedding(ctx context.Context, articleID uuid.UUID, content string) error {
	// Generate embedding for the article content
	embedding, err := s.generateEmbedding(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Update the article with the new embedding
	db := s.db.GetDB()
	result := db.Model(&models.Article{}).Where("id = ?", articleID).Update("embedding", embedding)
	if result.Error != nil {
		return fmt.Errorf("failed to update article embedding: %w", result.Error)
	}

	return nil
}
