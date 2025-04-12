package blog

import (
	"context"
	"encoding/json"
	"time"

	"blog-agent-go/internal/models"

	"gorm.io/gorm"
)

type ArticleService struct {
	db *gorm.DB
}

func NewArticleService(db *gorm.DB) *ArticleService {
	return &ArticleService{db: db}
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
	// TODO: Implement LLM integration for article generation
	article := &models.Article{
		Title:     title,
		Content:   prompt, // This will be replaced with generated content
		Author:    authorID,
		IsDraft:   draft,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

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

	// TODO: Implement LLM integration for article update
	article.UpdatedAt = time.Now().Unix()

	if err := s.db.Save(&article).Error; err != nil {
		return nil, err
	}

	return &article, nil
}
