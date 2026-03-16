package datasource

import (
	"context"
	"net/url"
	"sort"
	"strings"

	"backend/pkg/api/dto"
	"backend/pkg/core"
	"backend/pkg/database/repository"
	"backend/pkg/integrations/exa"
	"backend/pkg/types"

	"github.com/google/uuid"
)

const (
	defaultRecommendationLimit = 8
	maxRecommendationLimit     = 12
	maxDiscoverySeedSources    = 5
	maxDiscoveryResultsPerSeed = 6
)

// RecommendationSearchService defines the search provider used for source recommendations.
type RecommendationSearchService interface {
	Search(ctx context.Context, query string, options *exa.SearchOptions) (*exa.SearchResponse, error)
	FindSimilar(ctx context.Context, url string, options *exa.FindSimilarOptions) (*exa.SearchResponse, error)
	IsConfigured() bool
}

// RecommendationService recommends data sources from a freeform query without persisting them.
type RecommendationService struct {
	dataSourceRepo repository.DataSourceRepository
	searchService  RecommendationSearchService
}

// NewRecommendationService creates a recommendation service.
func NewRecommendationService(dataSourceRepo repository.DataSourceRepository, searchService RecommendationSearchService) *RecommendationService {
	return &RecommendationService{
		dataSourceRepo: dataSourceRepo,
		searchService:  searchService,
	}
}

// Recommend returns ephemeral source recommendations for the given owner.
func (s *RecommendationService) Recommend(ctx context.Context, orgID *uuid.UUID, userID *uuid.UUID, req dto.DataSourceRecommendationRequest) (*dto.DataSourceRecommendationsResponse, error) {
	if orgID == nil && userID == nil {
		return nil, core.InvalidInputError("Either organization_id or user_id must be provided")
	}
	if s.searchService == nil || !s.searchService.IsConfigured() {
		return nil, core.ExternalError("Source recommendations are unavailable because Exa is not configured")
	}

	limit := normalizeRecommendationLimit(req.Limit)
	existingSources, err := s.listExistingSources(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}

	searchResp, err := s.searchService.Search(ctx, req.Query, &exa.SearchOptions{
		NumResults:        limit * 3,
		UseAutoprompt:     true,
		IncludeText:       true,
		IncludeHighlights: true,
		IncludeSummary:    true,
	})
	if err != nil {
		return nil, core.ExternalError("Failed to fetch source recommendations")
	}

	recommendations := s.buildRecommendations(searchResp.Results, existingSources, limit)
	return &dto.DataSourceRecommendationsResponse{
		Mode:            "query",
		Query:           strings.TrimSpace(req.Query),
		Recommendations: recommendations,
	}, nil
}

// RecommendFromExistingSources returns ephemeral recommendations based on current manual sources.
func (s *RecommendationService) RecommendFromExistingSources(ctx context.Context, orgID *uuid.UUID, userID *uuid.UUID, req dto.DataSourceDiscoveryRecommendationRequest) (*dto.DataSourceRecommendationsResponse, error) {
	if orgID == nil && userID == nil {
		return nil, core.InvalidInputError("Either organization_id or user_id must be provided")
	}
	if s.searchService == nil || !s.searchService.IsConfigured() {
		return nil, core.ExternalError("Source discovery is unavailable because Exa is not configured")
	}

	limit := normalizeRecommendationLimit(req.Limit)
	existingSources, err := s.listExistingSources(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}

	seedSources := filterDiscoverySeedSources(existingSources)
	if len(seedSources) == 0 {
		return &dto.DataSourceRecommendationsResponse{
			Mode:            "discovery",
			Recommendations: []dto.DataSourceRecommendationResponse{},
		}, nil
	}
	if len(seedSources) > maxDiscoverySeedSources {
		seedSources = seedSources[:maxDiscoverySeedSources]
	}

	candidates := make([]discoveryCandidate, 0, len(seedSources)*maxDiscoveryResultsPerSeed)
	failedLookups := 0
	for _, seed := range seedSources {
		resp, findErr := s.searchService.FindSimilar(ctx, seed.URL, &exa.FindSimilarOptions{
			NumResults:          maxDiscoveryResultsPerSeed,
			ExcludeSourceDomain: true,
			IncludeHighlights:   true,
			IncludeSummary:      true,
			IncludeText:         true,
		})
		if findErr != nil {
			failedLookups++
			continue
		}

		for _, result := range resp.Results {
			candidates = append(candidates, discoveryCandidate{
				Result:   result,
				SeedName: seed.Name,
			})
		}
	}

	if len(candidates) == 0 && failedLookups > 0 {
		return nil, core.ExternalError("Failed to fetch adjacent source recommendations")
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Result.Score > candidates[j].Result.Score
	})

	recommendations := s.buildDiscoveryRecommendations(candidates, existingSources, seedSources, limit)
	return &dto.DataSourceRecommendationsResponse{
		Mode:            "discovery",
		SeedCount:       len(seedSources),
		Recommendations: recommendations,
	}, nil
}

func (s *RecommendationService) listExistingSources(ctx context.Context, orgID *uuid.UUID, userID *uuid.UUID) ([]types.DataSource, error) {
	if orgID != nil {
		return s.dataSourceRepo.FindByOrganizationID(ctx, *orgID)
	}
	return s.dataSourceRepo.FindByUserID(ctx, *userID)
}

func (s *RecommendationService) buildRecommendations(results []exa.SearchResult, existingSources []types.DataSource, limit int) []dto.DataSourceRecommendationResponse {
	existingURLs := make(map[string]struct{}, len(existingSources))
	existingDomains := make(map[string]struct{}, len(existingSources))
	for _, source := range existingSources {
		normalized, domain := normalizeRecommendationURL(source.URL)
		if normalized != "" {
			existingURLs[normalized] = struct{}{}
		}
		if domain != "" {
			existingDomains[domain] = struct{}{}
		}
	}

	recommendations := make([]dto.DataSourceRecommendationResponse, 0, limit)
	seenDomains := make(map[string]struct{}, limit)

	for _, result := range results {
		normalizedURL, domain := normalizeRecommendationURL(result.URL)
		if normalizedURL == "" || domain == "" {
			continue
		}

		if _, exists := existingURLs[normalizedURL]; exists {
			continue
		}
		if _, exists := existingDomains[domain]; exists {
			continue
		}
		if _, exists := seenDomains[domain]; exists {
			continue
		}

		recommendations = append(recommendations, dto.DataSourceRecommendationResponse{
			Name:        humanizeDomain(domain),
			URL:         normalizedURL,
			Domain:      domain,
			Summary:     summarizeResult(result),
			Reason:      reasonFromResult(result),
			SourceType:  inferRecommendationSourceType(result),
			Score:       result.Score,
			Favicon:     result.Favicon,
			SampleURL:   result.URL,
			SampleTitle: result.Title,
		})
		seenDomains[domain] = struct{}{}

		if len(recommendations) >= limit {
			break
		}
	}

	return recommendations
}

type discoveryCandidate struct {
	Result   exa.SearchResult
	SeedName string
}

func (s *RecommendationService) buildDiscoveryRecommendations(candidates []discoveryCandidate, existingSources []types.DataSource, seedSources []types.DataSource, limit int) []dto.DataSourceRecommendationResponse {
	existingURLs := make(map[string]struct{}, len(existingSources))
	existingRoots := make(map[string]struct{}, len(existingSources))
	for _, source := range existingSources {
		normalized, domain := normalizeRecommendationURL(source.URL)
		if normalized != "" {
			existingURLs[normalized] = struct{}{}
		}
		if root := extractRecommendationDomainRoot(domain); root != "" {
			existingRoots[root] = struct{}{}
		}
	}

	seedRoots := make(map[string]struct{}, len(seedSources))
	for _, source := range seedSources {
		_, domain := normalizeRecommendationURL(source.URL)
		if root := extractRecommendationDomainRoot(domain); root != "" {
			seedRoots[root] = struct{}{}
		}
	}

	recommendations := make([]dto.DataSourceRecommendationResponse, 0, limit)
	seenRoots := make(map[string]struct{}, limit)

	for _, candidate := range candidates {
		normalizedURL, domain := normalizeRecommendationURL(candidate.Result.URL)
		if normalizedURL == "" || domain == "" {
			continue
		}

		root := extractRecommendationDomainRoot(domain)
		if root == "" {
			continue
		}

		if _, exists := existingURLs[normalizedURL]; exists {
			continue
		}
		if _, exists := existingRoots[root]; exists {
			continue
		}
		if _, exists := seedRoots[root]; exists {
			continue
		}
		if _, exists := seenRoots[root]; exists {
			continue
		}

		recommendations = append(recommendations, dto.DataSourceRecommendationResponse{
			Name:        humanizeDomain(domain),
			URL:         normalizedURL,
			Domain:      domain,
			Summary:     summarizeResult(candidate.Result),
			Reason:      discoveryReasonFromCandidate(candidate),
			SourceType:  inferRecommendationSourceType(candidate.Result),
			Score:       candidate.Result.Score,
			Favicon:     candidate.Result.Favicon,
			SampleURL:   candidate.Result.URL,
			SampleTitle: candidate.Result.Title,
		})
		seenRoots[root] = struct{}{}

		if len(recommendations) >= limit {
			break
		}
	}

	return recommendations
}

func normalizeRecommendationLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultRecommendationLimit
	case limit > maxRecommendationLimit:
		return maxRecommendationLimit
	default:
		return limit
	}
}

func normalizeRecommendationURL(rawURL string) (string, string) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", ""
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", ""
	}

	normalized := parsed.Scheme + "://" + host
	return normalized, host
}

func summarizeResult(result exa.SearchResult) string {
	if summary := strings.TrimSpace(result.Summary); summary != "" {
		return summary
	}
	if len(result.Highlights) > 0 {
		return strings.TrimSpace(result.Highlights[0])
	}
	if text := strings.TrimSpace(result.Text); text != "" {
		if len(text) > 220 {
			return strings.TrimSpace(text[:220]) + "..."
		}
		return text
	}
	return ""
}

func reasonFromResult(result exa.SearchResult) string {
	parts := make([]string, 0, 3)
	if result.Title != "" {
		parts = append(parts, result.Title)
	}
	if len(result.Highlights) > 0 {
		parts = append(parts, strings.TrimSpace(result.Highlights[0]))
	}
	if len(parts) == 0 {
		return "Relevant source from AI search results"
	}
	return strings.Join(parts, " - ")
}

func inferRecommendationSourceType(result exa.SearchResult) string {
	combined := strings.ToLower(strings.Join([]string{result.URL, result.Title, result.Summary}, " "))

	switch {
	case strings.Contains(combined, "substack"), strings.Contains(combined, "newsletter"):
		return "newsletter"
	case strings.Contains(combined, "/feed"), strings.Contains(combined, " rss"), strings.Contains(combined, "rss "):
		return "rss"
	case strings.Contains(combined, "forum"), strings.Contains(combined, "community"), strings.Contains(combined, "discuss"), strings.Contains(combined, "reddit"), strings.Contains(combined, "news.ycombinator.com"):
		return "forum"
	case strings.Contains(combined, "news"), strings.Contains(combined, "press"):
		return "news"
	default:
		return "blog"
	}
}

func humanizeDomain(domain string) string {
	base := strings.TrimPrefix(domain, "www.")
	parts := strings.Split(base, ".")
	if len(parts) == 0 || parts[0] == "" {
		return domain
	}

	nameParts := strings.FieldsFunc(parts[0], func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, part := range nameParts {
		if part == "" {
			continue
		}
		nameParts[i] = strings.ToUpper(part[:1]) + part[1:]
	}

	if len(nameParts) == 0 {
		return domain
	}
	return strings.Join(nameParts, " ")
}

func filterDiscoverySeedSources(sources []types.DataSource) []types.DataSource {
	seeds := make([]types.DataSource, 0, len(sources))
	for _, source := range sources {
		if source.IsEnabled && !source.IsDiscovered {
			seeds = append(seeds, source)
		}
	}
	return seeds
}

func extractRecommendationDomainRoot(domain string) string {
	host := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(domain)), "www.")
	if host == "" {
		return ""
	}

	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return host
	}

	return parts[len(parts)-2] + "." + parts[len(parts)-1]
}

func discoveryReasonFromCandidate(candidate discoveryCandidate) string {
	if candidate.SeedName == "" {
		return "Adjacent site related to your current sources"
	}
	if title := strings.TrimSpace(candidate.Result.Title); title != "" {
		return "Similar to " + candidate.SeedName + " - " + title
	}
	return "Similar to " + candidate.SeedName
}
