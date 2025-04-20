package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"blog-agent-go/backend/models"

	"gorm.io/gorm"
)

type ArticleService struct {
	db          *gorm.DB
	writerAgent *WriterAgent
}

func NewArticleService(db *gorm.DB, writerAgent *WriterAgent) *ArticleService {
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

type ArticleListItem struct {
	ID                       int64    `json:"id"`
	Title                    string   `json:"title"`
	Slug                     string   `json:"slug"`
	Image                    string   `json:"image"`
	Content                  string   `json:"content"`
	CreatedAt                int64    `json:"created_at"`
	PublishedAt              *int64   `json:"published_at"`
	Author                   string   `json:"author"`
	Tags                     []string `json:"tags"`
	IsDraft                  bool     `json:"is_draft"`
	ImageGenerationRequestID *string  `json:"image_generation_request_id"`
}

type ArticleListResponse struct {
	Articles   []ArticleListItem `json:"articles"`
	TotalPages int               `json:"total_pages"`
}

const ITEMS_PER_PAGE = 6

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

func (s *ArticleService) GetArticle(id int64) (*ArticleListItem, error) {
	var article models.Article
	if err := s.db.Preload("Tags").First(&article, id).Error; err != nil {
		return nil, err
	}

	var author models.User
	if err := s.db.First(&author, article.Author).Error; err != nil {
		return nil, err
	}

	tagNames := make([]string, len(article.Tags))
	for i, tag := range article.Tags {
		tagNames[i] = tag.Name
	}

	return &ArticleListItem{
		ID:                       article.ID,
		Title:                    article.Title,
		Slug:                     article.Slug,
		Image:                    article.Image,
		Content:                  article.Content,
		CreatedAt:                article.CreatedAt,
		PublishedAt:              article.PublishedAt,
		Author:                   author.Name,
		Tags:                     tagNames,
		IsDraft:                  article.IsDraft,
		ImageGenerationRequestID: &article.ImageGenerationRequestID,
	}, nil
}

func (s *ArticleService) GetArticles(page int, tag string) (*ArticleListResponse, error) {
	offset := (page - 1) * ITEMS_PER_PAGE
	query := s.db.Model(&models.Article{}).
		Select("articles.*, users.name as author").
		Joins("LEFT JOIN users ON articles.author = users.id").
		Where("articles.is_draft = ?", false)

	if tag != "" && tag != "All" {
		query = query.Joins("LEFT JOIN article_tags ON articles.id = article_tags.article_id").
			Joins("LEFT JOIN tags ON article_tags.tag_id = tags.tag_id").
			Where("tags.name = ?", tag)
	}

	// Get total count for pagination
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	// Get articles with pagination
	var articles []struct {
		models.Article
		Author string `gorm:"column:author"`
	}
	if err := query.Offset(offset).Limit(ITEMS_PER_PAGE).Order("articles.published_at DESC").Find(&articles).Error; err != nil {
		return nil, err
	}

	// Get tags for each article
	articleList := make([]ArticleListItem, len(articles))
	for i, article := range articles {
		var tags []models.Tag
		if err := s.db.Model(&article.Article).Association("Tags").Find(&tags); err != nil {
			return nil, err
		}

		tagNames := make([]string, len(tags))
		for j, tag := range tags {
			tagNames[j] = tag.Name
		}

		articleList[i] = ArticleListItem{
			ID:                       article.ID,
			Title:                    article.Title,
			Slug:                     article.Slug,
			Image:                    article.Image,
			Content:                  article.Content,
			CreatedAt:                article.CreatedAt,
			PublishedAt:              article.PublishedAt,
			Author:                   article.Author,
			Tags:                     tagNames,
			IsDraft:                  article.IsDraft,
			ImageGenerationRequestID: &article.ImageGenerationRequestID,
		}
	}

	return &ArticleListResponse{
		Articles:   articleList,
		TotalPages: int(math.Ceil(float64(totalCount) / float64(ITEMS_PER_PAGE))),
	}, nil
}

func (s *ArticleService) SearchArticles(query string, page int, tag string) (*ArticleListResponse, error) {
	offset := (page - 1) * ITEMS_PER_PAGE
	searchQuery := s.db.Model(&models.Article{}).
		Select("articles.*, users.name as author").
		Joins("LEFT JOIN users ON articles.author = users.id").
		Where("articles.is_draft = ?", false).
		Where("articles.title LIKE ? OR articles.content LIKE ?", "%"+query+"%", "%"+query+"%")

	if tag != "" && tag != "All" {
		searchQuery = searchQuery.Joins("LEFT JOIN article_tags ON articles.id = article_tags.article_id").
			Joins("LEFT JOIN tags ON article_tags.tag_id = tags.tag_id").
			Where("tags.name = ?", tag)
	}

	// Get total count for pagination
	var totalCount int64
	if err := searchQuery.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	// Get articles with pagination
	var articles []struct {
		models.Article
		Author string `gorm:"column:author"`
	}
	if err := searchQuery.Offset(offset).Limit(ITEMS_PER_PAGE).Order("articles.published_at DESC").Find(&articles).Error; err != nil {
		return nil, err
	}

	// Get tags for each article
	articleList := make([]ArticleListItem, len(articles))
	for i, article := range articles {
		var tags []models.Tag
		if err := s.db.Model(&article.Article).Association("Tags").Find(&tags); err != nil {
			return nil, err
		}

		tagNames := make([]string, len(tags))
		for j, tag := range tags {
			tagNames[j] = tag.Name
		}

		articleList[i] = ArticleListItem{
			ID:                       article.ID,
			Title:                    article.Title,
			Slug:                     article.Slug,
			Image:                    article.Image,
			Content:                  article.Content,
			CreatedAt:                article.CreatedAt,
			PublishedAt:              article.PublishedAt,
			Author:                   article.Author,
			Tags:                     tagNames,
			IsDraft:                  article.IsDraft,
			ImageGenerationRequestID: &article.ImageGenerationRequestID,
		}
	}

	return &ArticleListResponse{
		Articles:   articleList,
		TotalPages: int(math.Ceil(float64(totalCount) / float64(ITEMS_PER_PAGE))),
	}, nil
}

func (s *ArticleService) GetPopularTags() ([]string, error) {
	var tags []struct {
		TagName string `gorm:"column:tag_name"`
		Count   int
	}

	if err := s.db.Model(&models.Tag{}).
		Select("tags.tag_name, COUNT(article_tags.article_id) as count").
		Joins("LEFT JOIN article_tags ON tags.tag_id = article_tags.tag_id").
		Joins("LEFT JOIN articles ON article_tags.article_id = articles.id").
		Where("articles.is_draft = ?", false).
		Group("tags.tag_name").
		Order("count DESC").
		Limit(10).
		Find(&tags).Error; err != nil {
		return nil, err
	}

	tagNames := make([]string, len(tags))

	for i, tag := range tags {
		tagNames[i] = tag.TagName
	}

	return tagNames, nil
}

type ArticleMetadata struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type ArticleData struct {
	Article    models.Article `json:"article"`
	Tags       []TagData      `json:"tags"`
	AuthorName string         `json:"author_name"`
}

type TagData struct {
	ArticleID int64  `json:"article_id"`
	TagID     int64  `json:"tag_id"`
	TagName   string `json:"tag_name"`
}

type RecommendedArticle struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Slug        string  `json:"slug"`
	Image       *string `json:"image"`
	PublishedAt *int64  `json:"published_at"`
	CreatedAt   int64   `json:"created_at"`
	Author      *string `json:"author"`
}

type ArticleRow struct {
	ID          int64    `json:"id"`
	Title       *string  `json:"title"`
	Content     *string  `json:"content"`
	CreatedAt   int64    `json:"created_at"`
	PublishedAt *int64   `json:"published_at"`
	IsDraft     bool     `json:"is_draft"`
	Slug        *string  `json:"slug"`
	Tags        []string `json:"tags"`
	Image       *string  `json:"image"`
}

func (s *ArticleService) GetArticleMetadata(slug string) (*ArticleMetadata, error) {
	var article models.Article
	if err := s.db.Where("slug = ?", slug).First(&article).Error; err != nil {
		return nil, err
	}

	description := ""
	if article.Content != "" {
		description = article.Content
		if len(description) > 160 {
			description = description[:160]
		}
	}

	return &ArticleMetadata{
		Title:       article.Title,
		Description: description,
	}, nil
}

func (s *ArticleService) GetArticleData(slug string) (*ArticleData, error) {
	var article models.Article
	if err := s.db.Where("slug = ?", slug).First(&article).Error; err != nil {
		return nil, err
	}

	var tagData []TagData
	if err := s.db.Table("article_tags").
		Select("article_tags.article_id, article_tags.tag_id, tags.tag_name as tag_name").
		Joins("LEFT JOIN tags ON article_tags.tag_id = tags.tag_id").
		Where("article_tags.article_id = ?", article.ID).
		Scan(&tagData).Error; err != nil {
		return nil, err
	}

	var author models.User
	if err := s.db.Unscoped().Where("id = ?", article.Author).First(&author).Error; err != nil {
		return nil, err
	}

	return &ArticleData{
		Article:    article,
		Tags:       tagData,
		AuthorName: author.Name,
	}, nil
}

func (s *ArticleService) GetRecommendedArticles(currentArticleID int64) ([]RecommendedArticle, error) {
	var articles []struct {
		models.Article
		AuthorName string `gorm:"column:author_name"`
	}

	if err := s.db.Table("articles").
		Select("articles.*, users.name as author_name").
		Joins("LEFT JOIN users ON articles.author = users.id").
		Where("articles.id != ? AND articles.is_draft = ?", currentArticleID, false).
		Order("articles.published_at DESC").
		Limit(3).
		Scan(&articles).Error; err != nil {
		return nil, err
	}

	recommended := make([]RecommendedArticle, len(articles))
	for i, article := range articles {
		recommended[i] = RecommendedArticle{
			ID:          article.ID,
			Title:       article.Title,
			Slug:        article.Slug,
			Image:       &article.Image,
			PublishedAt: article.PublishedAt,
			CreatedAt:   article.CreatedAt,
			Author:      &article.AuthorName,
		}
	}

	return recommended, nil
}

func (s *ArticleService) GetDashboardArticles() ([]ArticleRow, error) {
	var articles []struct {
		models.Article
		Tags string `gorm:"column:tags"`
	}

	if err := s.db.Table("articles").
		Select("articles.*, GROUP_CONCAT(tags.name) as tags").
		Joins("LEFT JOIN article_tags ON articles.id = article_tags.article_id").
		Joins("LEFT JOIN tags ON article_tags.tag_id = tags.tag_id").
		Group("articles.id").
		Order("articles.published_at DESC, articles.created_at DESC").
		Scan(&articles).Error; err != nil {
		return nil, err
	}

	rows := make([]ArticleRow, len(articles))
	for i, article := range articles {
		tags := []string{}
		if article.Tags != "" {
			tags = strings.Split(article.Tags, ",")
		}

		rows[i] = ArticleRow{
			ID:          article.ID,
			Title:       &article.Title,
			Content:     &article.Content,
			CreatedAt:   article.CreatedAt,
			PublishedAt: article.PublishedAt,
			IsDraft:     article.IsDraft,
			Slug:        &article.Slug,
			Tags:        tags,
			Image:       &article.Image,
		}
	}

	return rows, nil
}

func (s *ArticleService) DeleteArticle(id int64) error {
	// Delete article tag map
	if err := s.db.Where("article_id = ?", id).Delete(&models.ArticleTag{}).Error; err != nil {
		return err
	}

	// Delete article
	if err := s.db.Delete(&models.Article{}, id).Error; err != nil {
		return err
	}

	// Delete tags that no longer have article tag map references
	if err := s.db.Exec(`
		DELETE FROM tags
		WHERE NOT EXISTS (
			SELECT 1 FROM article_tags
			WHERE article_tags.tag_id = tags.tag_id
		)
	`).Error; err != nil {
		return err
	}

	return nil
}
