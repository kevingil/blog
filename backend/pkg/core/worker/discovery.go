package worker

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"backend/pkg/core/datasource"
	"backend/pkg/database"
	"backend/pkg/database/repository"
	"backend/pkg/integrations/exa"
	"backend/pkg/types"
)

// DiscoveryWorker discovers similar websites using Exa
type DiscoveryWorker struct {
	logger           *slog.Logger
	exaClient        *exa.Client
	maxDiscoveries   int
	maxSourcesPerRun int
}

// NewDiscoveryWorker creates a new DiscoveryWorker instance
func NewDiscoveryWorker(logger *slog.Logger, exaAPIKey string) *DiscoveryWorker {
	if logger == nil {
		logger = slog.Default()
	}

	var client *exa.Client
	if exaAPIKey != "" {
		client = exa.NewClient(exaAPIKey)
	}

	return &DiscoveryWorker{
		logger:           logger,
		exaClient:        client,
		maxDiscoveries:   5, // Max new sources to discover per existing source
		maxSourcesPerRun: 5, // Max sources to process per run
	}
}

// Name returns the worker name
func (w *DiscoveryWorker) Name() string {
	return "discovery"
}

// Run executes the discovery worker
func (w *DiscoveryWorker) Run(ctx context.Context) error {
	w.logger.Info("starting discovery worker run")
	statusService := GetStatusService()

	if w.exaClient == nil {
		w.logger.Warn("Exa client not configured, skipping discovery")
		statusService.UpdateStatus(w.Name(), StateRunning, 100, "Exa not configured, skipping")
		return nil
	}

	// Get active data sources (not discovered ones, as they're already derived)
	statusService.UpdateStatus(w.Name(), StateRunning, 0, "Fetching data sources...")
	repo := repository.NewDataSourceRepository(database.DB())
	sources, _, err := repo.List(ctx, 0, 100)
	if err != nil {
		return fmt.Errorf("failed to get data sources: %w", err)
	}

	// Filter to only user-added sources
	var activeManualSources []types.DataSource
	for _, s := range sources {
		if s.IsEnabled && !s.IsDiscovered {
			activeManualSources = append(activeManualSources, s)
		}
	}

	if len(activeManualSources) == 0 {
		w.logger.Info("no active manual data sources found")
		statusService.UpdateStatus(w.Name(), StateRunning, 100, "No manual sources to discover from")
		return nil
	}

	// Limit sources to process
	if len(activeManualSources) > w.maxSourcesPerRun {
		activeManualSources = activeManualSources[:w.maxSourcesPerRun]
	}

	w.logger.Info("discovering similar websites", "source_count", len(activeManualSources))
	statusService.SetProgress(w.Name(), 0, len(activeManualSources), fmt.Sprintf("Processing %d sources", len(activeManualSources)))

	totalDiscovered := 0
	for i, source := range activeManualSources {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			statusService.SetProgress(w.Name(), i, len(activeManualSources), fmt.Sprintf("Finding similar sites for: %s", source.Name))
			
			discovered, err := w.discoverSimilarSites(ctx, &source)
			if err != nil {
				w.logger.Error("failed to discover similar sites", "source_id", source.ID, "error", err)
				continue
			}
			totalDiscovered += discovered
		}
	}

	w.logger.Info("discovery worker completed", "total_discovered", totalDiscovered)
	statusService.SetProgress(w.Name(), len(activeManualSources), len(activeManualSources), fmt.Sprintf("Discovered %d new sites", totalDiscovered))
	return nil
}

// discoverSimilarSites discovers similar websites for a given source
func (w *DiscoveryWorker) discoverSimilarSites(ctx context.Context, source *types.DataSource) (int, error) {
	w.logger.Info("discovering similar sites", "source_id", source.ID, "source_url", source.URL)

	// Use Exa to find similar sites
	results, err := w.exaClient.FindSimilar(ctx, source.URL, &exa.FindSimilarOptions{
		NumResults:        w.maxDiscoveries + 5, // Get extra to account for filtering
		ExcludeSourceDomain: true,
	})
	if err != nil {
		return 0, fmt.Errorf("Exa findSimilar failed: %w", err)
	}

	if len(results.Results) == 0 {
		w.logger.Info("no similar sites found", "source_id", source.ID)
		return 0, nil
	}

	discoveredCount := 0
	for _, result := range results.Results {
		if discoveredCount >= w.maxDiscoveries {
			break
		}

		// Skip if URL looks invalid
		if result.URL == "" || !strings.HasPrefix(result.URL, "http") {
			continue
		}

		// Normalize URL (remove trailing slashes, etc.)
		normalizedURL := normalizeURL(result.URL)
		if normalizedURL == "" {
			continue
		}

		// Skip if it's too similar to source (same domain root)
		if isSameDomainRoot(source.URL, normalizedURL) {
			continue
		}

		// Create discovered data source
		name := result.Title
		if name == "" {
			name = extractDomainName(normalizedURL)
		}

		// Pass both orgID and userID from the source
		_, err := datasource.CreateDiscoveredSource(ctx, source.OrganizationID, source.UserID, source.ID, name, normalizedURL)
		if err != nil {
			// Skip if already exists
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			w.logger.Warn("failed to create discovered source", "url", normalizedURL, "error", err)
			continue
		}

		discoveredCount++
		w.logger.Info("discovered similar site", "url", normalizedURL, "name", name, "from_source", source.ID)
	}

	return discoveredCount, nil
}

// Helper functions

func normalizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	// Remove trailing slash
	parsed.Path = strings.TrimSuffix(parsed.Path, "/")

	// Remove fragment
	parsed.Fragment = ""

	// Remove common tracking parameters
	query := parsed.Query()
	for key := range query {
		lowerKey := strings.ToLower(key)
		if strings.HasPrefix(lowerKey, "utm_") ||
			lowerKey == "ref" ||
			lowerKey == "source" ||
			lowerKey == "campaign" {
			query.Del(key)
		}
	}
	parsed.RawQuery = query.Encode()

	return parsed.String()
}

func isSameDomainRoot(url1, url2 string) bool {
	root1 := extractDomainRoot(url1)
	root2 := extractDomainRoot(url2)
	return root1 != "" && root2 != "" && root1 == root2
}

func extractDomainRoot(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	host := strings.ToLower(parsed.Host)

	// Remove www prefix
	host = strings.TrimPrefix(host, "www.")

	// Extract domain root (e.g., "example.com" from "blog.example.com")
	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "." + parts[len(parts)-1]
	}

	return host
}

func extractDomainName(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	host := strings.TrimPrefix(parsed.Host, "www.")

	// Try to create a readable name from domain
	parts := strings.Split(host, ".")
	if len(parts) > 0 {
		// Capitalize first part
		name := parts[0]
		if len(name) > 0 {
			return strings.ToUpper(string(name[0])) + name[1:]
		}
	}

	return host
}
