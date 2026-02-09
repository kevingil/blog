package agent

import (
	"context"

	"backend/pkg/database/repository"

	"github.com/google/uuid"
)

// articleDraftAdapter implements ArticleDraftService using the article repository.
type articleDraftAdapter struct {
	repo repository.ArticleRepository
}

// NewArticleDraftService creates an ArticleDraftService backed by the article repository.
func NewArticleDraftService(repo repository.ArticleRepository) ArticleDraftService {
	return &articleDraftAdapter{repo: repo}
}

func (a *articleDraftAdapter) CreateDraftSnapshot(ctx context.Context, articleID uuid.UUID) (*uuid.UUID, error) {
	return a.repo.CreateDraftSnapshot(ctx, articleID)
}

func (a *articleDraftAdapter) UpdateDraftContent(ctx context.Context, articleID uuid.UUID, htmlContent string) error {
	return a.repo.UpdateDraftContent(ctx, articleID, htmlContent)
}
