package article

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"backend/pkg/core"
	"backend/pkg/core/agent"
	"backend/pkg/core/ml"
	"backend/pkg/core/tag"
	"backend/pkg/database"
	"backend/pkg/database/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
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

// inlineTagStore implements tag.TagStore using database.DB() directly
type inlineTagStore struct{}

func (s *inlineTagStore) FindByID(ctx context.Context, id int) (*tag.Tag, error) {
	db := database.DB()
	var model models.Tag
	if err := db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return &tag.Tag{ID: model.ID, Name: model.Name, CreatedAt: model.CreatedAt}, nil
}

func (s *inlineTagStore) FindByName(ctx context.Context, name string) (*tag.Tag, error) {
	db := database.DB()
	var model models.Tag
	if err := db.WithContext(ctx).Where("name = ?", name).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return &tag.Tag{ID: model.ID, Name: model.Name, CreatedAt: model.CreatedAt}, nil
}

func (s *inlineTagStore) FindByIDs(ctx context.Context, ids []int64) ([]tag.Tag, error) {
	db := database.DB()
	var tagModels []models.Tag
	if err := db.WithContext(ctx).Where("id IN ?", ids).Find(&tagModels).Error; err != nil {
		return nil, err
	}
	tags := make([]tag.Tag, len(tagModels))
	for i, m := range tagModels {
		tags[i] = tag.Tag{ID: m.ID, Name: m.Name, CreatedAt: m.CreatedAt}
	}
	return tags, nil
}

func (s *inlineTagStore) EnsureExists(ctx context.Context, names []string) ([]int64, error) {
	db := database.DB()
	var ids []int64
	for _, name := range names {
		var existing models.Tag
		err := db.WithContext(ctx).Where("name = ?", name).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			newTag := models.Tag{Name: name}
			if err := db.WithContext(ctx).Create(&newTag).Error; err != nil {
				return nil, err
			}
			ids = append(ids, int64(newTag.ID))
		} else if err != nil {
			return nil, err
		} else {
			ids = append(ids, int64(existing.ID))
		}
	}
	return ids, nil
}

func (s *inlineTagStore) List(ctx context.Context) ([]tag.Tag, error) {
	db := database.DB()
	var tagModels []models.Tag
	if err := db.WithContext(ctx).Order("name ASC").Find(&tagModels).Error; err != nil {
		return nil, err
	}
	tags := make([]tag.Tag, len(tagModels))
	for i, m := range tagModels {
		tags[i] = tag.Tag{ID: m.ID, Name: m.Name, CreatedAt: m.CreatedAt}
	}
	return tags, nil
}

func (s *inlineTagStore) Save(ctx context.Context, t *tag.Tag) error {
	db := database.DB()
	model := models.Tag{ID: t.ID, Name: t.Name, CreatedAt: t.CreatedAt}
	if err := db.WithContext(ctx).Save(&model).Error; err != nil {
		return err
	}
	t.ID = model.ID
	return nil
}

func (s *inlineTagStore) Delete(ctx context.Context, id int) error {
	db := database.DB()
	return db.WithContext(ctx).Delete(&models.Tag{}, id).Error
}

// getTagService returns a tag service instance
func getTagService() *tag.Service {
	return tag.NewService(&inlineTagStore{})
}

// getEmbeddingService returns an embedding service instance
func getEmbeddingService() *ml.EmbeddingService {
	return ml.NewEmbeddingService()
}

// GetByID retrieves an article by its ID with metadata
func GetByID(ctx context.Context, id uuid.UUID) (*ArticleListItem, error) {
	db := database.DB()
	var article models.Article
	if err := db.WithContext(ctx).First(&article, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return enrichArticleWithMetadata(ctx, article)
}

// GetBySlug retrieves an article by its slug with metadata
func GetBySlug(ctx context.Context, slug string) (*ArticleData, error) {
	db := database.DB()
	var article models.Article
	if err := db.WithContext(ctx).Where("slug = ?", slug).First(&article).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	author, err := getAuthorData(ctx, article.AuthorID)
	if err != nil {
		return nil, err
	}

	tags, err := getTagsData(ctx, article.ID, article.TagIDs)
	if err != nil {
		return nil, err
	}

	return &ArticleData{
		Article: article,
		Author:  author,
		Tags:    tags,
	}, nil
}

// GetIDBySlug retrieves an article ID by its slug
func GetIDBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	db := database.DB()
	var article models.Article
	if err := db.WithContext(ctx).Select("id").Where("slug = ?", slug).First(&article).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return uuid.UUID{}, core.ErrNotFound
		}
		return uuid.UUID{}, err
	}
	return article.ID, nil
}

// List retrieves articles with pagination, filtering, and sorting
func List(ctx context.Context, page int, tagName string, status string, articlesPerPage int, sortBy string, sortOrder string) (*ArticleListResponse, error) {
	db := database.DB()
	var articles []models.Article
	var totalCount int64

	if articlesPerPage <= 0 {
		articlesPerPage = ITEMS_PER_PAGE
	}

	query := db.WithContext(ctx).Model(&models.Article{})

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

	if tagName != "" {
		var tagModel models.Tag
		if err := db.WithContext(ctx).Where("LOWER(name) = ?", strings.ToLower(tagName)).First(&tagModel).Error; err == nil {
			query = query.Where("tag_ids @> ARRAY[?]::integer[]", tagModel.ID)
		}
	}

	countQuery := query
	countQuery.Count(&totalCount)

	offset := (page - 1) * articlesPerPage
	totalPages := int(math.Ceil(float64(totalCount) / float64(articlesPerPage)))

	orderClause := buildOrderClause(sortBy, sortOrder)
	if err := query.Order(orderClause).Offset(offset).Limit(articlesPerPage).Find(&articles).Error; err != nil {
		return nil, err
	}

	articleItems := make([]ArticleListItem, 0)
	for _, article := range articles {
		enriched, err := enrichArticleWithMetadata(ctx, article)
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
func Search(ctx context.Context, query string, page int, tagName string, status string, sortBy string, sortOrder string) (*ArticleListResponse, error) {
	db := database.DB()
	var articles []models.Article
	var totalCount int64

	searchQuery := db.WithContext(ctx).Model(&models.Article{}).
		Where("draft_title ILIKE ? OR draft_content ILIKE ? OR published_title ILIKE ? OR published_content ILIKE ? OR EXISTS (SELECT 1 FROM tag WHERE tag.id = ANY(tag_ids) AND tag.name ILIKE ?)",
			"%"+query+"%", "%"+query+"%", "%"+query+"%", "%"+query+"%", "%"+query+"%")

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

	if tagName != "" {
		var tagModel models.Tag
		if err := db.WithContext(ctx).Where("LOWER(name) = ?", strings.ToLower(tagName)).First(&tagModel).Error; err == nil {
			searchQuery = searchQuery.Where("tag_ids @> ARRAY[?]::integer[]", tagModel.ID)
		}
	}

	countQuery := searchQuery
	countQuery.Count(&totalCount)

	offset := (page - 1) * ITEMS_PER_PAGE
	totalPages := int(math.Ceil(float64(totalCount) / float64(ITEMS_PER_PAGE)))

	orderClause := buildOrderClause(sortBy, sortOrder)
	if err := searchQuery.Order(orderClause).Offset(offset).Limit(ITEMS_PER_PAGE).Find(&articles).Error; err != nil {
		return nil, err
	}

	articleItems := make([]ArticleListItem, 0)
	for _, article := range articles {
		enriched, err := enrichArticleWithMetadata(ctx, article)
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
func GetPopularTags(ctx context.Context) ([]string, error) {
	db := database.DB()
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
	if err := db.WithContext(ctx).Raw(sql).Scan(&results).Error; err != nil {
		return nil, err
	}

	var tags []string
	for _, result := range results {
		tags = append(tags, result.TagName)
	}
	return tags, nil
}

// GetRecommended retrieves recommended articles excluding a specific article
func GetRecommended(ctx context.Context, currentArticleID uuid.UUID) ([]RecommendedArticle, error) {
	db := database.DB()
	var articles []models.Article

	if err := db.WithContext(ctx).Where("id != ? AND published_at IS NOT NULL", currentArticleID).
		Order("created_at DESC").
		Limit(3).
		Find(&articles).Error; err != nil {
		return nil, err
	}

	var recommended []RecommendedArticle
	for _, article := range articles {
		var authorName *string
		var account models.Account
		if err := db.WithContext(ctx).First(&account, "id = ?", article.AuthorID).Error; err == nil {
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
func GenerateArticle(ctx context.Context, prompt string, title string, authorID uuid.UUID, publish bool) (*models.Article, error) {
	writerAgent := agent.NewWriterAgent()

	article, err := writerAgent.GenerateArticle(ctx, prompt, title, authorID)
	if err != nil {
		return nil, fmt.Errorf("error generating article: %w", err)
	}

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

	if publish {
		now := time.Now()
		gormArticle.PublishedTitle = &article.DraftTitle
		gormArticle.PublishedContent = &article.DraftContent
		gormArticle.PublishedImageURL = &article.DraftImageURL
		gormArticle.PublishedEmbedding = gormArticle.DraftEmbedding
		gormArticle.PublishedAt = &now
	}

	db := database.DB()
	if err := db.WithContext(ctx).Create(gormArticle).Error; err != nil {
		return nil, err
	}

	// Create initial version asynchronously
	status := "draft"
	if publish {
		status = "published"
	}
	go createVersionAsync(gormArticle.ID, article.DraftTitle, article.DraftContent, article.DraftImageURL, article.DraftEmbedding.Slice(), status, authorID)

	return gormArticle, nil
}

// Create creates a new article
func Create(ctx context.Context, req CreateRequest) (*ArticleListItem, error) {
	db := database.DB()
	tagService := getTagService()

	tagIDs, err := tagService.EnsureExists(ctx, req.Tags)
	if err != nil {
		return nil, err
	}

	article := models.Article{
		DraftTitle:    req.Title,
		DraftContent:  req.Content,
		DraftImageURL: req.ImageURL,
		AuthorID:      req.AuthorID,
		Slug:          generateUniqueSlug(ctx, req.Title, nil),
	}

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

	if err := db.WithContext(ctx).Create(&article).Error; err != nil {
		return nil, err
	}

	status := "draft"
	if req.Publish {
		status = "published"
	}
	go createVersionAsync(article.ID, req.Title, req.Content, req.ImageURL, nil, status, req.AuthorID)

	return GetByID(ctx, article.ID)
}

// Update updates an existing article
func Update(ctx context.Context, articleID uuid.UUID, req UpdateRequest) (*ArticleListItem, error) {
	db := database.DB()
	tagService := getTagService()

	var article models.Article
	if err := db.WithContext(ctx).First(&article, articleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	oldTitle := article.DraftTitle

	tagIDs, err := tagService.EnsureExists(ctx, req.Tags)
	if err != nil {
		return nil, err
	}

	article.DraftTitle = req.Title
	article.DraftContent = req.Content
	article.DraftImageURL = req.ImageURL

	if oldTitle != req.Title {
		article.Slug = generateUniqueSlug(ctx, req.Title, &articleID)
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
	if len(tagIDs) > 0 {
		article.TagIDs = pq.Int64Array(tagIDs)
	}

	updateFields := map[string]interface{}{
		"draft_title":     article.DraftTitle,
		"slug":            article.Slug,
		"draft_content":   article.DraftContent,
		"draft_image_url": article.DraftImageURL,
		"published_at":    article.PublishedAt,
		"tag_ids":         article.TagIDs,
		"updated_at":      time.Now(),
	}

	if err := db.WithContext(ctx).Model(&article).Updates(updateFields).Error; err != nil {
		return nil, err
	}

	go createVersionAsync(articleID, req.Title, req.Content, req.ImageURL, nil, "draft", article.AuthorID)

	go func() {
		ctx := context.Background()
		if err := regenerateArticleEmbedding(ctx, articleID, req.Content); err != nil {
			fmt.Printf("Warning: failed to regenerate embedding for article %s: %v\n", articleID, err)
		}
	}()

	return GetByID(ctx, article.ID)
}

// UpdateWithContext updates article content using AI context
func UpdateWithContext(ctx context.Context, articleID uuid.UUID) (*models.Article, error) {
	db := database.DB()
	writerAgent := agent.NewWriterAgent()

	var article models.Article
	if err := db.WithContext(ctx).First(&article, articleID).Error; err != nil {
		return nil, err
	}

	updatedContent, err := writerAgent.UpdateWithContext(ctx, &article)
	if err != nil {
		return nil, fmt.Errorf("error updating article content: %w", err)
	}

	article.DraftContent = updatedContent

	if err := db.WithContext(ctx).Model(&article).Update("draft_content", updatedContent).Error; err != nil {
		return nil, err
	}

	return &article, nil
}

// Delete removes an article by its ID
func Delete(ctx context.Context, id uuid.UUID) error {
	db := database.DB()
	result := db.WithContext(ctx).Delete(&models.Article{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// Publish publishes the current draft
func Publish(ctx context.Context, articleID uuid.UUID) (*ArticleListItem, error) {
	db := database.DB()

	var article models.Article
	if err := db.WithContext(ctx).First(&article, articleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	now := time.Now()

	updateFields := map[string]interface{}{
		"published_title":     article.DraftTitle,
		"published_content":   article.DraftContent,
		"published_image_url": article.DraftImageURL,
		"published_embedding": article.DraftEmbedding,
		"published_at":        now,
		"updated_at":          now,
	}

	if err := db.WithContext(ctx).Model(&article).Updates(updateFields).Error; err != nil {
		return nil, fmt.Errorf("failed to publish article: %w", err)
	}

	go createVersionAsync(articleID, article.DraftTitle, article.DraftContent, article.DraftImageURL, article.DraftEmbedding.Slice(), "published", article.AuthorID)

	return GetByID(ctx, article.ID)
}

// Unpublish removes published status
func Unpublish(ctx context.Context, articleID uuid.UUID) (*ArticleListItem, error) {
	db := database.DB()

	var article models.Article
	if err := db.WithContext(ctx).First(&article, articleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	if article.PublishedAt == nil {
		return nil, core.ErrValidation
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

	if err := db.WithContext(ctx).Model(&article).Updates(updateFields).Error; err != nil {
		return nil, fmt.Errorf("failed to unpublish article: %w", err)
	}

	return GetByID(ctx, article.ID)
}

// ListVersions returns all versions for an article
func ListVersions(ctx context.Context, articleID uuid.UUID) (*ArticleVersionListResponse, error) {
	db := database.DB()

	var article models.Article
	if err := db.WithContext(ctx).First(&article, articleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	var versions []models.ArticleVersion
	if err := db.WithContext(ctx).Where("article_id = ?", articleID).Order("version_number DESC").Find(&versions).Error; err != nil {
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
func GetVersion(ctx context.Context, versionID uuid.UUID) (*ArticleVersionResponse, error) {
	db := database.DB()

	var version models.ArticleVersion
	if err := db.WithContext(ctx).First(&version, versionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
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
func RevertToVersion(ctx context.Context, articleID, versionID uuid.UUID) (*ArticleListItem, error) {
	db := database.DB()

	var article models.Article
	if err := db.WithContext(ctx).First(&article, articleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	var version models.ArticleVersion
	if err := db.WithContext(ctx).First(&version, versionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	if version.ArticleID != articleID {
		return nil, core.ErrValidation
	}

	now := time.Now()

	updateFields := map[string]interface{}{
		"draft_title":     version.Title,
		"draft_content":   version.Content,
		"draft_image_url": version.ImageURL,
		"draft_embedding": version.Embedding,
		"updated_at":      now,
	}

	if err := db.WithContext(ctx).Model(&article).Updates(updateFields).Error; err != nil {
		return nil, fmt.Errorf("failed to revert article: %w", err)
	}

	go createVersionAsync(articleID, version.Title, version.Content, version.ImageURL, version.Embedding.Slice(), "draft", article.AuthorID)

	return GetByID(ctx, article.ID)
}

// Helper functions

func getAuthorData(ctx context.Context, authorID uuid.UUID) (AuthorData, error) {
	db := database.DB()
	var account models.Account

	if err := db.WithContext(ctx).First(&account, "id = ?", authorID).Error; err != nil {
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

func getTagsData(ctx context.Context, articleID uuid.UUID, tagIDs []int64) ([]TagData, error) {
	if len(tagIDs) == 0 {
		return []TagData{}, nil
	}

	tagService := getTagService()
	tags, err := tagService.GetByIDs(ctx, tagIDs)
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

func enrichArticleWithMetadata(ctx context.Context, article models.Article) (*ArticleListItem, error) {
	author, err := getAuthorData(ctx, article.AuthorID)
	if err != nil {
		return nil, err
	}

	tags, err := getTagsData(ctx, article.ID, article.TagIDs)
	if err != nil {
		return nil, err
	}

	return &ArticleListItem{
		Article: article,
		Author:  author,
		Tags:    tags,
	}, nil
}

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

func generateUniqueSlug(ctx context.Context, title string, excludeArticleID *uuid.UUID) string {
	db := database.DB()
	baseSlug := generateSlug(title)
	slug := baseSlug

	var count int64
	query := db.WithContext(ctx).Model(&models.Article{}).Where("slug = ?", slug)
	if excludeArticleID != nil {
		query = query.Where("id != ?", *excludeArticleID)
	}
	query.Count(&count)

	if count > 0 {
		shortUUID := uuid.New().String()[:8]
		slug = fmt.Sprintf("%s-%s", baseSlug, shortUUID)
	}

	return slug
}

func buildOrderClause(sortBy string, sortOrder string) string {
	order := "DESC"
	if strings.ToUpper(sortOrder) == "ASC" {
		order = "ASC"
	}

	validColumns := map[string]string{
		"title":        "draft_title",
		"created_at":   "created_at",
		"published_at": "published_at",
		"status":       "published_at",
	}

	column, exists := validColumns[sortBy]
	if !exists {
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

func createVersionAsync(articleID uuid.UUID, title, content, imageURL string, embedding []float32, status string, editedBy uuid.UUID) {
	db := database.DB()
	ctx := context.Background()

	var maxVersion int
	if err := db.WithContext(ctx).Model(&models.ArticleVersion{}).
		Where("article_id = ?", articleID).
		Select("COALESCE(MAX(version_number), 0)").
		Scan(&maxVersion).Error; err != nil {
		fmt.Printf("[VERSION] Error getting max version: %v\n", err)
		return
	}

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

	var err error
	if len(embedding) > 0 {
		version.Embedding = pgvector.NewVector(embedding)
		err = db.WithContext(ctx).Create(version).Error
	} else {
		err = db.WithContext(ctx).Omit("Embedding").Create(version).Error
	}

	if err != nil {
		fmt.Printf("[VERSION] Failed to create version for article %s: %v\n", articleID, err)
		return
	}

	pointerField := "current_draft_version_id"
	if status == "published" {
		pointerField = "current_published_version_id"
	}
	db.Model(&models.Article{}).
		Where("id = ?", articleID).
		Update(pointerField, version.ID)
}

func regenerateArticleEmbedding(ctx context.Context, articleID uuid.UUID, content string) error {
	embeddingService := getEmbeddingService()
	embedding, err := embeddingService.GenerateEmbedding(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	db := database.DB()
	result := db.WithContext(ctx).Model(&models.Article{}).Where("id = ?", articleID).Update("draft_embedding", embedding)
	if result.Error != nil {
		return fmt.Errorf("failed to update article embedding: %w", result.Error)
	}

	return nil
}

