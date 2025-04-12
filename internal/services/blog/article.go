package blog

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"blog-agent-go/internal/models"
	"blog-agent-go/internal/services/agents"

	"gorm.io/gorm"
)

type ArticleService struct {
	db          *gorm.DB
	writerAgent *agents.WriterAgent
}

func NewArticleService(db *gorm.DB, writerAgent *agents.WriterAgent) *ArticleService {
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

func (s *ArticleService) GenerateArticle(ctx context.Context, prompt string, title string, authorID int64, draft bool) (*models.Article, error) {
	article, err := s.writerAgent.GenerateArticle(ctx, prompt, title, authorID)
	if err != nil {
		return nil, fmt.Errorf("error generating article: %w", err)
	}

	article.IsDraft = draft
	article.CreatedAt = time.Now().Unix()
	article.UpdatedAt = time.Now().Unix()

	if err := s.db.Create(article).Error; err != nil {
		return nil, err
	}

	return article, nil
}

func (s *ArticleService) GetArticleChatHistory(ctx context.Context, articleID int64) (*ArticleChatHistory, error) {
	var article models.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		return nil, err
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

func (s *ArticleService) UpdateArticleWithContext(ctx context.Context, articleID int64) (*models.Article, error) {
	var article models.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		return nil, err
	}

	updatedContent, err := s.writerAgent.UpdateWithContext(ctx, &article)
	if err != nil {
		return nil, fmt.Errorf("error updating article content: %w", err)
	}

	article.Content = updatedContent
	article.UpdatedAt = time.Now().Unix()

	if err := s.db.Save(&article).Error; err != nil {
		return nil, err
	}

	return &article, nil
}
