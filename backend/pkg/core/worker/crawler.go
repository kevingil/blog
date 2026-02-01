package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"backend/pkg/core/datasource"
	"backend/pkg/core/insight"
	"backend/pkg/core/ml"
	"backend/pkg/database"
	"backend/pkg/database/repository"
	"backend/pkg/integrations/exa"
	"backend/pkg/types"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
)

// CrawlWorker crawls data sources and stores content
type CrawlWorker struct {
	logger            *slog.Logger
	exaClient         *exa.Client
	batchSize         int
	topicThreshold    float64
	embeddingService  *ml.EmbeddingService
	dataSourceService *datasource.Service
	insightService    *insight.Service
}

// NewCrawlWorker creates a new CrawlWorker instance
func NewCrawlWorker(logger *slog.Logger, exaAPIKey string) *CrawlWorker {
	if logger == nil {
		logger = slog.Default()
	}

	var exaClient *exa.Client
	if exaAPIKey != "" {
		exaClient = exa.NewClient(exaAPIKey)
	}

	// Initialize datasource service
	db := database.DB()
	dataSourceRepo := repository.NewDataSourceRepository(db)
	crawledContentRepo := repository.NewCrawledContentRepository(db)
	dataSourceService := datasource.NewService(dataSourceRepo, crawledContentRepo)

	// Initialize insight service
	insightService := insight.NewService(
		repository.NewInsightRepository(db),
		repository.NewInsightTopicRepository(db),
		repository.NewUserInsightStatusRepository(db),
		crawledContentRepo,
		repository.NewContentTopicMatchRepository(db),
		ml.NewEmbeddingService(),
	)

	return &CrawlWorker{
		logger:            logger,
		exaClient:         exaClient,
		batchSize:         10,
		topicThreshold:    0.6,
		embeddingService:  ml.NewEmbeddingService(),
		dataSourceService: dataSourceService,
		insightService:    insightService,
	}
}

// Name returns the worker name
func (w *CrawlWorker) Name() string {
	return "crawl"
}

// Run executes the crawl worker
func (w *CrawlWorker) Run(ctx context.Context) error {
	w.logger.Info("starting crawl worker run")
	statusService := GetStatusService()

	if w.exaClient == nil || !w.exaClient.IsConfigured() {
		w.logger.Warn("Exa client not configured, skipping crawl")
		statusService.UpdateStatus(w.Name(), StateRunning, 100, "Exa not configured, skipping")
		return nil
	}

	// Get data sources due for crawling
	statusService.UpdateStatus(w.Name(), StateRunning, 0, "Fetching sources to crawl...")
	sources, err := w.dataSourceService.GetDueToCrawl(ctx, w.batchSize)
	if err != nil {
		return fmt.Errorf("failed to get sources due for crawling: %w", err)
	}

	if len(sources) == 0 {
		w.logger.Info("no data sources due for crawling")
		statusService.UpdateStatus(w.Name(), StateRunning, 100, "No sources due for crawling")
		return nil
	}

	w.logger.Info("found data sources to crawl", "count", len(sources))
	statusService.SetProgress(w.Name(), 0, len(sources), fmt.Sprintf("Found %d sources to crawl", len(sources)))

	for i, source := range sources {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			statusService.SetProgress(w.Name(), i, len(sources), fmt.Sprintf("Crawling: %s", source.Name))

			if err := w.crawlSource(ctx, &source); err != nil {
				w.logger.Error("failed to crawl source", "id", source.ID, "url", source.URL, "error", err)
				errMsg := err.Error()
				_ = w.dataSourceService.UpdateCrawlStatus(ctx, source.ID, "failed", &errMsg)
			} else {
				_ = w.dataSourceService.UpdateCrawlStatus(ctx, source.ID, "success", nil)
				_ = w.dataSourceService.SetNextCrawlTime(ctx, source.ID, source.CrawlFrequency)
			}
		}
	}

	statusService.SetProgress(w.Name(), len(sources), len(sources), fmt.Sprintf("Completed crawling %d sources", len(sources)))
	return nil
}

// crawlSource crawls a single data source
func (w *CrawlWorker) crawlSource(ctx context.Context, source *types.DataSource) error {
	w.logger.Info("crawling source", "id", source.ID, "name", source.Name, "url", source.URL)

	// Update status to crawling
	_ = w.dataSourceService.UpdateCrawlStatus(ctx, source.ID, "crawling", nil)

	// Determine crawl strategy based on source type
	var contents []crawledItem
	var err error

	switch source.SourceType {
	case "rss", "newsletter":
		contents, err = w.crawlRSS(ctx, source)
	default:
		contents, err = w.crawlWebsite(ctx, source)
	}

	if err != nil {
		return err
	}

	if len(contents) == 0 {
		w.logger.Info("no new content found", "source_id", source.ID)
		return nil
	}

	w.logger.Info("found content items", "source_id", source.ID, "count", len(contents))

	// Process and store each content item
	contentRepo := repository.NewCrawledContentRepository(database.DB())
	newContentCount := 0

	for _, item := range contents {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := w.processContent(ctx, contentRepo, source, &item); err != nil {
				w.logger.Error("failed to process content", "url", item.URL, "error", err)
				continue
			}
			newContentCount++
		}
	}

	// Update content count
	if newContentCount > 0 {
		dataSourceRepo := repository.NewDataSourceRepository(database.DB())
		_ = dataSourceRepo.IncrementContentCount(ctx, source.ID, newContentCount)
	}

	w.logger.Info("crawl completed", "source_id", source.ID, "new_content", newContentCount)
	return nil
}

// crawledItem represents a single piece of crawled content
type crawledItem struct {
	URL         string
	Title       string
	Content     string
	Author      string
	PublishedAt *time.Time
}

// crawlWebsite crawls a website and extracts content using Exa
func (w *CrawlWorker) crawlWebsite(ctx context.Context, source *types.DataSource) ([]crawledItem, error) {
	// First, check if this is a PDF by doing a quick HEAD request
	isPDF, err := w.checkIfPDF(source.URL)
	if err == nil && isPDF {
		// Fall back to HTTP fetch for PDFs since Exa doesn't handle them
		return w.crawlPDF(ctx, source.URL)
	}

	// Extract domain from source URL for Exa search
	domain := extractDomainForExa(source.URL)
	if domain == "" {
		return nil, fmt.Errorf("failed to extract domain from URL: %s", source.URL)
	}

	// Use Exa to search for content from this domain
	// Search for recent content from the specific domain
	searchQuery := fmt.Sprintf("site:%s", domain)

	results, err := w.exaClient.Search(ctx, searchQuery, &exa.SearchOptions{
		NumResults:     20, // Get up to 20 articles per crawl
		IncludeDomains: []string{domain},
		IncludeText:    true,
		IncludeSummary: true,
	})
	if err != nil {
		w.logger.Warn("Exa search failed, falling back to HTTP crawl", "error", err)
		return w.crawlWebsiteFallback(ctx, source)
	}

	if len(results.Results) == 0 {
		w.logger.Info("no results from Exa, trying fallback", "source_id", source.ID)
		return w.crawlWebsiteFallback(ctx, source)
	}

	// Convert Exa results to crawledItems
	var items []crawledItem
	for _, result := range results.Results {
		// Skip if no text content
		if result.Text == "" {
			continue
		}

		// Parse published date if available
		var publishedAt *time.Time
		if result.PublishedDate != "" {
			// Exa returns dates in ISO format
			if t, err := time.Parse("2006-01-02", result.PublishedDate); err == nil {
				publishedAt = &t
			} else if t, err := time.Parse(time.RFC3339, result.PublishedDate); err == nil {
				publishedAt = &t
			}
		}

		items = append(items, crawledItem{
			URL:         result.URL,
			Title:       result.Title,
			Content:     result.Text,
			Author:      result.Author,
			PublishedAt: publishedAt,
		})
	}

	return items, nil
}

// crawlPDF handles PDF crawling via HTTP fetch
func (w *CrawlWorker) crawlPDF(ctx context.Context, pdfURL string) ([]crawledItem, error) {
	resp, err := w.fetchURL(pdfURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PDF: %w", err)
	}
	defer resp.Body.Close()

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF response: %w", err)
	}

	title, content, err := w.extractTextFromPDF(bodyData)
	if err != nil {
		return nil, err
	}

	return []crawledItem{{
		URL:     pdfURL,
		Title:   title,
		Content: content,
	}}, nil
}

// checkIfPDF does a HEAD request to check if URL is a PDF
func (w *CrawlWorker) checkIfPDF(targetURL string) (bool, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("HEAD", targetURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; BlogAgent/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	return strings.Contains(contentType, "application/pdf"), nil
}

// extractDomainForExa extracts the domain from a URL for Exa search
func extractDomainForExa(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}

// crawlWebsiteFallback uses traditional HTTP crawling when Exa fails
func (w *CrawlWorker) crawlWebsiteFallback(ctx context.Context, source *types.DataSource) ([]crawledItem, error) {
	// Fetch the main page
	resp, err := w.fetchURL(source.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check if it's a PDF
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/pdf") || (len(bodyData) > 4 && string(bodyData[:4]) == "%PDF") {
		title, content, err := w.extractTextFromPDF(bodyData)
		if err != nil {
			return nil, err
		}
		return []crawledItem{{
			URL:     source.URL,
			Title:   title,
			Content: content,
		}}, nil
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var items []crawledItem

	// Try to find article links
	articleLinks := w.findArticleLinks(doc, source.URL)

	if len(articleLinks) > 0 {
		// Crawl individual articles
		for _, link := range articleLinks {
			if len(items) >= 20 { // Limit to 20 articles per crawl
				break
			}

			item, err := w.crawlArticle(ctx, link)
			if err != nil {
				w.logger.Warn("failed to crawl article", "url", link, "error", err)
				continue
			}
			items = append(items, *item)
		}
	} else {
		// Extract content from the main page
		title := doc.Find("title").Text()
		if title == "" {
			title = doc.Find("h1").First().Text()
		}

		content := w.extractMainContent(doc)
		if content != "" {
			items = append(items, crawledItem{
				URL:     source.URL,
				Title:   strings.TrimSpace(title),
				Content: content,
			})
		}
	}

	return items, nil
}

// crawlRSS crawls an RSS feed
func (w *CrawlWorker) crawlRSS(ctx context.Context, source *types.DataSource) ([]crawledItem, error) {
	feedURL := source.FeedURL
	if feedURL == nil || *feedURL == "" {
		// Try to discover RSS feed
		discovered := w.discoverRSSFeed(source.URL)
		if discovered == "" {
			return nil, fmt.Errorf("no RSS feed found")
		}
		feedURL = &discovered
	}

	// For now, treat RSS as regular website crawl
	// TODO: Implement proper RSS parsing
	return w.crawlWebsite(ctx, source)
}

// crawlArticle crawls a single article page
func (w *CrawlWorker) crawlArticle(ctx context.Context, articleURL string) (*crawledItem, error) {
	resp, err := w.fetchURL(articleURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	title := doc.Find("title").Text()
	if title == "" {
		title = doc.Find("h1").First().Text()
	}

	content := w.extractMainContent(doc)
	if content == "" {
		return nil, fmt.Errorf("no content found")
	}

	// Try to extract author
	author := ""
	doc.Find("[rel='author'], .author, .byline").Each(func(i int, s *goquery.Selection) {
		if author == "" {
			author = strings.TrimSpace(s.Text())
		}
	})

	// Try to extract publish date
	var publishedAt *time.Time
	doc.Find("time[datetime], [property='article:published_time']").Each(func(i int, s *goquery.Selection) {
		if publishedAt == nil {
			if datetime, exists := s.Attr("datetime"); exists {
				if t, err := time.Parse(time.RFC3339, datetime); err == nil {
					publishedAt = &t
				}
			}
			if content, exists := s.Attr("content"); exists {
				if t, err := time.Parse(time.RFC3339, content); err == nil {
					publishedAt = &t
				}
			}
		}
	})

	return &crawledItem{
		URL:         articleURL,
		Title:       strings.TrimSpace(title),
		Content:     content,
		Author:      author,
		PublishedAt: publishedAt,
	}, nil
}

// processContent processes a single content item (embed and match to topics)
func (w *CrawlWorker) processContent(ctx context.Context, repo *repository.CrawledContentRepository, source *types.DataSource, item *crawledItem) error {
	// Check if content already exists
	existing, err := repo.FindByURL(ctx, source.ID, item.URL)
	if err == nil && existing != nil {
		// Content already exists, skip
		return nil
	}

	// Truncate content for embedding (max ~8000 chars)
	contentForEmbedding := item.Content
	if len(contentForEmbedding) > 8000 {
		contentForEmbedding = contentForEmbedding[:8000]
	}

	// Generate embedding
	embedding, err := w.embeddingService.GenerateEmbedding(ctx, contentForEmbedding)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Create content record
	content := &types.CrawledContent{
		ID:           uuid.New(),
		DataSourceID: source.ID,
		URL:          item.URL,
		Title:        &item.Title,
		Content:      item.Content,
		Author:       &item.Author,
		PublishedAt:  item.PublishedAt,
		Embedding:    embedding.Slice(),
	}

	if err := repo.Save(ctx, content); err != nil {
		return fmt.Errorf("failed to save content: %w", err)
	}

	// Match content to topics
	_, err = w.insightService.MatchContentToTopics(ctx, content.ID, embedding.Slice(), w.topicThreshold)
	if err != nil {
		w.logger.Warn("failed to match content to topics", "content_id", content.ID, "error", err)
	}

	return nil
}

// Helper methods

func (w *CrawlWorker) fetchURL(targetURL string) (*http.Response, error) {
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
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; BlogAgent/1.0)")

	return client.Do(req)
}

func (w *CrawlWorker) extractMainContent(doc *goquery.Document) string {
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

	if len(result) > 10000 {
		result = result[:10000] + "..."
	}

	return result
}

func (w *CrawlWorker) findArticleLinks(doc *goquery.Document, baseURL string) []string {
	var links []string
	seen := make(map[string]bool)

	parsedBase, _ := url.Parse(baseURL)

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		// Skip common non-article links
		if strings.HasPrefix(href, "#") ||
			strings.Contains(href, "javascript:") ||
			strings.Contains(href, "mailto:") ||
			strings.HasSuffix(href, ".css") ||
			strings.HasSuffix(href, ".js") ||
			strings.HasSuffix(href, ".png") ||
			strings.HasSuffix(href, ".jpg") {
			return
		}

		// Resolve relative URLs
		parsedHref, err := url.Parse(href)
		if err != nil {
			return
		}

		resolvedURL := parsedBase.ResolveReference(parsedHref).String()

		// Only include links from same domain
		resolvedParsed, _ := url.Parse(resolvedURL)
		if resolvedParsed.Host != parsedBase.Host {
			return
		}

		// Skip if already seen
		if seen[resolvedURL] {
			return
		}
		seen[resolvedURL] = true

		// Check if it looks like an article URL
		if w.isLikelyArticleURL(resolvedURL) {
			links = append(links, resolvedURL)
		}
	})

	return links
}

func (w *CrawlWorker) isLikelyArticleURL(urlStr string) bool {
	// Common patterns for blog article URLs
	patterns := []string{
		"/blog/",
		"/post/",
		"/article/",
		"/news/",
		"/20", // Date patterns like /2024/, /2023/
	}

	for _, pattern := range patterns {
		if strings.Contains(urlStr, pattern) {
			return true
		}
	}

	return false
}

func (w *CrawlWorker) discoverRSSFeed(websiteURL string) string {
	resp, err := w.fetchURL(websiteURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return ""
	}

	// Look for RSS/Atom link tags
	var feedURL string
	doc.Find("link[type='application/rss+xml'], link[type='application/atom+xml']").Each(func(i int, s *goquery.Selection) {
		if feedURL == "" {
			if href, exists := s.Attr("href"); exists {
				parsedBase, _ := url.Parse(websiteURL)
				parsedHref, _ := url.Parse(href)
				feedURL = parsedBase.ResolveReference(parsedHref).String()
			}
		}
	})

	return feedURL
}

func (w *CrawlWorker) extractTextFromPDF(pdfData []byte) (string, string, error) {
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
