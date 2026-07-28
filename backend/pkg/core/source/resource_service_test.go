package source

import (
	"context"
	"testing"
	"time"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
)

type stubSourceRepository struct {
	findByID      func(ctx context.Context, id uuid.UUID) (*types.Source, error)
	findByArticle func(ctx context.Context, articleID uuid.UUID) ([]types.Source, error)
	save          func(ctx context.Context, source *types.Source) error
	update        func(ctx context.Context, source *types.Source) error
}

func (s *stubSourceRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Source, error) {
	return s.findByID(ctx, id)
}

func (s *stubSourceRepository) FindByArticleID(ctx context.Context, articleID uuid.UUID) ([]types.Source, error) {
	return s.findByArticle(ctx, articleID)
}

func (s *stubSourceRepository) List(ctx context.Context, opts types.SourceListOptions) ([]types.SourceWithArticle, int64, error) {
	return nil, 0, nil
}

func (s *stubSourceRepository) SearchSimilar(ctx context.Context, articleID uuid.UUID, embedding []float32, limit int) ([]types.Source, error) {
	return nil, nil
}

func (s *stubSourceRepository) Save(ctx context.Context, source *types.Source) error {
	return s.save(ctx, source)
}

func (s *stubSourceRepository) Update(ctx context.Context, source *types.Source) error {
	return s.update(ctx, source)
}

func (s *stubSourceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

type stubArticleRepository struct {
	findByID func(ctx context.Context, id uuid.UUID) (*types.Article, error)
}

func (s *stubArticleRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Article, error) {
	return s.findByID(ctx, id)
}

func (s *stubArticleRepository) FindBySlug(ctx context.Context, slug string) (*types.Article, error) {
	return nil, nil
}

func (s *stubArticleRepository) List(ctx context.Context, opts types.ArticleListOptions) ([]types.Article, int64, error) {
	return nil, 0, nil
}

func (s *stubArticleRepository) Search(ctx context.Context, opts types.ArticleSearchOptions) ([]types.Article, int64, error) {
	return nil, 0, nil
}

func (s *stubArticleRepository) SearchByEmbedding(ctx context.Context, embedding []float32, limit int) ([]types.Article, error) {
	return nil, nil
}

func (s *stubArticleRepository) Save(ctx context.Context, a *types.Article) error {
	return nil
}

func (s *stubArticleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (s *stubArticleRepository) GetPopularTags(ctx context.Context, limit int) ([]int64, error) {
	return nil, nil
}

func (s *stubArticleRepository) SlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error) {
	return false, nil
}

func (s *stubArticleRepository) SaveDraft(ctx context.Context, a *types.Article) error {
	return nil
}

func (s *stubArticleRepository) Publish(ctx context.Context, a *types.Article, publishedAt *time.Time) error {
	return nil
}

func (s *stubArticleRepository) Unpublish(ctx context.Context, a *types.Article) error {
	return nil
}

func (s *stubArticleRepository) ListVersions(ctx context.Context, articleID uuid.UUID) ([]types.ArticleVersion, error) {
	return nil, nil
}

func (s *stubArticleRepository) GetVersion(ctx context.Context, versionID uuid.UUID) (*types.ArticleVersion, error) {
	return nil, nil
}

func (s *stubArticleRepository) RevertToVersion(ctx context.Context, articleID, versionID uuid.UUID) error {
	return nil
}

func (s *stubArticleRepository) CreateDraftSnapshot(ctx context.Context, articleID uuid.UUID) (*uuid.UUID, error) {
	return nil, nil
}

func (s *stubArticleRepository) UpdateDraftContent(ctx context.Context, articleID uuid.UUID, htmlContent string) error {
	return nil
}

type stubEmbeddingService struct {
	generate func(ctx context.Context, text string) (pgvector.Vector, error)
}

func (s *stubEmbeddingService) GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error) {
	return s.generate(ctx, text)
}

func TestService_UpsertAgentResourceCreatesSource(t *testing.T) {
	ctx := context.Background()
	articleRepo := &stubArticleRepository{}
	embeddingSvc := &stubEmbeddingService{}
	sourceRepo := &stubSourceRepository{}

	articleID := uuid.New()
	svc := NewService(sourceRepo, articleRepo)
	svc.embeddingService = embeddingSvc

	articleRepo.findByID = func(_ context.Context, gotArticleID uuid.UUID) (*types.Article, error) {
		assert.Equal(t, articleID, gotArticleID)
		return &types.Article{ID: articleID}, nil
	}
	sourceRepo.findByArticle = func(_ context.Context, gotArticleID uuid.UUID) ([]types.Source, error) {
		assert.Equal(t, articleID, gotArticleID)
		return []types.Source{}, nil
	}
	embeddingSvc.generate = func(_ context.Context, text string) (pgvector.Vector, error) {
		assert.Equal(t, "selected excerpt", text)
		return pgvector.NewVector([]float32{0.1, 0.2}), nil
	}
	sourceRepo.save = func(_ context.Context, s *types.Source) error {
		resourceMeta, ok := s.MetaData["resource"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, articleID, s.ArticleID)
		assert.Equal(t, "Citation Title", s.Title)
		assert.Equal(t, "selected excerpt", s.Content)
		assert.Equal(t, "ask_question", resourceMeta["origin_tool"])
		assert.Equal(t, "selected excerpt", resourceMeta["selected_excerpt"])
		assert.Equal(t, "used", resourceMeta["usage_status"])
		return nil
	}

	result, err := svc.UpsertAgentResource(ctx, AgentResourceSelection{
		ArticleID:       articleID,
		Title:           "Citation Title",
		URL:             "https://example.com/citation",
		OriginTool:      "ask_question",
		OriginQuestion:  "What changed in Go 1.24?",
		SelectedExcerpt: "selected excerpt",
		RequestID:       "req-123",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Citation Title", result.Title)
	assert.Equal(t, "selected excerpt", result.Content)
}

func TestService_UpsertAgentResourceUpdatesExistingSource(t *testing.T) {
	ctx := context.Background()
	sourceRepo := &stubSourceRepository{}

	articleID := uuid.New()
	sourceID := uuid.New()
	svc := NewService(sourceRepo, nil)

	existing := &types.Source{
		ID:         sourceID,
		ArticleID:  articleID,
		Title:      "Existing Source",
		Content:    "stored content",
		URL:        "https://example.com/source",
		SourceType: "web_search",
		MetaData: map[string]interface{}{
			"resource": map[string]interface{}{
				"origin_tool":  "search_web_sources",
				"usage_status": "available",
			},
		},
		CreatedAt: time.Now(),
	}

	sourceRepo.findByID = func(_ context.Context, gotSourceID uuid.UUID) (*types.Source, error) {
		assert.Equal(t, sourceID, gotSourceID)
		return existing, nil
	}
	sourceRepo.update = func(_ context.Context, s *types.Source) error {
		resourceMeta, ok := s.MetaData["resource"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, sourceID, s.ID)
		assert.Equal(t, "new excerpt", resourceMeta["selected_excerpt"])
		assert.Equal(t, "chunk-1", resourceMeta["selected_excerpt_id"])
		assert.Equal(t, "used", resourceMeta["usage_status"])
		return nil
	}

	result, err := svc.UpsertAgentResource(ctx, AgentResourceSelection{
		ArticleID:         articleID,
		SourceID:          &sourceID,
		SelectedExcerpt:   "new excerpt",
		SelectedExcerptID: "chunk-1",
		RequestID:         "req-456",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, sourceID, result.ID)
}
