package services

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"blog-agent-go/backend/internal/database"
	"blog-agent-go/backend/internal/models"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ArticleSourceService struct {
	db database.Service
}

func NewArticleSourceService(db database.Service) *ArticleSourceService {
	return &ArticleSourceService{
		db: db,
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

// CreateSource creates a new article source with embedding generation
func (s *ArticleSourceService) CreateSource(ctx context.Context, req CreateSourceRequest) (*models.ArticleSource, error) {
	db := s.db.GetDB()

	// Validate that the article exists
	var article models.Article
	if err := db.First(&article, req.ArticleID).Error; err != nil {
		return nil, fmt.Errorf("article not found: %w", err)
	}

	// Generate embedding for the content
	embedding, err := s.generateEmbedding(ctx, req.Content)
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

	source := &models.ArticleSource{
		ArticleID:  req.ArticleID,
		Title:      req.Title,
		Content:    req.Content,
		URL:        req.URL,
		SourceType: sourceType,
		Embedding:  embedding,
	}

	if err := db.Create(source).Error; err != nil {
		return nil, fmt.Errorf("failed to create source: %w", err)
	}

	return source, nil
}

// ScrapeAndCreateSource scrapes content from a URL and creates a source
func (s *ArticleSourceService) ScrapeAndCreateSource(ctx context.Context, articleID uuid.UUID, targetURL string) (*models.ArticleSource, error) {
	// Scrape the content
	scraped, err := s.scrapeURL(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape URL: %w", err)
	}

	// Create the source
	req := CreateSourceRequest{
		ArticleID:  articleID,
		Title:      scraped.Title,
		Content:    scraped.Content,
		URL:        scraped.URL,
		SourceType: "web",
	}

	return s.CreateSource(ctx, req)
}

// GetSourcesByArticleID retrieves all sources for an article
func (s *ArticleSourceService) GetSourcesByArticleID(articleID uuid.UUID) ([]*models.ArticleSource, error) {
	db := s.db.GetDB()
	var sources []*models.ArticleSource

	err := db.Where("article_id = ?", articleID).
		Order("created_at DESC").
		Find(&sources).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get sources: %w", err)
	}

	return sources, nil
}

// GetSource retrieves a specific source by ID
func (s *ArticleSourceService) GetSource(sourceID uuid.UUID) (*models.ArticleSource, error) {
	db := s.db.GetDB()
	var source models.ArticleSource

	err := db.First(&source, sourceID).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get source: %w", err)
	}

	return &source, nil
}

// UpdateSource updates an existing source
func (s *ArticleSourceService) UpdateSource(ctx context.Context, sourceID uuid.UUID, req UpdateSourceRequest) (*models.ArticleSource, error) {
	db := s.db.GetDB()

	var source models.ArticleSource
	if err := db.First(&source, sourceID).Error; err != nil {
		return nil, fmt.Errorf("source not found: %w", err)
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
		embedding, err := s.generateEmbedding(ctx, source.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding: %w", err)
		}
		source.Embedding = embedding
	}

	if err := db.Save(&source).Error; err != nil {
		return nil, fmt.Errorf("failed to update source: %w", err)
	}

	return &source, nil
}

// DeleteSource deletes a source
func (s *ArticleSourceService) DeleteSource(sourceID uuid.UUID) error {
	db := s.db.GetDB()

	result := db.Delete(&models.ArticleSource{}, sourceID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete source: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
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

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
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

// generateEmbedding generates an embedding vector for the given text
func (s *ArticleSourceService) generateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// TODO: Implement OpenAI embeddings integration
	// For now, return a dummy embedding to get the basic functionality working
	// This should be replaced with actual OpenAI embeddings API call

	// Return a dummy embedding of the expected size (1536 dimensions)
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = 0.1 // Dummy value
	}

	return embedding, nil
}

// SearchSimilarSources finds sources similar to the given query using vector similarity
func (s *ArticleSourceService) SearchSimilarSources(ctx context.Context, articleID uuid.UUID, query string, limit int) ([]*models.ArticleSource, error) {
	// Generate embedding for the query
	queryEmbedding, err := s.generateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	db := s.db.GetDB()
	var sources []*models.ArticleSource

	// Use PostgreSQL's vector similarity search
	// Note: This requires the pgvector extension
	err = db.Raw(`
		SELECT * FROM article_source 
		WHERE article_id = ? 
		ORDER BY embedding <-> ? 
		LIMIT ?`,
		articleID, queryEmbedding, limit).
		Scan(&sources).Error

	if err != nil {
		return nil, fmt.Errorf("failed to search similar sources: %w", err)
	}

	return sources, nil
}
