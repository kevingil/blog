package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"blog-agent-go/backend/database"
	"blog-agent-go/backend/models"
)

type ArticleService struct {
	db          database.Service
	writerAgent *WriterAgent
}

func NewArticleService(db database.Service, writerAgent *WriterAgent) *ArticleService {
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

	db := s.db.GetDB()
	_, err = db.Exec(`INSERT INTO articles (image, slug, title, content, author, created_at, updated_at, is_draft, embedding, image_generation_request_id, published_at, chat_history) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		article.Image, article.Slug, article.Title, article.Content, article.Author, article.CreatedAt, article.UpdatedAt,
		article.IsDraft, article.Embedding, article.ImageGenerationRequestID, article.PublishedAt, article.ChatHistory)
	if err != nil {
		return nil, err
	}

	return article, nil
}

func (s *ArticleService) GetArticleChatHistory(ctx context.Context, articleID int64) (*ArticleChatHistory, error) {
	db := s.db.GetDB()
	var chatHistory []byte

	err := db.QueryRow("SELECT chat_history FROM articles WHERE id = ?", articleID).Scan(&chatHistory)
	if err != nil {
		return nil, err
	}

	if chatHistory == nil {
		return nil, nil
	}

	var history ArticleChatHistory
	if err := json.Unmarshal(chatHistory, &history); err != nil {
		return nil, err
	}

	return &history, nil
}

func (s *ArticleService) UpdateArticleWithContext(ctx context.Context, articleID int64) (*models.Article, error) {
	db := s.db.GetDB()
	var article models.Article
	var imageGenRequestID sql.NullString

	err := db.QueryRow(`SELECT id, image, slug, title, content, author, created_at, updated_at, is_draft, embedding, image_generation_request_id, published_at, chat_history 
		FROM articles WHERE id = ?`, articleID).Scan(
		&article.ID, &article.Image, &article.Slug, &article.Title, &article.Content, &article.Author,
		&article.CreatedAt, &article.UpdatedAt, &article.IsDraft, &article.Embedding,
		&imageGenRequestID, &article.PublishedAt, &article.ChatHistory)
	if err != nil {
		return nil, err
	}

	// Handle NULL image_generation_request_id
	if imageGenRequestID.Valid {
		article.ImageGenerationRequestID = imageGenRequestID.String
	}

	updatedContent, err := s.writerAgent.UpdateWithContext(ctx, &article)
	if err != nil {
		return nil, fmt.Errorf("error updating article content: %w", err)
	}

	article.Content = updatedContent
	article.UpdatedAt = time.Now().Unix()

	_, err = db.Exec("UPDATE articles SET content = ?, updated_at = ? WHERE id = ?", article.Content, article.UpdatedAt, articleID)
	if err != nil {
		return nil, err
	}

	return &article, nil
}

func (s *ArticleService) GetArticle(id int64) (*ArticleListItem, error) {
	db := s.db.GetDB()
	var article models.Article
	var authorName string
	var imageGenRequestID sql.NullString

	err := db.QueryRow(`SELECT a.id, a.image, a.slug, a.title, a.content, a.author, a.created_at, a.updated_at, a.is_draft, 
		a.embedding, a.image_generation_request_id, a.published_at, a.chat_history, u.name 
		FROM articles a LEFT JOIN users u ON a.author = u.id WHERE a.id = ?`, id).Scan(
		&article.ID, &article.Image, &article.Slug, &article.Title, &article.Content, &article.Author,
		&article.CreatedAt, &article.UpdatedAt, &article.IsDraft, &article.Embedding,
		&imageGenRequestID, &article.PublishedAt, &article.ChatHistory, &authorName)
	if err != nil {
		return nil, err
	}

	// Get tags for this article
	rows, err := db.Query(`SELECT t.tag_name FROM tags t 
		JOIN article_tags at ON t.tag_id = at.tag_id 
		WHERE at.article_id = ?`, article.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tagNames []string
	for rows.Next() {
		var tagName string
		if err := rows.Scan(&tagName); err != nil {
			return nil, err
		}
		tagNames = append(tagNames, tagName)
	}

	var imageGenRequestIDPtr *string
	if imageGenRequestID.Valid {
		imageGenRequestIDPtr = &imageGenRequestID.String
	}

	return &ArticleListItem{
		ID:                       article.ID,
		Title:                    article.Title,
		Slug:                     article.Slug,
		Image:                    article.Image,
		Content:                  article.Content,
		CreatedAt:                article.CreatedAt,
		PublishedAt:              article.PublishedAt,
		Author:                   authorName,
		Tags:                     tagNames,
		IsDraft:                  article.IsDraft,
		ImageGenerationRequestID: imageGenRequestIDPtr,
	}, nil
}

func (s *ArticleService) GetArticles(page int, tag string) (*ArticleListResponse, error) {
	db := s.db.GetDB()
	offset := (page - 1) * ITEMS_PER_PAGE

	var baseQuery string
	var countQuery string
	var args []interface{}

	if tag != "" && tag != "All" {
		baseQuery = `SELECT a.id, a.image, a.slug, a.title, a.content, a.author, a.created_at, a.updated_at, 
			a.is_draft, a.embedding, a.image_generation_request_id, a.published_at, a.chat_history, u.name as author_name
			FROM articles a 
			LEFT JOIN users u ON a.author = u.id 
			LEFT JOIN article_tags at ON a.id = at.article_id 
			LEFT JOIN tags t ON at.tag_id = t.tag_id 
			WHERE a.is_draft = ? AND t.tag_name = ? 
			ORDER BY a.published_at DESC LIMIT ? OFFSET ?`
		countQuery = `SELECT COUNT(DISTINCT a.id) FROM articles a 
			LEFT JOIN article_tags at ON a.id = at.article_id 
			LEFT JOIN tags t ON at.tag_id = t.tag_id 
			WHERE a.is_draft = ? AND t.tag_name = ?`
		args = []interface{}{false, tag, ITEMS_PER_PAGE, offset}
	} else {
		baseQuery = `SELECT a.id, a.image, a.slug, a.title, a.content, a.author, a.created_at, a.updated_at, 
			a.is_draft, a.embedding, a.image_generation_request_id, a.published_at, a.chat_history, u.name as author_name
			FROM articles a 
			LEFT JOIN users u ON a.author = u.id 
			WHERE a.is_draft = ? 
			ORDER BY a.published_at DESC LIMIT ? OFFSET ?`
		countQuery = `SELECT COUNT(*) FROM articles WHERE is_draft = ?`
		args = []interface{}{false, ITEMS_PER_PAGE, offset}
	}

	// Get total count for pagination
	var totalCount int64
	var countArgs []interface{}
	if tag != "" && tag != "All" {
		countArgs = []interface{}{false, tag}
	} else {
		countArgs = []interface{}{false}
	}
	err := db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, err
	}

	// Get articles with pagination
	rows, err := db.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articleList []ArticleListItem
	for rows.Next() {
		var article models.Article
		var authorName string
		var imageGenRequestID sql.NullString

		err := rows.Scan(&article.ID, &article.Image, &article.Slug, &article.Title, &article.Content,
			&article.Author, &article.CreatedAt, &article.UpdatedAt, &article.IsDraft, &article.Embedding,
			&imageGenRequestID, &article.PublishedAt, &article.ChatHistory, &authorName)
		if err != nil {
			return nil, err
		}

		// Get tags for this article
		tagRows, err := db.Query(`SELECT t.tag_name FROM tags t 
			JOIN article_tags at ON t.tag_id = at.tag_id 
			WHERE at.article_id = ?`, article.ID)
		if err != nil {
			return nil, err
		}

		var tagNames []string
		for tagRows.Next() {
			var tagName string
			if err := tagRows.Scan(&tagName); err != nil {
				tagRows.Close()
				return nil, err
			}
			tagNames = append(tagNames, tagName)
		}
		tagRows.Close()

		var imageGenRequestIDPtr *string
		if imageGenRequestID.Valid {
			imageGenRequestIDPtr = &imageGenRequestID.String
		}

		articleList = append(articleList, ArticleListItem{
			ID:                       article.ID,
			Title:                    article.Title,
			Slug:                     article.Slug,
			Image:                    article.Image,
			Content:                  article.Content,
			CreatedAt:                article.CreatedAt,
			PublishedAt:              article.PublishedAt,
			Author:                   authorName,
			Tags:                     tagNames,
			IsDraft:                  article.IsDraft,
			ImageGenerationRequestID: imageGenRequestIDPtr,
		})
	}

	return &ArticleListResponse{
		Articles:   articleList,
		TotalPages: int(math.Ceil(float64(totalCount) / float64(ITEMS_PER_PAGE))),
	}, nil
}

func (s *ArticleService) SearchArticles(query string, page int, tag string) (*ArticleListResponse, error) {
	db := s.db.GetDB()
	offset := (page - 1) * ITEMS_PER_PAGE
	searchTerm := "%" + query + "%"

	var baseQuery string
	var countQuery string
	var args []interface{}

	if tag != "" && tag != "All" {
		baseQuery = `SELECT a.id, a.image, a.slug, a.title, a.content, a.author, a.created_at, a.updated_at, 
			a.is_draft, a.embedding, a.image_generation_request_id, a.published_at, a.chat_history, u.name as author_name
			FROM articles a 
			LEFT JOIN users u ON a.author = u.id 
			LEFT JOIN article_tags at ON a.id = at.article_id 
			LEFT JOIN tags t ON at.tag_id = t.tag_id 
			WHERE a.is_draft = ? AND (a.title LIKE ? OR a.content LIKE ?) AND t.tag_name = ? 
			ORDER BY a.published_at DESC LIMIT ? OFFSET ?`
		countQuery = `SELECT COUNT(DISTINCT a.id) FROM articles a 
			LEFT JOIN article_tags at ON a.id = at.article_id 
			LEFT JOIN tags t ON at.tag_id = t.tag_id 
			WHERE a.is_draft = ? AND (a.title LIKE ? OR a.content LIKE ?) AND t.tag_name = ?`
		args = []interface{}{false, searchTerm, searchTerm, tag, ITEMS_PER_PAGE, offset}
	} else {
		baseQuery = `SELECT a.id, a.image, a.slug, a.title, a.content, a.author, a.created_at, a.updated_at, 
			a.is_draft, a.embedding, a.image_generation_request_id, a.published_at, a.chat_history, u.name as author_name
			FROM articles a 
			LEFT JOIN users u ON a.author = u.id 
			WHERE a.is_draft = ? AND (a.title LIKE ? OR a.content LIKE ?) 
			ORDER BY a.published_at DESC LIMIT ? OFFSET ?`
		countQuery = `SELECT COUNT(*) FROM articles WHERE is_draft = ? AND (title LIKE ? OR content LIKE ?)`
		args = []interface{}{false, searchTerm, searchTerm, ITEMS_PER_PAGE, offset}
	}

	// Get total count for pagination
	var totalCount int64
	var countArgs []interface{}
	if tag != "" && tag != "All" {
		countArgs = []interface{}{false, searchTerm, searchTerm, tag}
	} else {
		countArgs = []interface{}{false, searchTerm, searchTerm}
	}
	err := db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, err
	}

	// Get articles with pagination
	rows, err := db.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articleList []ArticleListItem
	for rows.Next() {
		var article models.Article
		var authorName string
		var imageGenRequestID sql.NullString

		err := rows.Scan(&article.ID, &article.Image, &article.Slug, &article.Title, &article.Content,
			&article.Author, &article.CreatedAt, &article.UpdatedAt, &article.IsDraft, &article.Embedding,
			&imageGenRequestID, &article.PublishedAt, &article.ChatHistory, &authorName)
		if err != nil {
			return nil, err
		}

		// Get tags for this article
		tagRows, err := db.Query(`SELECT t.tag_name FROM tags t 
			JOIN article_tags at ON t.tag_id = at.tag_id 
			WHERE at.article_id = ?`, article.ID)
		if err != nil {
			return nil, err
		}

		var tagNames []string
		for tagRows.Next() {
			var tagName string
			if err := tagRows.Scan(&tagName); err != nil {
				tagRows.Close()
				return nil, err
			}
			tagNames = append(tagNames, tagName)
		}
		tagRows.Close()

		var imageGenRequestIDPtr *string
		if imageGenRequestID.Valid {
			imageGenRequestIDPtr = &imageGenRequestID.String
		}

		articleList = append(articleList, ArticleListItem{
			ID:                       article.ID,
			Title:                    article.Title,
			Slug:                     article.Slug,
			Image:                    article.Image,
			Content:                  article.Content,
			CreatedAt:                article.CreatedAt,
			PublishedAt:              article.PublishedAt,
			Author:                   authorName,
			Tags:                     tagNames,
			IsDraft:                  article.IsDraft,
			ImageGenerationRequestID: imageGenRequestIDPtr,
		})
	}

	return &ArticleListResponse{
		Articles:   articleList,
		TotalPages: int(math.Ceil(float64(totalCount) / float64(ITEMS_PER_PAGE))),
	}, nil
}

func (s *ArticleService) GetPopularTags() ([]string, error) {
	db := s.db.GetDB()

	rows, err := db.Query(`SELECT t.tag_name, COUNT(at.article_id) as count 
		FROM tags t 
		LEFT JOIN article_tags at ON t.tag_id = at.tag_id 
		LEFT JOIN articles a ON at.article_id = a.id 
		WHERE a.is_draft = ? 
		GROUP BY t.tag_name 
		ORDER BY count DESC 
		LIMIT 10`, false)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tagNames []string
	for rows.Next() {
		var tagName string
		var count int
		if err := rows.Scan(&tagName, &count); err != nil {
			return nil, err
		}
		tagNames = append(tagNames, tagName)
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
	db := s.db.GetDB()
	var article models.Article

	err := db.QueryRow("SELECT title, content FROM articles WHERE slug = ?", slug).Scan(&article.Title, &article.Content)
	if err != nil {
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
	db := s.db.GetDB()
	var article models.Article
	var imageGenRequestID sql.NullString

	fmt.Println("slug", slug)

	err := db.QueryRow(`SELECT id, image, slug, title, content, author, created_at, updated_at, is_draft, 
		embedding, image_generation_request_id, published_at, chat_history 
		FROM articles WHERE slug = ?`, slug).Scan(
		&article.ID, &article.Image, &article.Slug, &article.Title, &article.Content, &article.Author,
		&article.CreatedAt, &article.UpdatedAt, &article.IsDraft, &article.Embedding,
		&imageGenRequestID, &article.PublishedAt, &article.ChatHistory)
	if err != nil {
		return nil, err
	}

	// Handle NULL image_generation_request_id
	if imageGenRequestID.Valid {
		article.ImageGenerationRequestID = imageGenRequestID.String
	}

	// Get tag data
	tagRows, err := db.Query(`SELECT at.article_id, at.tag_id, t.tag_name 
		FROM article_tags at 
		LEFT JOIN tags t ON at.tag_id = t.tag_id 
		WHERE at.article_id = ?`, article.ID)
	if err != nil {
		return nil, err
	}
	defer tagRows.Close()

	var tagData []TagData
	for tagRows.Next() {
		var tag TagData
		if err := tagRows.Scan(&tag.ArticleID, &tag.TagID, &tag.TagName); err != nil {
			return nil, err
		}
		tagData = append(tagData, tag)
	}

	// Get author
	var authorName string
	err = db.QueryRow("SELECT name FROM users WHERE id = ?", article.Author).Scan(&authorName)
	if err != nil {
		return nil, err
	}

	return &ArticleData{
		Article:    article,
		Tags:       tagData,
		AuthorName: authorName,
	}, nil
}

func (s *ArticleService) GetRecommendedArticles(currentArticleID int64) ([]RecommendedArticle, error) {
	db := s.db.GetDB()

	rows, err := db.Query(`SELECT a.id, a.title, a.slug, a.image, a.published_at, a.created_at, u.name as author_name 
		FROM articles a 
		LEFT JOIN users u ON a.author = u.id 
		WHERE a.id != ? AND a.is_draft = ? 
		ORDER BY a.published_at DESC 
		LIMIT 3`, currentArticleID, false)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recommended []RecommendedArticle
	for rows.Next() {
		var article RecommendedArticle
		var image sql.NullString
		var authorName sql.NullString

		err := rows.Scan(&article.ID, &article.Title, &article.Slug, &image,
			&article.PublishedAt, &article.CreatedAt, &authorName)
		if err != nil {
			return nil, err
		}

		if image.Valid {
			article.Image = &image.String
		}
		if authorName.Valid {
			article.Author = &authorName.String
		}

		recommended = append(recommended, article)
	}

	return recommended, nil
}

func (s *ArticleService) GetDashboardArticles() ([]ArticleRow, error) {
	db := s.db.GetDB()

	rows, err := db.Query(`SELECT a.id, a.title, a.content, a.created_at, a.published_at, a.is_draft, a.slug, a.image,
		GROUP_CONCAT(t.tag_name) as tags 
		FROM articles a 
		LEFT JOIN article_tags at ON a.id = at.article_id 
		LEFT JOIN tags t ON at.tag_id = t.tag_id 
		GROUP BY a.id 
		ORDER BY a.published_at DESC, a.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []ArticleRow
	for rows.Next() {
		var article ArticleRow
		var tagsStr sql.NullString

		err := rows.Scan(&article.ID, &article.Title, &article.Content, &article.CreatedAt,
			&article.PublishedAt, &article.IsDraft, &article.Slug, &article.Image, &tagsStr)
		if err != nil {
			return nil, err
		}

		if tagsStr.Valid && tagsStr.String != "" {
			article.Tags = strings.Split(tagsStr.String, ",")
		} else {
			article.Tags = []string{}
		}

		articles = append(articles, article)
	}

	return articles, nil
}

func (s *ArticleService) DeleteArticle(id int64) error {
	db := s.db.GetDB()

	// Delete article tag map
	_, err := db.Exec("DELETE FROM article_tags WHERE article_id = ?", id)
	if err != nil {
		return err
	}

	// Delete article
	_, err = db.Exec("DELETE FROM articles WHERE id = ?", id)
	if err != nil {
		return err
	}

	// Delete tags that no longer have article tag map references
	_, err = db.Exec(`DELETE FROM tags WHERE NOT EXISTS (
		SELECT 1 FROM article_tags WHERE article_tags.tag_id = tags.tag_id
	)`)
	if err != nil {
		return err
	}

	return nil
}
