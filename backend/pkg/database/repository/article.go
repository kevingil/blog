package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"backend/pkg/core"
	"backend/pkg/core/article"
	"backend/pkg/database/models"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// ArticleRepository implements article.ArticleStore using GORM
type ArticleRepository struct {
	db *gorm.DB
}

// NewArticleRepository creates a new ArticleRepository
func NewArticleRepository(db *gorm.DB) *ArticleRepository {
	return &ArticleRepository{db: db}
}

// FindByID retrieves an article by its ID
func (r *ArticleRepository) FindByID(ctx context.Context, id uuid.UUID) (*article.Article, error) {
	var model models.ArticleModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return model.ToCore(), nil
}

// FindBySlug retrieves an article by its slug
func (r *ArticleRepository) FindBySlug(ctx context.Context, slug string) (*article.Article, error) {
	var model models.ArticleModel
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return model.ToCore(), nil
}

// List retrieves articles with pagination and filtering
func (r *ArticleRepository) List(ctx context.Context, opts article.ListOptions) ([]article.Article, int64, error) {
	var articleModels []models.ArticleModel
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ArticleModel{})

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

	// Convert to domain types
	articles := make([]article.Article, len(articleModels))
	for i, m := range articleModels {
		articles[i] = *m.ToCore()
	}

	return articles, total, nil
}

// Search performs full-text search on articles
func (r *ArticleRepository) Search(ctx context.Context, opts article.SearchOptions) ([]article.Article, int64, error) {
	var articleModels []models.ArticleModel
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ArticleModel{})

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

	// Convert to domain types
	articles := make([]article.Article, len(articleModels))
	for i, m := range articleModels {
		articles[i] = *m.ToCore()
	}

	return articles, total, nil
}

// SearchByEmbedding performs vector similarity search
func (r *ArticleRepository) SearchByEmbedding(ctx context.Context, embedding []float32, limit int) ([]article.Article, error) {
	var articleModels []models.ArticleModel

	embeddingVector := pgvector.NewVector(embedding)
	query := fmt.Sprintf(
		"SELECT * FROM article WHERE draft_embedding IS NOT NULL ORDER BY draft_embedding <-> '%s' LIMIT %d",
		embeddingVector.String(),
		limit,
	)

	if err := r.db.WithContext(ctx).Raw(query).Scan(&articleModels).Error; err != nil {
		return nil, err
	}

	articles := make([]article.Article, len(articleModels))
	for i, m := range articleModels {
		articles[i] = *m.ToCore()
	}

	return articles, nil
}

// Save creates or updates an article
func (r *ArticleRepository) Save(ctx context.Context, a *article.Article) error {
	model := models.ArticleModelFromCore(a)

	// Check if article exists
	var existing models.ArticleModel
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
func (r *ArticleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.ArticleModel{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// GetPopularTags returns the most frequently used tag IDs
func (r *ArticleRepository) GetPopularTags(ctx context.Context, limit int) ([]int64, error) {
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
func (r *ArticleRepository) SaveDraft(ctx context.Context, a *article.Article) error {
	now := time.Now()

	// 1. Update article table synchronously (draft fields only)
	err := r.db.WithContext(ctx).Model(&models.ArticleModel{}).
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
func (r *ArticleRepository) Publish(ctx context.Context, a *article.Article) error {
	now := time.Now()

	// 1. Copy draft fields to published fields synchronously
	err := r.db.WithContext(ctx).Model(&models.ArticleModel{}).
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
func (r *ArticleRepository) Unpublish(ctx context.Context, a *article.Article) error {
	now := time.Now()

	err := r.db.WithContext(ctx).Model(&models.ArticleModel{}).
		Where("id = ?", a.ID).
		Updates(map[string]interface{}{
			"published_title":             nil,
			"published_content":           nil,
			"published_image_url":         nil,
			"published_embedding":         nil,
			"published_at":                nil,
			"current_published_version_id": nil,
			"updated_at":                  now,
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
func (r *ArticleRepository) ListVersions(ctx context.Context, articleID uuid.UUID) ([]article.ArticleVersion, error) {
	var versionModels []models.ArticleVersionModel

	err := r.db.WithContext(ctx).
		Where("article_id = ?", articleID).
		Order("version_number DESC").
		Find(&versionModels).Error
	if err != nil {
		return nil, err
	}

	versions := make([]article.ArticleVersion, len(versionModels))
	for i, m := range versionModels {
		versions[i] = *m.ToCore()
	}

	return versions, nil
}

// GetVersion retrieves a specific version by ID
func (r *ArticleRepository) GetVersion(ctx context.Context, versionID uuid.UUID) (*article.ArticleVersion, error) {
	var model models.ArticleVersionModel

	if err := r.db.WithContext(ctx).First(&model, versionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	return model.ToCore(), nil
}

// RevertToVersion creates a new draft by copying content from a historical version
func (r *ArticleRepository) RevertToVersion(ctx context.Context, articleID, versionID uuid.UUID) error {
	// Get the version to revert to
	var versionModel models.ArticleVersionModel
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

	err := r.db.WithContext(ctx).Model(&models.ArticleModel{}).
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
func (r *ArticleRepository) createVersionAsync(articleID uuid.UUID, title, content, imageURL string, embedding []float32, status string, editedBy uuid.UUID) {
	ctx := context.Background()

	// Get next version number
	var maxVersion int
	r.db.Model(&models.ArticleVersionModel{}).
		Where("article_id = ?", articleID).
		Select("COALESCE(MAX(version_number), 0)").
		Scan(&maxVersion)

	var embeddingVector pgvector.Vector
	if len(embedding) > 0 {
		embeddingVector = pgvector.NewVector(embedding)
	}

	var editedByPtr *uuid.UUID
	if editedBy != uuid.Nil {
		editedByPtr = &editedBy
	}

	version := &models.ArticleVersionModel{
		ID:            uuid.New(),
		ArticleID:     articleID,
		VersionNumber: maxVersion + 1,
		Status:        status,
		Title:         title,
		Content:       content,
		ImageURL:      imageURL,
		Embedding:     embeddingVector,
		EditedBy:      editedByPtr,
		CreatedAt:     time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(version).Error; err != nil {
		log.Printf("Failed to create version for article %s: %v", articleID, err)
		return
	}

	// Update version pointer on article
	pointerField := "current_draft_version_id"
	if status == "published" {
		pointerField = "current_published_version_id"
	}
	r.db.Model(&models.ArticleModel{}).
		Where("id = ?", articleID).
		Update(pointerField, version.ID)
}
