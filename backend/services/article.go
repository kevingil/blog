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
	Article models.Article `json:"article"`
	Author  AuthorData     `json:"author"`
	Tags    []TagData      `json:"tags"`
}

type ArticleUpdateRequest struct {
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Image       string   `json:"image"`
	Tags        []string `json:"tags"`
	IsDraft     bool     `json:"is_draft"`
	PublishedAt int64    `json:"published_at"`
}

type ArticleListResponse struct {
	Articles      []ArticleListItem `json:"articles"`
	TotalPages    int               `json:"total_pages"`
	IncludeDrafts bool              `json:"include_drafts"`
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

func (s *ArticleService) UpdateArticle(ctx context.Context, articleID int64, req ArticleUpdateRequest) (*ArticleListItem, error) {
	db := s.db.GetDB()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Update article fields
	_, err = tx.Exec(`UPDATE articles SET 
		title = ?, content = ?, image = ?, is_draft = ?, published_at = ?, updated_at = ? 
		WHERE id = ?`,
		req.Title, req.Content, req.Image, req.IsDraft, req.PublishedAt, time.Now().Unix(), articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to update article: %w", err)
	}

	// Remove existing tag relationships for this article
	_, err = tx.Exec("DELETE FROM article_tags WHERE article_id = ?", articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete existing article tags: %w", err)
	}

	// Process tags
	var processedTags []string
	for _, tagName := range req.Tags {
		if tagName == "" {
			continue
		}

		// Normalize tag name to lowercase and trim whitespace
		normalizedTag := strings.ToLower(strings.TrimSpace(tagName))
		if normalizedTag == "" {
			continue
		}

		// Avoid duplicate tags in the same request
		isDuplicate := false
		for _, existing := range processedTags {
			if existing == normalizedTag {
				isDuplicate = true
				break
			}
		}
		if isDuplicate {
			continue
		}
		processedTags = append(processedTags, normalizedTag)

		// Check if tag exists (case-insensitive lookup)
		var tagID int64
		err = tx.QueryRow("SELECT tag_id FROM tags WHERE LOWER(tag_name) = ?", normalizedTag).Scan(&tagID)

		if err == sql.ErrNoRows {
			// Tag doesn't exist, create it
			result, err := tx.Exec("INSERT INTO tags (tag_name) VALUES (?)", normalizedTag)
			if err != nil {
				return nil, fmt.Errorf("failed to create tag '%s': %w", normalizedTag, err)
			}
			tagID, err = result.LastInsertId()
			if err != nil {
				return nil, fmt.Errorf("failed to get new tag ID: %w", err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("failed to check tag existence: %w", err)
		}

		// Create article-tag relationship (avoid duplicates)
		_, err = tx.Exec(`INSERT OR IGNORE INTO article_tags (article_id, tag_id) VALUES (?, ?)`,
			articleID, tagID)
		if err != nil {
			return nil, fmt.Errorf("failed to create article-tag relationship: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Get updated article with tags
	article, err := s.GetArticle(articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated article: %w", err)
	}

	return article, nil
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

func (s *ArticleService) GetArticleIDBySlug(slug string) (int64, error) {
	db := s.db.GetDB()
	var articleID int64

	err := db.QueryRow("SELECT id FROM articles WHERE slug = ?", slug).Scan(&articleID)
	if err != nil {
		return 0, err
	}

	return articleID, nil
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

	var imageGenRequestIDPtr string
	if imageGenRequestID.Valid {
		imageGenRequestIDPtr = imageGenRequestID.String
	}

	// Build TagData slice
	var tagData []TagData
	for _, name := range tagNames {
		tagData = append(tagData, TagData{ArticleID: article.ID, TagID: 0, TagName: name})
	}

	return &ArticleListItem{
		Article: models.Article{
			ID:                       article.ID,
			Title:                    article.Title,
			Slug:                     article.Slug,
			Image:                    article.Image,
			Content:                  article.Content,
			CreatedAt:                article.CreatedAt,
			PublishedAt:              article.PublishedAt,
			IsDraft:                  article.IsDraft,
			Embedding:                article.Embedding,
			ImageGenerationRequestID: imageGenRequestIDPtr,
			ChatHistory:              article.ChatHistory,
		},
		Author: AuthorData{
			ID:   article.Author,
			Name: authorName,
		},
		Tags: tagData,
	}, nil
}

func (s *ArticleService) GetArticles(page int, tag string, includeDrafts bool) (*ArticleListResponse, error) {
	db := s.db.GetDB()
	offset := (page - 1) * ITEMS_PER_PAGE

	// Base SELECT and FROM clauses
	selectClause := `SELECT a.id, a.image, a.slug, a.title, a.content, a.author, a.created_at, a.updated_at, ` +
		`a.is_draft, a.embedding, a.image_generation_request_id, a.published_at, a.chat_history, u.name as author_name`
	fromClause := ` FROM articles a LEFT JOIN users u ON a.author = u.id `

	// We only need the tag join if the caller filtered by tag
	joinTagClause := ``
	conditions := []string{}
	args := []interface{}{}

	// Apply draft filter only when drafts should be excluded
	// includeDrafts == true  -> no filter (show drafts + published)
	// includeDrafts == false -> show only published (is_draft = 0)
	if !includeDrafts {
		conditions = append(conditions, "a.is_draft = 0")
	}

	if tag != "" && tag != "All" {
		joinTagClause = ` LEFT JOIN article_tags at ON a.id = at.article_id LEFT JOIN tags t ON at.tag_id = t.tag_id `
		conditions = append(conditions, "t.tag_name = ?")
		args = append(args, tag)
	}

	// Assemble WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Order clause (optionally add LIMIT/OFFSET for pagination)
	orderClause := " ORDER BY COALESCE(a.published_at, a.created_at) DESC"
	if page != 0 {
		orderClause += " LIMIT ? OFFSET ?"
		args = append(args, ITEMS_PER_PAGE, offset)
	}

	// Final query for data
	baseQuery := selectClause + fromClause + joinTagClause + whereClause + orderClause

	// Build count query (for pagination)
	countSelect := "SELECT COUNT(DISTINCT a.id)"
	countQuery := countSelect + fromClause + joinTagClause + whereClause

	// Total count
	var totalCount int64
	// If page == 0, args might not include limit/offset, so use args as-is
	countArgs := args
	if page != 0 {
		// exclude the limit & offset values which are the last two items
		countArgs = args[:len(args)-2]
	}
	if err := db.QueryRow(countQuery, countArgs...).Scan(&totalCount); err != nil {
		return nil, err
	}

	// Fetch rows
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
		var image sql.NullString

		if err := rows.Scan(&article.ID, &image, &article.Slug, &article.Title, &article.Content,
			&article.Author, &article.CreatedAt, &article.UpdatedAt, &article.IsDraft, &article.Embedding,
			&imageGenRequestID, &article.PublishedAt, &article.ChatHistory, &authorName); err != nil {
			return nil, err
		}

		// Get tags for this article
		tagRows, err := db.Query(`SELECT t.tag_name FROM tags t JOIN article_tags at ON t.tag_id = at.tag_id WHERE at.article_id = ?`, article.ID)
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

		var imageGenRequestIDPtr string
		if imageGenRequestID.Valid {
			imageGenRequestIDPtr = imageGenRequestID.String
		}

		// Build TagData slice
		var tagData []TagData
		for _, name := range tagNames {
			tagData = append(tagData, TagData{ArticleID: article.ID, TagID: 0, TagName: name})
		}

		if image.Valid {
			article.Image = image.String
		}

		articleList = append(articleList, ArticleListItem{
			Article: models.Article{
				ID:                       article.ID,
				Title:                    article.Title,
				Slug:                     article.Slug,
				Image:                    article.Image,
				Content:                  article.Content,
				CreatedAt:                article.CreatedAt,
				PublishedAt:              article.PublishedAt,
				IsDraft:                  article.IsDraft,
				Embedding:                article.Embedding,
				ImageGenerationRequestID: imageGenRequestIDPtr,
				ChatHistory:              article.ChatHistory,
			},
			Author: AuthorData{
				ID:   article.Author,
				Name: authorName,
			},
			Tags: tagData,
		})
	}

	return &ArticleListResponse{
		Articles:      articleList,
		TotalPages:    int(math.Ceil(float64(totalCount) / float64(ITEMS_PER_PAGE))),
		IncludeDrafts: includeDrafts,
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

		var imageGenRequestIDPtr string
		if imageGenRequestID.Valid {
			imageGenRequestIDPtr = imageGenRequestID.String
		}

		// Build TagData slice
		var tagData []TagData
		for _, name := range tagNames {
			tagData = append(tagData, TagData{ArticleID: article.ID, TagID: 0, TagName: name})
		}

		articleList = append(articleList, ArticleListItem{
			Article: models.Article{
				ID:                       article.ID,
				Title:                    article.Title,
				Slug:                     article.Slug,
				Image:                    article.Image,
				Content:                  article.Content,
				CreatedAt:                article.CreatedAt,
				PublishedAt:              article.PublishedAt,
				IsDraft:                  article.IsDraft,
				Embedding:                article.Embedding,
				ImageGenerationRequestID: imageGenRequestIDPtr,
				ChatHistory:              article.ChatHistory,
			},
			Author: AuthorData{
				ID:   article.Author,
				Name: authorName,
			},
			Tags: tagData,
		})
	}

	return &ArticleListResponse{
		Articles:      articleList,
		TotalPages:    int(math.Ceil(float64(totalCount) / float64(ITEMS_PER_PAGE))),
		IncludeDrafts: false,
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
	Article models.Article `json:"article"`
	Tags    []TagData      `json:"tags"`
	Author  AuthorData     `json:"author"`
}

type AuthorData struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
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
	var author AuthorData
	err = db.QueryRow("SELECT id, name FROM users WHERE id = ?", article.Author).Scan(&author.ID, &author.Name)
	if err != nil {
		return nil, err
	}

	return &ArticleData{
		Article: article,
		Tags:    tagData,
		Author:  author,
	}, nil
}

// TODO: Use embeddings
func (s *ArticleService) GetRecommendedArticles(currentArticleID int64) ([]RecommendedArticle, error) {
	db := s.db.GetDB()

	rows, err := db.Query(`SELECT a.id, a.title, a.slug, a.image, a.published_at, a.created_at, u.name as author_name 
		FROM articles a 
		LEFT JOIN users u ON a.author = u.id 
		WHERE a.id != ? AND a.is_draft = ? 
		ORDER BY a.published_at DESC 
		LIMIT 3`, currentArticleID, 0)
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

type ArticleCreateRequest struct {
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Image    string   `json:"image"`
	Tags     []string `json:"tags"`
	IsDraft  bool     `json:"isDraft"`
	AuthorID int64    `json:"authorId"`
}

func (s *ArticleService) CreateArticle(ctx context.Context, req ArticleCreateRequest) (*ArticleListItem, error) {
	db := s.db.GetDB()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate slug from title
	slug := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.Title), " ", "-"))
	// Remove special characters and ensure it's URL-safe
	slug = strings.ReplaceAll(slug, ",", "")
	slug = strings.ReplaceAll(slug, ".", "")
	slug = strings.ReplaceAll(slug, "!", "")
	slug = strings.ReplaceAll(slug, "?", "")

	now := time.Now().Unix()
	var publishedAt *int64
	if !req.IsDraft {
		publishedAt = &now
	}

	// Insert article
	result, err := tx.Exec(`INSERT INTO articles 
		(title, content, image, slug, author, created_at, updated_at, is_draft, published_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Title, req.Content, req.Image, slug, req.AuthorID, now, now, req.IsDraft, publishedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create article: %w", err)
	}

	articleID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get new article ID: %w", err)
	}

	// Process tags (same logic as UpdateArticle)
	var processedTags []string
	for _, tagName := range req.Tags {
		if tagName == "" {
			continue
		}

		// Normalize tag name to lowercase and trim whitespace
		normalizedTag := strings.ToLower(strings.TrimSpace(tagName))
		if normalizedTag == "" {
			continue
		}

		// Avoid duplicate tags in the same request
		isDuplicate := false
		for _, existing := range processedTags {
			if existing == normalizedTag {
				isDuplicate = true
				break
			}
		}
		if isDuplicate {
			continue
		}
		processedTags = append(processedTags, normalizedTag)

		// Check if tag exists (case-insensitive lookup)
		var tagID int64
		err = tx.QueryRow("SELECT tag_id FROM tags WHERE LOWER(tag_name) = ?", normalizedTag).Scan(&tagID)

		if err == sql.ErrNoRows {
			// Tag doesn't exist, create it
			tagResult, err := tx.Exec("INSERT INTO tags (tag_name) VALUES (?)", normalizedTag)
			if err != nil {
				return nil, fmt.Errorf("failed to create tag '%s': %w", normalizedTag, err)
			}
			tagID, err = tagResult.LastInsertId()
			if err != nil {
				return nil, fmt.Errorf("failed to get new tag ID: %w", err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("failed to check tag existence: %w", err)
		}

		// Create article-tag relationship
		_, err = tx.Exec(`INSERT INTO article_tags (article_id, tag_id) VALUES (?, ?)`,
			articleID, tagID)
		if err != nil {
			return nil, fmt.Errorf("failed to create article-tag relationship: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Get created article with tags
	article, err := s.GetArticle(articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created article: %w", err)
	}

	return article, nil
}
