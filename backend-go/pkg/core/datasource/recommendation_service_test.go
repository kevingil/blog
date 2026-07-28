package datasource_test

import (
	"context"
	"testing"

	"backend/pkg/api/dto"
	"backend/pkg/core"
	"backend/pkg/core/datasource"
	"backend/pkg/integrations/exa"
	"backend/pkg/types"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRecommendationSearchService struct {
	mock.Mock
}

func (m *mockRecommendationSearchService) Search(ctx context.Context, query string, options *exa.SearchOptions) (*exa.SearchResponse, error) {
	args := m.Called(ctx, query, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*exa.SearchResponse), args.Error(1)
}

func (m *mockRecommendationSearchService) FindSimilar(ctx context.Context, url string, options *exa.FindSimilarOptions) (*exa.SearchResponse, error) {
	args := m.Called(ctx, url, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*exa.SearchResponse), args.Error(1)
}

func (m *mockRecommendationSearchService) IsConfigured() bool {
	args := m.Called()
	return args.Bool(0)
}

func TestRecommendationService_Recommend(t *testing.T) {
	ctx := context.Background()

	t.Run("returns normalized recommendations grouped by domain", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockSearch := new(mockRecommendationSearchService)
		svc := datasource.NewRecommendationService(mockDSStore, mockSearch)

		orgID := uuid.New()
		req := dto.DataSourceRecommendationRequest{
			Query: "best AI engineering blogs",
			Limit: 3,
		}

		mockDSStore.On("FindByOrganizationID", ctx, orgID).Return([]types.DataSource{
			{Name: "Existing", URL: "https://existing.dev"},
		}, nil).Once()
		mockSearch.On("IsConfigured").Return(true).Once()
		mockSearch.On("Search", ctx, req.Query, mock.MatchedBy(func(options *exa.SearchOptions) bool {
			return options != nil && options.NumResults == 9 && options.IncludeSummary && options.IncludeHighlights && options.IncludeText && options.UseAutoprompt
		})).Return(&exa.SearchResponse{
			Results: []exa.SearchResult{
				{
					Title:      "AI Engineer from Example",
					URL:        "https://example.com/posts/ai-engineer",
					Summary:    "In-depth AI engineering coverage.",
					Highlights: []string{"Covers AI engineering releases and workflows."},
					Score:      0.92,
				},
				{
					Title:      "Another Example story",
					URL:        "https://example.com/posts/second",
					Summary:    "Duplicate domain that should be grouped.",
					Highlights: []string{"Second result from same domain."},
					Score:      0.81,
				},
				{
					Title:      "Open Source Weekly",
					URL:        "https://www.opensourceweekly.dev/archive/latest",
					Summary:    "Newsletter for open source AI tooling.",
					Highlights: []string{"Weekly curated issue."},
					Score:      0.88,
				},
				{
					Title:      "Existing Source result",
					URL:        "https://existing.dev/blog/post",
					Summary:    "Should be filtered because the source already exists.",
					Highlights: []string{"Already present."},
					Score:      0.7,
				},
			},
		}, nil).Once()

		result, err := svc.Recommend(ctx, &orgID, nil, req)

		assert.NoError(t, err)
		if assert.NotNil(t, result) {
			assert.Equal(t, req.Query, result.Query)
			assert.Equal(t, "query", result.Mode)
			assert.Len(t, result.Recommendations, 2)
			assert.Equal(t, "Example", result.Recommendations[0].Name)
			assert.Equal(t, "https://example.com", result.Recommendations[0].URL)
			assert.Equal(t, "example.com", result.Recommendations[0].Domain)
			assert.Equal(t, "blog", result.Recommendations[0].SourceType)
			assert.Equal(t, "https://example.com/posts/ai-engineer", result.Recommendations[0].SampleURL)
			assert.Equal(t, "Opensourceweekly", result.Recommendations[1].Name)
			assert.Equal(t, "newsletter", result.Recommendations[1].SourceType)
		}

		mockDSStore.AssertExpectations(t)
		mockSearch.AssertExpectations(t)
	})

	t.Run("returns discovery recommendations from existing manual sources", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockSearch := new(mockRecommendationSearchService)
		svc := datasource.NewRecommendationService(mockDSStore, mockSearch)

		userID := uuid.New()
		mockDSStore.On("FindByUserID", ctx, userID).Return([]types.DataSource{
			{Name: "Lenny's Newsletter", URL: "https://www.lennysnewsletter.com", IsEnabled: true},
			{Name: "Existing Discovered", URL: "https://adjacent.dev", IsEnabled: true, IsDiscovered: true},
			{Name: "Disabled Source", URL: "https://disabled.dev", IsEnabled: false},
		}, nil).Once()
		mockSearch.On("IsConfigured").Return(true).Once()
		mockSearch.On("FindSimilar", ctx, "https://www.lennysnewsletter.com", mock.MatchedBy(func(options *exa.FindSimilarOptions) bool {
			return options != nil && options.NumResults == 6 && options.ExcludeSourceDomain && options.IncludeSummary && options.IncludeHighlights && options.IncludeText
		})).Return(&exa.SearchResponse{
			Results: []exa.SearchResult{
				{
					Title:      "First Round Review",
					URL:        "https://review.firstround.com/growth",
					Summary:    "Startup and product writing.",
					Highlights: []string{"Great fit for startup operators."},
					Score:      0.91,
				},
				{
					Title:      "Adjacent should be filtered",
					URL:        "https://adjacent.dev/post",
					Summary:    "Already added source.",
					Highlights: []string{"Existing source."},
					Score:      0.89,
				},
			},
		}, nil).Once()

		result, err := svc.RecommendFromExistingSources(ctx, nil, &userID, dto.DataSourceDiscoveryRecommendationRequest{
			Limit: 4,
		})

		assert.NoError(t, err)
		if assert.NotNil(t, result) {
			assert.Equal(t, "discovery", result.Mode)
			assert.Equal(t, 1, result.SeedCount)
			assert.Len(t, result.Recommendations, 1)
			assert.Equal(t, "https://review.firstround.com", result.Recommendations[0].URL)
			assert.Contains(t, result.Recommendations[0].Reason, "Lenny's Newsletter")
		}

		mockDSStore.AssertExpectations(t)
		mockSearch.AssertExpectations(t)
	})

	t.Run("returns empty discovery recommendations when no manual seeds exist", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockSearch := new(mockRecommendationSearchService)
		svc := datasource.NewRecommendationService(mockDSStore, mockSearch)

		userID := uuid.New()
		mockDSStore.On("FindByUserID", ctx, userID).Return([]types.DataSource{
			{Name: "Discovered", URL: "https://seeded.dev", IsEnabled: true, IsDiscovered: true},
		}, nil).Once()
		mockSearch.On("IsConfigured").Return(true).Once()

		result, err := svc.RecommendFromExistingSources(ctx, nil, &userID, dto.DataSourceDiscoveryRecommendationRequest{})

		assert.NoError(t, err)
		if assert.NotNil(t, result) {
			assert.Equal(t, "discovery", result.Mode)
			assert.Equal(t, 0, result.SeedCount)
			assert.Empty(t, result.Recommendations)
		}

		mockDSStore.AssertExpectations(t)
		mockSearch.AssertExpectations(t)
	})

	t.Run("returns provider error when exa is unavailable", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockSearch := new(mockRecommendationSearchService)
		svc := datasource.NewRecommendationService(mockDSStore, mockSearch)

		mockSearch.On("IsConfigured").Return(false).Once()

		result, err := svc.Recommend(ctx, nil, &uuid.UUID{}, dto.DataSourceRecommendationRequest{
			Query: "security news",
		})

		assert.Nil(t, result)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrExternal)
		mockSearch.AssertExpectations(t)
	})

	t.Run("returns recommendations for user-owned sources", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockSearch := new(mockRecommendationSearchService)
		svc := datasource.NewRecommendationService(mockDSStore, mockSearch)

		userID := uuid.New()
		mockDSStore.On("FindByUserID", ctx, userID).Return([]types.DataSource{}, nil).Once()
		mockSearch.On("IsConfigured").Return(true).Once()
		mockSearch.On("Search", ctx, "observability blogs", mock.Anything).Return(&exa.SearchResponse{
			Results: []exa.SearchResult{
				{
					Title:   "Observability News",
					URL:     "https://ops.example.org/blog/post",
					Summary: "Monitoring and observability updates.",
					Score:   0.76,
				},
			},
		}, nil).Once()

		result, err := svc.Recommend(ctx, nil, &userID, dto.DataSourceRecommendationRequest{
			Query: "observability blogs",
		})

		assert.NoError(t, err)
		if assert.NotNil(t, result) {
			assert.Len(t, result.Recommendations, 1)
			assert.Equal(t, "https://ops.example.org", result.Recommendations[0].URL)
		}

		mockDSStore.AssertExpectations(t)
		mockSearch.AssertExpectations(t)
	})
}
