package source

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"backend/pkg/core"
	"backend/pkg/core/ml"
	"backend/pkg/database"
	"backend/pkg/database/models"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
	"gorm.io/gorm"
)

// CreateRequest represents a request to create a source
type CreateRequest struct {
	ArticleID  uuid.UUID `json:"article_id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	URL        string    `json:"url"`
	SourceType string    `json:"source_type"`
}

// UpdateRequest represents a request to update a source
type UpdateRequest struct {
	Title      *string `json:"title,omitempty"`
	Content    *string `json:"content,omitempty"`
	URL        *string `json:"url,omitempty"`
	SourceType *string `json:"source_type,omitempty"`
}

// ScrapedContent represents scraped content from a URL
type ScrapedContent struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	URL     string `json:"url"`
}

// SourceWithArticle includes article metadata with the source
type SourceWithArticle struct {
	models.Source
	ArticleTitle string `json:"article_title"`
	ArticleSlug  string `json:"article_slug"`
}

// ListResponse represents a paginated list of sources
type ListResponse struct {
	Sources    []SourceWithArticle `json:"sources"`
	TotalPages int                 `json:"total_pages"`
	Page       int                 `json:"page"`
}

// getEmbeddingService returns an embedding service instance
func getEmbeddingService() *ml.EmbeddingService {
	return ml.NewEmbeddingService()
}

// GetByID retrieves a source by its ID
func GetByID(ctx context.Context, id uuid.UUID) (*models.Source, error) {
	db := database.DB()
	var source models.Source

	if err := db.WithContext(ctx).First(&source, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	return &source, nil
}

// GetByArticleID retrieves all sources for an article
func GetByArticleID(ctx context.Context, articleID uuid.UUID) ([]*models.Source, error) {
	db := database.DB()
	var sources []*models.Source

	if err := db.WithContext(ctx).Where("article_id = ?", articleID).
		Order("created_at DESC").
		Find(&sources).Error; err != nil {
		return nil, err
	}

	return sources, nil
}

// List retrieves all sources with pagination and article metadata
func List(ctx context.Context, page, limit int) (*ListResponse, error) {
	db := database.DB()

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	var total int64
	if err := db.WithContext(ctx).Model(&models.Source{}).Count(&total).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	var sources []SourceWithArticle
	err := db.WithContext(ctx).Table("article_source").
		Select("article_source.*, article.draft_title as article_title, article.slug as article_slug").
		Joins("LEFT JOIN article ON article.id = article_source.article_id").
		Order("article_source.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&sources).Error

	if err != nil {
		return nil, err
	}

	return &ListResponse{
		Sources:    sources,
		TotalPages: totalPages,
		Page:       page,
	}, nil
}

// Create creates a new source with embedding generation
func Create(ctx context.Context, req CreateRequest) (*models.Source, error) {
	db := database.DB()
	embeddingService := getEmbeddingService()

	// Validate that the article exists
	var article models.Article
	if err := db.WithContext(ctx).First(&article, req.ArticleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	// Generate embedding for the content
	embedding, err := embeddingService.GenerateEmbedding(ctx, req.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Set default source type if not provided
	sourceType := req.SourceType
	if sourceType == "" {
		if req.URL != "" {
			sourceType = "web"
		} else {
			sourceType = "manual"
		}
	}

	source := &models.Source{
		ArticleID:  req.ArticleID,
		Title:      req.Title,
		Content:    req.Content,
		URL:        req.URL,
		SourceType: sourceType,
		Embedding:  embedding,
	}

	if err := db.WithContext(ctx).Create(source).Error; err != nil {
		return nil, err
	}

	return source, nil
}

// ScrapeAndCreate scrapes content from a URL and creates a source
func ScrapeAndCreate(ctx context.Context, articleID uuid.UUID, targetURL string) (*models.Source, error) {
	// Scrape the content
	scraped, err := scrapeURL(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape URL: %w", err)
	}

	// Determine source type based on URL and content
	sourceType := "web"
	if isPDFURL(targetURL) {
		sourceType = "pdf"
	}

	// Create the source
	req := CreateRequest{
		ArticleID:  articleID,
		Title:      scraped.Title,
		Content:    scraped.Content,
		URL:        scraped.URL,
		SourceType: sourceType,
	}

	return Create(ctx, req)
}

// Update updates an existing source
func Update(ctx context.Context, sourceID uuid.UUID, req UpdateRequest) (*models.Source, error) {
	db := database.DB()
	embeddingService := getEmbeddingService()

	var source models.Source
	if err := db.WithContext(ctx).First(&source, sourceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	needsEmbeddingUpdate := false

	if req.Title != nil {
		source.Title = *req.Title
	}
	if req.Content != nil {
		source.Content = *req.Content
		needsEmbeddingUpdate = true
	}
	if req.URL != nil {
		source.URL = *req.URL
	}
	if req.SourceType != nil {
		source.SourceType = *req.SourceType
	}

	if needsEmbeddingUpdate {
		embedding, err := embeddingService.GenerateEmbedding(ctx, source.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding: %w", err)
		}
		source.Embedding = embedding
	}

	if err := db.WithContext(ctx).Save(&source).Error; err != nil {
		return nil, err
	}

	return &source, nil
}

// Delete removes a source by its ID
func Delete(ctx context.Context, id uuid.UUID) error {
	db := database.DB()

	result := db.WithContext(ctx).Delete(&models.Source{}, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}

	return nil
}

// SearchSimilar finds sources similar to the given query using vector similarity
func SearchSimilar(ctx context.Context, articleID uuid.UUID, query string, limit int) ([]*models.Source, error) {
	embeddingService := getEmbeddingService()

	queryEmbedding, err := embeddingService.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	db := database.DB()
	var sources []*models.Source

	err = db.WithContext(ctx).Raw(`
		SELECT * FROM article_source 
		WHERE article_id = ? AND embedding IS NOT NULL
		ORDER BY embedding <-> ? 
		LIMIT ?`,
		articleID, queryEmbedding, limit).
		Scan(&sources).Error

	if err != nil {
		return nil, fmt.Errorf("failed to search similar sources: %w", err)
	}

	return sources, nil
}

// Scraping helper functions

func isPDFURL(targetURL string) bool {
	if strings.HasSuffix(strings.ToLower(targetURL), ".pdf") {
		return true
	}

	lowerURL := strings.ToLower(targetURL)
	return strings.Contains(lowerURL, ".pdf") ||
		strings.Contains(lowerURL, "/pdf/") ||
		strings.Contains(lowerURL, "content-type=application/pdf")
}

func scrapeURL(targetURL string) (*ScrapedContent, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
		targetURL = parsedURL.String()
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; BlogAgent/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if strings.Contains(contentType, "application/pdf") ||
		(len(bodyData) > 4 && string(bodyData[:4]) == "%PDF") {

		title, content, err := extractTextFromPDF(bodyData)
		if err != nil {
			return nil, fmt.Errorf("failed to extract PDF content: %w", err)
		}

		return &ScrapedContent{
			Title:   title,
			Content: content,
			URL:     targetURL,
		}, nil
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	title := doc.Find("title").Text()
	if title == "" {
		title = doc.Find("h1").First().Text()
	}
	title = strings.TrimSpace(title)

	content := extractMainContent(doc)

	return &ScrapedContent{
		Title:   title,
		Content: content,
		URL:     targetURL,
	}, nil
}

func extractTextFromPDF(pdfData []byte) (string, string, error) {
	reader := bytes.NewReader(pdfData)

	pdfReader, err := pdf.NewReader(reader, int64(len(pdfData)))
	if err != nil {
		return "", "", fmt.Errorf("failed to open PDF: %w", err)
	}

	var textContent strings.Builder
	var title string

	for i := 1; i <= pdfReader.NumPage(); i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		pageText, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		if title == "" && pageText != "" {
			lines := strings.Split(pageText, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if len(line) > 0 && len(line) < 200 {
					title = line
					break
				}
			}
		}

		textContent.WriteString(pageText)
		textContent.WriteString("\n\n")
	}

	content := strings.TrimSpace(textContent.String())
	if content == "" {
		return "", "", fmt.Errorf("no text content found in PDF")
	}

	if title == "" {
		title = "PDF Document"
	}

	return title, content, nil
}

func extractMainContent(doc *goquery.Document) string {
	var content strings.Builder

	selectors := []string{
		"article",
		"main",
		".content",
		".post-content",
		".entry-content",
		".article-content",
		"#content",
		".main-content",
	}

	var contentNode *goquery.Selection

	for _, selector := range selectors {
		node := doc.Find(selector).First()
		if node.Length() > 0 {
			contentNode = node
			break
		}
	}

	if contentNode == nil {
		contentNode = doc.Find("body")
	}

	contentNode.Find("p, h1, h2, h3, h4, h5, h6, li").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" && len(text) > 10 {
			content.WriteString(text)
			content.WriteString("\n\n")
		}
	})

	result := strings.TrimSpace(content.String())

	if len(result) > 5000 {
		result = result[:5000] + "..."
	}

	return result
}

