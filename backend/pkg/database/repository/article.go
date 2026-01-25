package repository

import (
	"context"
	"fmt"

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
	if opts.IsDraft != nil {
		query = query.Where("is_draft = ?", *opts.IsDraft)
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

	// Apply search filter
	if opts.Query != "" {
		searchPattern := "%" + opts.Query + "%"
		query = query.Where("title ILIKE ? OR content ILIKE ?", searchPattern, searchPattern)
	}

	// Apply draft filter
	if opts.IsDraft != nil {
		query = query.Where("is_draft = ?", *opts.IsDraft)
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
		"SELECT * FROM article WHERE embedding IS NOT NULL ORDER BY embedding <-> '%s' LIMIT %d",
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
		WHERE is_draft = false 
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
