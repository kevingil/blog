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
	"backend/pkg/database/repository"
	"backend/pkg/types"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
)

// CreateRequest represents a request to create a source
type CreateRequest struct {
	ArticleID  uuid.UUID `json:"article_id" validate:"required"`
	Title      string    `json:"title"`
	Content    string    `json:"content" validate:"required"`
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

// ListResponse represents a paginated list of sources
type ListResponse struct {
	Sources    []SourceWithArticle `json:"sources"`
	TotalPages int                 `json:"total_pages"`
	Page       int                 `json:"page"`
}

// Service provides business logic for sources
type Service struct {
	sourceRepo       repository.SourceRepository
	articleRepo      repository.ArticleRepository
	embeddingService EmbeddingService
}

// NewService creates a new source service with the provided repositories
func NewService(sourceRepo repository.SourceRepository, articleRepo repository.ArticleRepository) *Service {
	return &Service{
		sourceRepo:       sourceRepo,
		articleRepo:      articleRepo,
		embeddingService: ml.NewEmbeddingService(),
	}
}

// GetByID retrieves a source by its ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*types.Source, error) {
	return s.sourceRepo.FindByID(ctx, id)
}

// GetByArticleID retrieves all sources for an article
func (s *Service) GetByArticleID(ctx context.Context, articleID uuid.UUID) ([]types.Source, error) {
	return s.sourceRepo.FindByArticleID(ctx, articleID)
}

// List retrieves all sources with pagination and article metadata
func (s *Service) List(ctx context.Context, page, limit int) (*ListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	opts := SourceListOptions{
		Page:    page,
		PerPage: limit,
	}

	sources, total, err := s.sourceRepo.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &ListResponse{
		Sources:    sources,
		TotalPages: totalPages,
		Page:       page,
	}, nil
}

// Create creates a new source with embedding generation
func (s *Service) Create(ctx context.Context, req CreateRequest) (*types.Source, error) {
	// Validate that the article exists
	_, err := s.articleRepo.FindByID(ctx, req.ArticleID)
	if err != nil {
		return nil, err
	}

	// Generate embedding for the content
	embedding, err := s.embeddingService.GenerateEmbedding(ctx, req.Content)
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

	source := &types.Source{
		ID:         uuid.New(),
		ArticleID:  req.ArticleID,
		Title:      req.Title,
		Content:    req.Content,
		URL:        req.URL,
		SourceType: sourceType,
		Embedding:  embedding.Slice(),
		CreatedAt:  time.Now(),
	}

	if err := s.sourceRepo.Save(ctx, source); err != nil {
		return nil, err
	}

	return source, nil
}

// ScrapeAndCreate scrapes content from a URL and creates a source
func (s *Service) ScrapeAndCreate(ctx context.Context, articleID uuid.UUID, targetURL string) (*types.Source, error) {
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

	return s.Create(ctx, req)
}

// Update updates an existing source
func (s *Service) Update(ctx context.Context, sourceID uuid.UUID, req UpdateRequest) (*types.Source, error) {
	source, err := s.sourceRepo.FindByID(ctx, sourceID)
	if err != nil {
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
		embedding, err := s.embeddingService.GenerateEmbedding(ctx, source.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding: %w", err)
		}
		source.Embedding = embedding.Slice()
	}

	if err := s.sourceRepo.Update(ctx, source); err != nil {
		return nil, err
	}

	return source, nil
}

// Delete removes a source by its ID
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.sourceRepo.Delete(ctx, id)
}

// SearchSimilar finds sources similar to the given query using vector similarity
func (s *Service) SearchSimilar(ctx context.Context, articleID uuid.UUID, query string, limit int) ([]types.Source, error) {
	queryEmbedding, err := s.embeddingService.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	sources, err := s.sourceRepo.SearchSimilar(ctx, articleID, queryEmbedding.Slice(), limit)
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
		return "", "", core.InvalidInputError("no text content found in PDF")
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
