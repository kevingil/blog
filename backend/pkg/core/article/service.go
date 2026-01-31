package article

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"backend/pkg/core"
	"backend/pkg/core/agent"
	"backend/pkg/core/ml"
	"backend/pkg/database/models"
	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
)

// ArticleListItem represents an article with author and tag data
type ArticleListItem struct {
	Article models.Article `json:"article"`
	Author  AuthorData     `json:"author"`
	Tags    []TagData      `json:"tags"`
}

// ArticleData represents full article data with metadata
type ArticleData struct {
	Article models.Article `json:"article"`
	Tags    []TagData      `json:"tags"`
	Author  AuthorData     `json:"author"`
}

// AuthorData represents author information
type AuthorData struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// TagData represents tag information for an article
type TagData struct {
	ArticleID uuid.UUID `json:"article_id"`
	TagID     int       `json:"tag_id"`
	TagName   string    `json:"name"`
}

// RecommendedArticle represents a recommended article
type RecommendedArticle struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	ImageURL    *string   `json:"image_url"`
	PublishedAt *string   `json:"published_at"`
	CreatedAt   string    `json:"created_at"`
	Author      *string   `json:"author"`
}

// ArticleListResponse represents the response for listing articles
type ArticleListResponse struct {
	Articles      []ArticleListItem `json:"articles"`
	TotalPages    int               `json:"total_pages"`
	IncludeDrafts bool              `json:"include_drafts"`
}

// ArticleVersionResponse represents a version in responses
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

// ArticleVersionListResponse represents version list response
type ArticleVersionListResponse struct {
	Versions []ArticleVersionResponse `json:"versions"`
	Total    int                      `json:"total"`
}

// CreateRequest represents a request to create an article
type CreateRequest struct {
	Title    string    `json:"title" validate:"required,min=3,max=200"`
	Content  string    `json:"content" validate:"required,min=10"`
	ImageURL string    `json:"image_url" validate:"omitempty,url"`
	Tags     []string  `json:"tags" validate:"max=10,dive,min=2,max=30"`
	Publish  bool      `json:"publish"`
	AuthorID uuid.UUID `json:"authorId" validate:"required"`
}

// UpdateRequest represents a request to update an article
type UpdateRequest struct {
	Title       string   `json:"title" validate:"required,min=3,max=200"`
	Content     string   `json:"content" validate:"required,min=10"`
	ImageURL    string   `json:"image_url" validate:"omitempty,url"`
	Tags        []string `json:"tags" validate:"max=10,dive,min=2,max=30"`
	PublishedAt *int64   `json:"published_at"`
}

const ITEMS_PER_PAGE = 6

// Service provides business logic for articles
type Service struct {
	articleStore   ArticleStore
	accountStore   AccountStore
	tagStore       TagStore
	embeddingStore *ml.EmbeddingService
}

// NewService creates a new article service with the provided stores
func NewService(articleStore ArticleStore, accountStore AccountStore, tagStore TagStore) *Service {
	return &Service{
		articleStore:   articleStore,
		accountStore:   accountStore,
		tagStore:       tagStore,
		embeddingStore: ml.NewEmbeddingService(),
	}
}

// GetByID retrieves an article by its ID with metadata
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*ArticleListItem, error) {
	article, err := s.articleStore.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.enrichArticleWithMetadata(ctx, article)
}

// GetBySlug retrieves an article by its slug with metadata
func (s *Service) GetBySlug(ctx context.Context, slug string) (*ArticleData, error) {
	article, err := s.articleStore.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	author, err := s.getAuthorData(ctx, article.AuthorID)
	if err != nil {
		return nil, err
	}

	tags, err := s.getTagsData(ctx, article.ID, article.TagIDs)
	if err != nil {
		return nil, err
	}

	return &ArticleData{
		Article: s.typeToModel(article),
		Author:  author,
		Tags:    tags,
	}, nil
}

// GetIDBySlug retrieves an article ID by its slug
func (s *Service) GetIDBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	article, err := s.articleStore.FindBySlug(ctx, slug)
	if err != nil {
		return uuid.UUID{}, err
	}
	return article.ID, nil
}

// List retrieves articles with pagination, filtering, and sorting
func (s *Service) List(ctx context.Context, page int, tagName string, status string, articlesPerPage int, sortBy string, sortOrder string) (*ArticleListResponse, error) {
	if articlesPerPage <= 0 {
		articlesPerPage = ITEMS_PER_PAGE
	}

	// Determine published filter
	publishedOnly := true
	switch status {
	case "published":
		publishedOnly = true
	case "drafts":
		publishedOnly = false
	case "all":
		publishedOnly = false
	}

	// Get tag ID if filtering by tag
	var tagID *int
	if tagName != "" {
		tag, err := s.tagStore.FindByName(ctx, tagName)
		if err == nil {
			tagID = &tag.ID
		}
	}

	opts := types.ArticleListOptions{
		Page:          page,
		PerPage:       articlesPerPage,
		PublishedOnly: publishedOnly && status != "all",
		TagID:         tagID,
	}

	articles, totalCount, err := s.articleStore.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(articlesPerPage)))

	articleItems := make([]ArticleListItem, 0)
	for _, article := range articles {
		enriched, err := s.enrichArticleWithMetadata(ctx, &article)
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

// Search performs full-text search on articles
func (s *Service) Search(ctx context.Context, query string, page int, tagName string, status string, sortBy string, sortOrder string) (*ArticleListResponse, error) {
	publishedOnly := true
	switch status {
	case "published":
		publishedOnly = true
	case "drafts":
		publishedOnly = false
	case "all":
		publishedOnly = false
	}

	opts := types.ArticleSearchOptions{
		Query:         query,
		Page:          page,
		PerPage:       ITEMS_PER_PAGE,
		PublishedOnly: publishedOnly && status != "all",
	}

	articles, totalCount, err := s.articleStore.Search(ctx, opts)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(ITEMS_PER_PAGE)))

	articleItems := make([]ArticleListItem, 0)
	for _, article := range articles {
		enriched, err := s.enrichArticleWithMetadata(ctx, &article)
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

// GetPopularTags returns popular tag names
func (s *Service) GetPopularTags(ctx context.Context) ([]string, error) {
	tagIDs, err := s.articleStore.GetPopularTags(ctx, 10)
	if err != nil {
		return nil, err
	}

	tags, err := s.tagStore.FindByIDs(ctx, tagIDs)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Name
	}
	return names, nil
}

// GetRecommended retrieves recommended articles excluding a specific article
func (s *Service) GetRecommended(ctx context.Context, currentArticleID uuid.UUID) ([]RecommendedArticle, error) {
	opts := types.ArticleListOptions{
		Page:          1,
		PerPage:       4,
		PublishedOnly: true,
	}

	articles, _, err := s.articleStore.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	var recommended []RecommendedArticle
	for _, article := range articles {
		if article.ID == currentArticleID {
			continue
		}
		if len(recommended) >= 3 {
			break
		}

		var authorName *string
		account, err := s.accountStore.FindByID(ctx, article.AuthorID)
		if err == nil {
			authorName = &account.Name
		}

		var image *string
		if article.PublishedImageURL != nil && *article.PublishedImageURL != "" {
			image = article.PublishedImageURL
		} else if article.DraftImageURL != "" {
			image = &article.DraftImageURL
		}

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

// GenerateArticle uses AI to generate an article
func (s *Service) GenerateArticle(ctx context.Context, prompt string, title string, authorID uuid.UUID, publish bool) (*models.Article, error) {
	writerAgent := agent.NewWriterAgent()

	article, err := writerAgent.GenerateArticle(ctx, prompt, title, authorID)
	if err != nil {
		return nil, fmt.Errorf("error generating article: %w", err)
	}

	slug, err := s.generateUniqueSlug(ctx, title, nil)
	if err != nil {
		return nil, err
	}

	// Convert pgvector.Vector to []float32
	var draftEmbedding []float32
	if article.DraftEmbedding.Slice() != nil {
		draftEmbedding = article.DraftEmbedding.Slice()
	}

	// Convert datatypes.JSON to map[string]interface{}
	var sessionMemory map[string]interface{}
	if article.SessionMemory != nil {
		_ = json.Unmarshal(article.SessionMemory, &sessionMemory)
	}

	newArticle := &types.Article{
		ID:              uuid.New(),
		DraftImageURL:   article.DraftImageURL,
		Slug:            slug,
		DraftTitle:      article.DraftTitle,
		DraftContent:    article.DraftContent,
		AuthorID:        authorID,
		DraftEmbedding:  draftEmbedding,
		ImagenRequestID: article.ImagenRequestID,
		SessionMemory:   sessionMemory,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if publish {
		now := time.Now()
		newArticle.PublishedTitle = &article.DraftTitle
		newArticle.PublishedContent = &article.DraftContent
		newArticle.PublishedImageURL = &article.DraftImageURL
		newArticle.PublishedEmbedding = newArticle.DraftEmbedding
		newArticle.PublishedAt = &now
	}

	if err := s.articleStore.Save(ctx, newArticle); err != nil {
		return nil, err
	}

	return s.typeToModelPtr(newArticle), nil
}

// Create creates a new article
func (s *Service) Create(ctx context.Context, req CreateRequest) (*ArticleListItem, error) {
	tagIDs, err := s.tagStore.EnsureExists(ctx, req.Tags)
	if err != nil {
		return nil, err
	}

	slug, err := s.generateUniqueSlug(ctx, req.Title, nil)
	if err != nil {
		return nil, err
	}

	article := &types.Article{
		ID:            uuid.New(),
		DraftTitle:    req.Title,
		DraftContent:  req.Content,
		DraftImageURL: req.ImageURL,
		AuthorID:      req.AuthorID,
		Slug:          slug,
		TagIDs:        tagIDs,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if req.Publish {
		now := time.Now()
		article.PublishedTitle = &req.Title
		article.PublishedContent = &req.Content
		article.PublishedImageURL = &req.ImageURL
		article.PublishedAt = &now
	}

	if err := s.articleStore.Save(ctx, article); err != nil {
		return nil, err
	}

	return s.GetByID(ctx, article.ID)
}

// Update updates an existing article
func (s *Service) Update(ctx context.Context, articleID uuid.UUID, req UpdateRequest) (*ArticleListItem, error) {
	article, err := s.articleStore.FindByID(ctx, articleID)
	if err != nil {
		return nil, err
	}

	oldTitle := article.DraftTitle

	tagIDs, err := s.tagStore.EnsureExists(ctx, req.Tags)
	if err != nil {
		return nil, err
	}

	article.DraftTitle = req.Title
	article.DraftContent = req.Content
	article.DraftImageURL = req.ImageURL
	article.TagIDs = tagIDs
	article.UpdatedAt = time.Now()

	if oldTitle != req.Title {
		slug, err := s.generateUniqueSlug(ctx, req.Title, &articleID)
		if err != nil {
			return nil, err
		}
		article.Slug = slug
	}

	if req.PublishedAt != nil {
		var t time.Time
		if *req.PublishedAt > 1e12 {
			t = time.Unix(0, *req.PublishedAt*int64(time.Millisecond))
		} else {
			t = time.Unix(*req.PublishedAt, 0)
		}
		article.PublishedAt = &t
	}

	if err := s.articleStore.Save(ctx, article); err != nil {
		return nil, err
	}

	// Regenerate embedding asynchronously
	go func() {
		ctx := context.Background()
		if err := s.regenerateArticleEmbedding(ctx, articleID, req.Content); err != nil {
			fmt.Printf("Warning: failed to regenerate embedding for article %s: %v\n", articleID, err)
		}
	}()

	return s.GetByID(ctx, article.ID)
}

// UpdateWithContext updates article content using AI context
func (s *Service) UpdateWithContext(ctx context.Context, articleID uuid.UUID) (*models.Article, error) {
	writerAgent := agent.NewWriterAgent()

	article, err := s.articleStore.FindByID(ctx, articleID)
	if err != nil {
		return nil, err
	}

	articleModel := s.typeToModelPtr(article)
	updatedContent, err := writerAgent.UpdateWithContext(ctx, articleModel)
	if err != nil {
		return nil, fmt.Errorf("error updating article content: %w", err)
	}

	article.DraftContent = updatedContent
	article.UpdatedAt = time.Now()

	if err := s.articleStore.Save(ctx, article); err != nil {
		return nil, err
	}

	return s.typeToModelPtr(article), nil
}

// Delete removes an article by its ID
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.articleStore.Delete(ctx, id)
}

// Publish publishes the current draft
func (s *Service) Publish(ctx context.Context, articleID uuid.UUID) (*ArticleListItem, error) {
	article, err := s.articleStore.FindByID(ctx, articleID)
	if err != nil {
		return nil, err
	}

	if err := s.articleStore.Publish(ctx, article); err != nil {
		return nil, fmt.Errorf("failed to publish article: %w", err)
	}

	return s.GetByID(ctx, article.ID)
}

// Unpublish removes published status
func (s *Service) Unpublish(ctx context.Context, articleID uuid.UUID) (*ArticleListItem, error) {
	article, err := s.articleStore.FindByID(ctx, articleID)
	if err != nil {
		return nil, err
	}

	if article.PublishedAt == nil {
		return nil, core.ErrValidation
	}

	if err := s.articleStore.Unpublish(ctx, article); err != nil {
		return nil, fmt.Errorf("failed to unpublish article: %w", err)
	}

	return s.GetByID(ctx, article.ID)
}

// ListVersions returns all versions for an article
func (s *Service) ListVersions(ctx context.Context, articleID uuid.UUID) (*ArticleVersionListResponse, error) {
	// Verify article exists
	_, err := s.articleStore.FindByID(ctx, articleID)
	if err != nil {
		return nil, err
	}

	versions, err := s.articleStore.ListVersions(ctx, articleID)
	if err != nil {
		return nil, err
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
func (s *Service) GetVersion(ctx context.Context, versionID uuid.UUID) (*ArticleVersionResponse, error) {
	version, err := s.articleStore.GetVersion(ctx, versionID)
	if err != nil {
		return nil, err
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
func (s *Service) RevertToVersion(ctx context.Context, articleID, versionID uuid.UUID) (*ArticleListItem, error) {
	// Verify article exists
	_, err := s.articleStore.FindByID(ctx, articleID)
	if err != nil {
		return nil, err
	}

	if err := s.articleStore.RevertToVersion(ctx, articleID, versionID); err != nil {
		return nil, err
	}

	return s.GetByID(ctx, articleID)
}

// Helper methods

func (s *Service) getAuthorData(ctx context.Context, authorID uuid.UUID) (AuthorData, error) {
	account, err := s.accountStore.FindByID(ctx, authorID)
	if err != nil {
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

func (s *Service) getTagsData(ctx context.Context, articleID uuid.UUID, tagIDs []int64) ([]TagData, error) {
	if len(tagIDs) == 0 {
		return []TagData{}, nil
	}

	tags, err := s.tagStore.FindByIDs(ctx, tagIDs)
	if err != nil {
		return nil, err
	}

	var tagData []TagData
	for _, t := range tags {
		tagData = append(tagData, TagData{
			ArticleID: articleID,
			TagID:     t.ID,
			TagName:   t.Name,
		})
	}

	return tagData, nil
}

func (s *Service) enrichArticleWithMetadata(ctx context.Context, article *types.Article) (*ArticleListItem, error) {
	author, err := s.getAuthorData(ctx, article.AuthorID)
	if err != nil {
		return nil, err
	}

	tags, err := s.getTagsData(ctx, article.ID, article.TagIDs)
	if err != nil {
		return nil, err
	}

	return &ArticleListItem{
		Article: s.typeToModel(article),
		Author:  author,
		Tags:    tags,
	}, nil
}

func (s *Service) generateUniqueSlug(ctx context.Context, title string, excludeArticleID *uuid.UUID) (string, error) {
	baseSlug := generateSlug(title)
	slug := baseSlug

	exists, err := s.articleStore.SlugExists(ctx, slug, excludeArticleID)
	if err != nil {
		return "", err
	}

	if exists {
		shortUUID := uuid.New().String()[:8]
		slug = fmt.Sprintf("%s-%s", baseSlug, shortUUID)
	}

	return slug, nil
}

func (s *Service) regenerateArticleEmbedding(ctx context.Context, articleID uuid.UUID, content string) error {
	embeddingVector, err := s.embeddingStore.GenerateEmbedding(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	article, err := s.articleStore.FindByID(ctx, articleID)
	if err != nil {
		return err
	}

	// Convert pgvector.Vector to []float32
	article.DraftEmbedding = embeddingVector.Slice()
	return s.articleStore.Save(ctx, article)
}

// typeToModel converts types.Article to models.Article
func (s *Service) typeToModel(a *types.Article) models.Article {
	var draftEmbedding pgvector.Vector
	if len(a.DraftEmbedding) > 0 {
		draftEmbedding = pgvector.NewVector(a.DraftEmbedding)
	}

	var publishedEmbedding pgvector.Vector
	if len(a.PublishedEmbedding) > 0 {
		publishedEmbedding = pgvector.NewVector(a.PublishedEmbedding)
	}

	var sessionMemory datatypes.JSON
	if a.SessionMemory != nil {
		data, _ := json.Marshal(a.SessionMemory)
		sessionMemory = datatypes.JSON(data)
	}

	return models.Article{
		ID:                        a.ID,
		Slug:                      a.Slug,
		AuthorID:                  a.AuthorID,
		TagIDs:                    pq.Int64Array(a.TagIDs),
		DraftTitle:                a.DraftTitle,
		DraftContent:              a.DraftContent,
		DraftImageURL:             a.DraftImageURL,
		DraftEmbedding:            draftEmbedding,
		PublishedTitle:            a.PublishedTitle,
		PublishedContent:          a.PublishedContent,
		PublishedImageURL:         a.PublishedImageURL,
		PublishedEmbedding:        publishedEmbedding,
		PublishedAt:               a.PublishedAt,
		CurrentDraftVersionID:     a.CurrentDraftVersionID,
		CurrentPublishedVersionID: a.CurrentPublishedVersionID,
		ImagenRequestID:           a.ImagenRequestID,
		SessionMemory:             sessionMemory,
		CreatedAt:                 a.CreatedAt,
		UpdatedAt:                 a.UpdatedAt,
	}
}

// typeToModelPtr converts types.Article to *models.Article
func (s *Service) typeToModelPtr(a *types.Article) *models.Article {
	m := s.typeToModel(a)
	return &m
}

// Utility functions (stateless)

func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	slug = reg.ReplaceAllString(slug, "")
	multiDash := regexp.MustCompile(`-+`)
	slug = multiDash.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "untitled"
	}
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
	str := t.UTC().Format(time.RFC3339)
	return &str
}
