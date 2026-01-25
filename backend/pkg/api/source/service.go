package source

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
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

type ArticleSourceService struct {
	db               database.Service
	embeddingService *ml.EmbeddingService
}

var sourcesSvc *ArticleSourceService

// InitSourcesService initializes the sources service singleton
func InitSourcesService() {
	if sourcesSvc == nil {
		sourcesSvc = NewArticleSourceService(database.New())
	}
}

// Sources returns the sources service singleton
func Sources() *ArticleSourceService {
	if sourcesSvc == nil {
		log.Fatal("ArticleSourceService not initialized. Call InitSourcesService() first.")
	}
	return sourcesSvc
}

func NewArticleSourceService(db database.Service) *ArticleSourceService {
	return &ArticleSourceService{
		db:               db,
		embeddingService: ml.NewEmbeddingService(),
	}
}

type CreateSourceRequest struct {
	ArticleID  uuid.UUID `json:"article_id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	URL        string    `json:"url"`
	SourceType string    `json:"source_type"`
}

type UpdateSourceRequest struct {
	Title      *string `json:"title,omitempty"`
	Content    *string `json:"content,omitempty"`
	URL        *string `json:"url,omitempty"`
	SourceType *string `json:"source_type,omitempty"`
}

type ScrapedContent struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	URL     string `json:"url"`
}

// ArticleSourceWithArticle includes article metadata with the source
type ArticleSourceWithArticle struct {
	models.Source
	ArticleTitle string `json:"article_title"`
	ArticleSlug  string `json:"article_slug"`
}

// CreateSource creates a new article source with embedding generation
func (s *ArticleSourceService) CreateSource(ctx context.Context, req CreateSourceRequest) (*models.Source, error) {
	db := s.db.GetDB()

	// Validate that the article exists
	var article models.Article
	if err := db.First(&article, req.ArticleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Article")
		}
		return nil, core.InternalError("Failed to fetch article")
	}

	// Generate embedding for the content
	embedding, err := s.embeddingService.GenerateEmbedding(ctx, req.Content)
	if err != nil {
		return nil, core.InternalError("Failed to generate embedding")
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

	if err := db.Create(source).Error; err != nil {
		return nil, core.InternalError("Failed to create source")
	}

	return source, nil
}

// ScrapeAndCreateSource scrapes content from a URL and creates a source
func (s *ArticleSourceService) ScrapeAndCreateSource(ctx context.Context, articleID uuid.UUID, targetURL string) (*models.Source, error) {
	// Scrape the content
	scraped, err := s.scrapeURL(targetURL)
	if err != nil {
		return nil, core.InternalError(fmt.Sprintf("Failed to scrape URL: %v", err))
	}

	// Determine source type based on URL and content
	sourceType := "web"
	if s.isPDFURL(targetURL) {
		sourceType = "pdf"
	}

	// Create the source
	req := CreateSourceRequest{
		ArticleID:  articleID,
		Title:      scraped.Title,
		Content:    scraped.Content,
		URL:        scraped.URL,
		SourceType: sourceType,
	}

	return s.CreateSource(ctx, req)
}

// GetSourcesByArticleID retrieves all sources for an article
func (s *ArticleSourceService) GetSourcesByArticleID(articleID uuid.UUID) ([]*models.Source, error) {
	db := s.db.GetDB()
	var sources []*models.Source

	err := db.Where("article_id = ?", articleID).
		Order("created_at DESC").
		Find(&sources).Error

	if err != nil {
		return nil, core.InternalError("Failed to get sources")
	}

	return sources, nil
}

// GetAllSources retrieves all sources with pagination and article metadata
func (s *ArticleSourceService) GetAllSources(page, limit int) ([]ArticleSourceWithArticle, int, error) {
	db := s.db.GetDB()

	// Default pagination values
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Count total sources
	var total int64
	if err := db.Model(&models.Source{}).Count(&total).Error; err != nil {
		return nil, 0, core.InternalError("Failed to count sources")
	}

	// Calculate total pages
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Query sources with article join
	var sources []ArticleSourceWithArticle
	err := db.Table("article_source").
		Select("article_source.*, article.title as article_title, article.slug as article_slug").
		Joins("LEFT JOIN article ON article.id = article_source.article_id").
		Order("article_source.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&sources).Error

	if err != nil {
		return nil, 0, core.InternalError("Failed to get sources")
	}

	return sources, totalPages, nil
}

// GetSource retrieves a specific source by ID
func (s *ArticleSourceService) GetSource(sourceID uuid.UUID) (*models.Source, error) {
	db := s.db.GetDB()
	var source models.Source

	err := db.First(&source, sourceID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Source")
		}
		return nil, core.InternalError("Failed to get source")
	}

	return &source, nil
}

// UpdateSource updates an existing source
func (s *ArticleSourceService) UpdateSource(ctx context.Context, sourceID uuid.UUID, req UpdateSourceRequest) (*models.Source, error) {
	db := s.db.GetDB()

	var source models.Source
	if err := db.First(&source, sourceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Source")
		}
		return nil, core.InternalError("Failed to fetch source")
	}

	// Track if we need to regenerate embedding
	needsEmbeddingUpdate := false

	// Update fields
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

	// Regenerate embedding if content changed
	if needsEmbeddingUpdate {
		embedding, err := s.embeddingService.GenerateEmbedding(ctx, source.Content)
		if err != nil {
			return nil, core.InternalError("Failed to generate embedding")
		}
		source.Embedding = embedding
	}

	if err := db.Save(&source).Error; err != nil {
		return nil, core.InternalError("Failed to update source")
	}

	return &source, nil
}

// DeleteSource deletes a source
func (s *ArticleSourceService) DeleteSource(sourceID uuid.UUID) error {
	db := s.db.GetDB()

	result := db.Delete(&models.Source{}, sourceID)
	if result.Error != nil {
		return core.InternalError("Failed to delete source")
	}

	if result.RowsAffected == 0 {
		return core.NotFoundError("Source")
	}

	return nil
}

// extractTextFromPDF extracts text content from PDF data
func (s *ArticleSourceService) extractTextFromPDF(pdfData []byte) (string, string, error) {
	reader := bytes.NewReader(pdfData)

	// Open the PDF
	pdfReader, err := pdf.NewReader(reader, int64(len(pdfData)))
	if err != nil {
		return "", "", fmt.Errorf("failed to open PDF: %w", err)
	}

	var textContent strings.Builder
	var title string

	// Extract text from all pages
	for i := 1; i <= pdfReader.NumPage(); i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		pageText, err := page.GetPlainText(nil)
		if err != nil {
			// Log the error but continue with other pages
			continue
		}

		// Use first non-empty line as title if we haven't found one yet
		if title == "" && pageText != "" {
			lines := strings.Split(pageText, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if len(line) > 0 && len(line) < 200 { // Reasonable title length
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

	// Clean up the title
	if title == "" {
		title = "PDF Document"
	}

	return title, content, nil
}

// isPDFURL checks if a URL likely points to a PDF file
func (s *ArticleSourceService) isPDFURL(targetURL string) bool {
	// Check file extension
	if strings.HasSuffix(strings.ToLower(targetURL), ".pdf") {
		return true
	}

	// Check if URL path contains PDF-related patterns
	lowerURL := strings.ToLower(targetURL)
	return strings.Contains(lowerURL, ".pdf") ||
		strings.Contains(lowerURL, "/pdf/") ||
		strings.Contains(lowerURL, "content-type=application/pdf")
}

// scrapeURL scrapes content from a web URL
func (s *ArticleSourceService) scrapeURL(targetURL string) (*ScrapedContent, error) {
	// Validate URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
		targetURL = parsedURL.String()
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make request
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; BlogAgent/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// Check content type to determine if it's a PDF
	contentType := resp.Header.Get("Content-Type")

	// Read the response body
	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle PDF content
	if strings.Contains(contentType, "application/pdf") ||
		(len(bodyData) > 4 && string(bodyData[:4]) == "%PDF") {

		title, content, err := s.extractTextFromPDF(bodyData)
		if err != nil {
			return nil, fmt.Errorf("failed to extract PDF content: %w", err)
		}

		return &ScrapedContent{
			Title:   title,
			Content: content,
			URL:     targetURL,
		}, nil
	}

	// Handle HTML content (existing logic)
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract title
	title := doc.Find("title").Text()
	if title == "" {
		title = doc.Find("h1").First().Text()
	}
	title = strings.TrimSpace(title)

	// Extract main content
	content := s.extractMainContent(doc)

	return &ScrapedContent{
		Title:   title,
		Content: content,
		URL:     targetURL,
	}, nil
}

// extractMainContent extracts the main content from an HTML document
func (s *ArticleSourceService) extractMainContent(doc *goquery.Document) string {
	var content strings.Builder

	// Try common content selectors in order of preference
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

	// Find the best content container
	for _, selector := range selectors {
		node := doc.Find(selector).First()
		if node.Length() > 0 {
			contentNode = node
			break
		}
	}

	// Fallback to body if no specific content container found
	if contentNode == nil {
		contentNode = doc.Find("body")
	}

	// Extract text from paragraphs, headings, and lists
	contentNode.Find("p, h1, h2, h3, h4, h5, h6, li").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" && len(text) > 10 { // Filter out very short text
			content.WriteString(text)
			content.WriteString("\n\n")
		}
	})

	// Clean up the content
	result := strings.TrimSpace(content.String())

	// Limit content length to avoid extremely long sources
	if len(result) > 5000 {
		result = result[:5000] + "..."
	}

	return result
}

// SearchSimilarSources finds sources similar to the given query using vector similarity
func (s *ArticleSourceService) SearchSimilarSources(ctx context.Context, articleID uuid.UUID, query string, limit int) ([]*models.Source, error) {
	// Generate embedding for the query
	queryEmbedding, err := s.embeddingService.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	db := s.db.GetDB()
	var sources []*models.Source

	// Use PostgreSQL's vector similarity search with pgvector
	err = db.Raw(`
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
