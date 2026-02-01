package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"backend/pkg/core"
	"backend/pkg/database/models"
	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ArticleRepository defines the interface for article data access
type ArticleRepository interface {
	// Basic CRUD operations
	FindByID(ctx context.Context, id uuid.UUID) (*types.Article, error)
	FindBySlug(ctx context.Context, slug string) (*types.Article, error)
	List(ctx context.Context, opts types.ArticleListOptions) ([]types.Article, int64, error)
	Search(ctx context.Context, opts types.ArticleSearchOptions) ([]types.Article, int64, error)
	SearchByEmbedding(ctx context.Context, embedding []float32, limit int) ([]types.Article, error)
	Save(ctx context.Context, article *types.Article) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetPopularTags(ctx context.Context, limit int) ([]int64, error)

	// Slug uniqueness check
	SlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error)

	// Version management operations
	SaveDraft(ctx context.Context, article *types.Article) error
	Publish(ctx context.Context, article *types.Article) error
	Unpublish(ctx context.Context, article *types.Article) error
	ListVersions(ctx context.Context, articleID uuid.UUID) ([]types.ArticleVersion, error)
	GetVersion(ctx context.Context, versionID uuid.UUID) (*types.ArticleVersion, error)
	RevertToVersion(ctx context.Context, articleID, versionID uuid.UUID) error
}

// articleRepository implements data access for articles using GORM
type articleRepository struct {
	db *gorm.DB
}

// NewArticleRepository creates a new ArticleRepository
func NewArticleRepository(db *gorm.DB) ArticleRepository {
	return &articleRepository{db: db}
}

// articleModelToType converts a database model to types
func articleModelToType(m *models.Article) *types.Article {
	var sessionMemory map[string]interface{}
	if m.SessionMemory != nil {
		_ = json.Unmarshal(m.SessionMemory, &sessionMemory)
	}

	var draftEmbedding []float32
	if m.DraftEmbedding.Slice() != nil {
		draftEmbedding = m.DraftEmbedding.Slice()
	}

	var publishedEmbedding []float32
	if m.PublishedEmbedding.Slice() != nil {
		publishedEmbedding = m.PublishedEmbedding.Slice()
	}

	return &types.Article{
		ID:                        m.ID,
		Slug:                      m.Slug,
		AuthorID:                  m.AuthorID,
		TagIDs:                    m.TagIDs,
		DraftTitle:                m.DraftTitle,
		DraftContent:              m.DraftContent,
		DraftImageURL:             m.DraftImageURL,
		DraftEmbedding:            draftEmbedding,
		PublishedTitle:            m.PublishedTitle,
		PublishedContent:          m.PublishedContent,
		PublishedImageURL:         m.PublishedImageURL,
		PublishedEmbedding:        publishedEmbedding,
		PublishedAt:               m.PublishedAt,
		CurrentDraftVersionID:     m.CurrentDraftVersionID,
		CurrentPublishedVersionID: m.CurrentPublishedVersionID,
		ImagenRequestID:           m.ImagenRequestID,
		SessionMemory:             sessionMemory,
		CreatedAt:                 m.CreatedAt,
		UpdatedAt:                 m.UpdatedAt,
	}
}

// articleTypeToModel creates a GORM model from the types
func articleTypeToModel(a *types.Article) *models.Article {
	var sessionMemory datatypes.JSON
	if a.SessionMemory != nil {
		data, _ := json.Marshal(a.SessionMemory)
		sessionMemory = datatypes.JSON(data)
	}

	var draftEmbedding pgvector.Vector
	if len(a.DraftEmbedding) > 0 {
		draftEmbedding = pgvector.NewVector(a.DraftEmbedding)
	}

	var publishedEmbedding pgvector.Vector
	if len(a.PublishedEmbedding) > 0 {
		publishedEmbedding = pgvector.NewVector(a.PublishedEmbedding)
	}

	return &models.Article{
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

// articleVersionModelToType converts a version model to types
func articleVersionModelToType(m *models.ArticleVersion) *types.ArticleVersion {
	var embedding []float32
	if m.Embedding.Slice() != nil {
		embedding = m.Embedding.Slice()
	}

	return &types.ArticleVersion{
		ID:            m.ID,
		ArticleID:     m.ArticleID,
		VersionNumber: m.VersionNumber,
		Status:        m.Status,
		Title:         m.Title,
		Content:       m.Content,
		ImageURL:      m.ImageURL,
		Embedding:     embedding,
		EditedBy:      m.EditedBy,
		CreatedAt:     m.CreatedAt,
	}
}

// FindByID retrieves an article by its ID
func (r *articleRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Article, error) {
	var model models.Article
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return articleModelToType(&model), nil
}

// FindBySlug retrieves an article by its slug
func (r *articleRepository) FindBySlug(ctx context.Context, slug string) (*types.Article, error) {
	var model models.Article
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return articleModelToType(&model), nil
}

// List retrieves articles with pagination and filtering
func (r *articleRepository) List(ctx context.Context, opts types.ArticleListOptions) ([]types.Article, int64, error) {
	var articleModels []models.Article
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Article{})

	// Apply filters
	if opts.PublishedOnly {
		query = query.Where("published_at IS NOT NULL")
	}
	if opts.AuthorID != nil {
		query = query.Where("author_id = ?", *opts.AuthorID)
	}
	if opts.TagID != nil {
		query = query.Where("? = ANY(tag_ids)", *opts.TagID)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (opts.Page - 1) * opts.PerPage
	if err := query.Offset(offset).Limit(opts.PerPage).Order("created_at DESC").Find(&articleModels).Error; err != nil {
		return nil, 0, err
	}

	// Convert to types
	articles := make([]types.Article, len(articleModels))
	for i, m := range articleModels {
		articles[i] = *articleModelToType(&m)
	}

	return articles, total, nil
}

// Search performs full-text search on articles
func (r *articleRepository) Search(ctx context.Context, opts types.ArticleSearchOptions) ([]types.Article, int64, error) {
	var articleModels []models.Article
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Article{})

	// Apply search filter - search in both draft and published content
	if opts.Query != "" {
		searchPattern := "%" + opts.Query + "%"
		query = query.Where(
			"draft_title ILIKE ? OR draft_content ILIKE ? OR published_title ILIKE ? OR published_content ILIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	// Apply published filter
	if opts.PublishedOnly {
		query = query.Where("published_at IS NOT NULL")
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (opts.Page - 1) * opts.PerPage
	if err := query.Offset(offset).Limit(opts.PerPage).Order("created_at DESC").Find(&articleModels).Error; err != nil {
		return nil, 0, err
	}

	// Convert to types
	articles := make([]types.Article, len(articleModels))
	for i, m := range articleModels {
		articles[i] = *articleModelToType(&m)
	}

	return articles, total, nil
}

// SearchByEmbedding performs vector similarity search
func (r *articleRepository) SearchByEmbedding(ctx context.Context, embedding []float32, limit int) ([]types.Article, error) {
	var articleModels []models.Article

	embeddingVector := pgvector.NewVector(embedding)
	query := fmt.Sprintf(
		"SELECT * FROM article WHERE draft_embedding IS NOT NULL ORDER BY draft_embedding <-> '%s' LIMIT %d",
		embeddingVector.String(),
		limit,
	)

	if err := r.db.WithContext(ctx).Raw(query).Scan(&articleModels).Error; err != nil {
		return nil, err
	}

	articles := make([]types.Article, len(articleModels))
	for i, m := range articleModels {
		articles[i] = *articleModelToType(&m)
	}

	return articles, nil
}

// Save creates or updates an article
func (r *articleRepository) Save(ctx context.Context, a *types.Article) error {
	model := articleTypeToModel(a)

	// Check if article exists
	var existing models.Article
	err := r.db.WithContext(ctx).First(&existing, a.ID).Error
	if err == gorm.ErrRecordNotFound {
		// Create new article
		if a.ID == uuid.Nil {
			a.ID = uuid.New()
			model.ID = a.ID
		}
		return r.db.WithContext(ctx).Create(model).Error
	} else if err != nil {
		return err
	}

	// Update existing article
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete removes an article by its ID
func (r *articleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Article{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// SlugExists checks if a slug already exists, optionally excluding a specific article ID
func (r *articleRepository) SlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.Article{}).Where("slug = ?", slug)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetPopularTags returns the most frequently used tag IDs
func (r *articleRepository) GetPopularTags(ctx context.Context, limit int) ([]int64, error) {
	var results []struct {
		TagID int64 `gorm:"column:tag_id"`
		Count int   `gorm:"column:count"`
	}

	query := `
		SELECT unnest(tag_ids) as tag_id, COUNT(*) as count 
		FROM article 
		WHERE published_at IS NOT NULL 
		GROUP BY tag_id 
		ORDER BY count DESC 
		LIMIT ?
	`

	if err := r.db.WithContext(ctx).Raw(query, limit).Scan(&results).Error; err != nil {
		return nil, err
	}

	tagIDs := make([]int64, len(results))
	for i, r := range results {
		tagIDs[i] = r.TagID
	}

	return tagIDs, nil
}

// SaveDraft updates cached draft content and creates version asynchronously
func (r *articleRepository) SaveDraft(ctx context.Context, a *types.Article) error {
	now := time.Now()

	// 1. Update article table synchronously (draft fields only)
	err := r.db.WithContext(ctx).Model(&models.Article{}).
		Where("id = ?", a.ID).
		Updates(map[string]interface{}{
			"draft_title":     a.DraftTitle,
			"draft_content":   a.DraftContent,
			"draft_image_url": a.DraftImageURL,
			"updated_at":      now,
		}).Error
	if err != nil {
		return err
	}

	a.UpdatedAt = now

	// 2. Create version record asynchronously
	go r.createVersionAsync(a.ID, a.DraftTitle, a.DraftContent, a.DraftImageURL, a.DraftEmbedding, "draft", a.AuthorID)

	return nil
}

// Publish copies draft to published and creates version asynchronously
func (r *articleRepository) Publish(ctx context.Context, a *types.Article) error {
	now := time.Now()

	// 1. Copy draft fields to published fields synchronously
	err := r.db.WithContext(ctx).Model(&models.Article{}).
		Where("id = ?", a.ID).
		Updates(map[string]interface{}{
			"published_title":     a.DraftTitle,
			"published_content":   a.DraftContent,
			"published_image_url": a.DraftImageURL,
			"published_embedding": func() interface{} {
				if len(a.DraftEmbedding) > 0 {
					return pgvector.NewVector(a.DraftEmbedding)
				}
				return nil
			}(),
			"published_at": now,
			"updated_at":   now,
		}).Error
	if err != nil {
		return err
	}

	// Update domain object
	a.PublishedTitle = &a.DraftTitle
	a.PublishedContent = &a.DraftContent
	a.PublishedImageURL = &a.DraftImageURL
	a.PublishedEmbedding = a.DraftEmbedding
	a.PublishedAt = &now
	a.UpdatedAt = now

	// 2. Create published version asynchronously
	go r.createVersionAsync(a.ID, a.DraftTitle, a.DraftContent, a.DraftImageURL, a.DraftEmbedding, "published", a.AuthorID)

	return nil
}

// Unpublish removes published status while preserving version history
func (r *articleRepository) Unpublish(ctx context.Context, a *types.Article) error {
	now := time.Now()

	err := r.db.WithContext(ctx).Model(&models.Article{}).
		Where("id = ?", a.ID).
		Updates(map[string]interface{}{
			"published_title":              nil,
			"published_content":            nil,
			"published_image_url":          nil,
			"published_embedding":          nil,
			"published_at":                 nil,
			"current_published_version_id": nil,
			"updated_at":                   now,
		}).Error
	if err != nil {
		return err
	}

	// Update domain object
	a.PublishedTitle = nil
	a.PublishedContent = nil
	a.PublishedImageURL = nil
	a.PublishedEmbedding = nil
	a.PublishedAt = nil
	a.CurrentPublishedVersionID = nil
	a.UpdatedAt = now

	return nil
}

// ListVersions retrieves all versions for an article
func (r *articleRepository) ListVersions(ctx context.Context, articleID uuid.UUID) ([]types.ArticleVersion, error) {
	var versionModels []models.ArticleVersion

	err := r.db.WithContext(ctx).
		Where("article_id = ?", articleID).
		Order("version_number DESC").
		Find(&versionModels).Error
	if err != nil {
		return nil, err
	}

	versions := make([]types.ArticleVersion, len(versionModels))
	for i, m := range versionModels {
		versions[i] = *articleVersionModelToType(&m)
	}

	return versions, nil
}

// GetVersion retrieves a specific version by ID
func (r *articleRepository) GetVersion(ctx context.Context, versionID uuid.UUID) (*types.ArticleVersion, error) {
	var model models.ArticleVersion

	if err := r.db.WithContext(ctx).First(&model, versionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	return articleVersionModelToType(&model), nil
}

// RevertToVersion creates a new draft by copying content from a historical version
func (r *articleRepository) RevertToVersion(ctx context.Context, articleID, versionID uuid.UUID) error {
	// Get the version to revert to
	var versionModel models.ArticleVersion
	if err := r.db.WithContext(ctx).First(&versionModel, versionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return core.ErrNotFound
		}
		return err
	}

	// Verify the version belongs to this article
	if versionModel.ArticleID != articleID {
		return core.InvalidInputError("Version does not belong to this article")
	}

	now := time.Now()

	// Update draft fields with version content
	var embedding interface{}
	if versionModel.Embedding.Slice() != nil && len(versionModel.Embedding.Slice()) > 0 {
		embedding = versionModel.Embedding
	}

	err := r.db.WithContext(ctx).Model(&models.Article{}).
		Where("id = ?", articleID).
		Updates(map[string]interface{}{
			"draft_title":     versionModel.Title,
			"draft_content":   versionModel.Content,
			"draft_image_url": versionModel.ImageURL,
			"draft_embedding": embedding,
			"updated_at":      now,
		}).Error
	if err != nil {
		return err
	}

	// Create a new draft version asynchronously (to record the revert action)
	go r.createVersionAsync(
		articleID,
		versionModel.Title,
		versionModel.Content,
		versionModel.ImageURL,
		versionModel.Embedding.Slice(),
		"draft",
		uuid.Nil, // TODO: pass actual user ID when available
	)

	return nil
}

// createVersionAsync creates a version record asynchronously
func (r *articleRepository) createVersionAsync(articleID uuid.UUID, title, content, imageURL string, embedding []float32, status string, editedBy uuid.UUID) {
	ctx := context.Background()

	// Get next version number
	var maxVersion int
	r.db.Model(&models.ArticleVersion{}).
		Where("article_id = ?", articleID).
		Select("COALESCE(MAX(version_number), 0)").
		Scan(&maxVersion)

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
		err = r.db.WithContext(ctx).Create(version).Error
	} else {
		err = r.db.WithContext(ctx).Omit("Embedding").Create(version).Error
	}

	if err != nil {
		log.Printf("Failed to create version for article %s: %v", articleID, err)
		return
	}

	// Update version pointer on article
	pointerField := "current_draft_version_id"
	if status == "published" {
		pointerField = "current_published_version_id"
	}
	r.db.Model(&models.Article{}).
		Where("id = ?", articleID).
		Update(pointerField, version.ID)
}
