package datasource_test

import (
	"context"
	"testing"
	"time"

	"backend/pkg/api/dto"
	"backend/pkg/core"
	"backend/pkg/core/datasource"
	"backend/pkg/types"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns data source when found", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dsID := uuid.New()
		orgID := uuid.New()
		testDS := &types.DataSource{
			ID:             dsID,
			OrganizationID: &orgID,
			Name:           "Test Blog",
			URL:            "https://example.com",
			SourceType:     "blog",
			CrawlFrequency: "daily",
			IsEnabled:      true,
			CrawlStatus:    "pending",
			ContentCount:   10,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		mockDSStore.On("FindByID", ctx, dsID).Return(testDS, nil).Once()

		result, err := svc.GetByID(ctx, dsID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, dsID, result.ID)
		assert.Equal(t, "Test Blog", result.Name)
		assert.Equal(t, "https://example.com", result.URL)
		mockDSStore.AssertExpectations(t)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dsID := uuid.New()
		mockDSStore.On("FindByID", ctx, dsID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetByID(ctx, dsID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockDSStore.AssertExpectations(t)
	})
}

func TestService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("returns data sources for organization", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		orgID := uuid.New()
		testSources := []types.DataSource{
			{
				ID:             uuid.New(),
				OrganizationID: &orgID,
				Name:           "Blog 1",
				URL:            "https://blog1.com",
				SourceType:     "blog",
				CrawlFrequency: "daily",
				IsEnabled:      true,
				CrawlStatus:    "completed",
			},
			{
				ID:             uuid.New(),
				OrganizationID: &orgID,
				Name:           "Blog 2",
				URL:            "https://blog2.com",
				SourceType:     "blog",
				CrawlFrequency: "weekly",
				IsEnabled:      false,
				CrawlStatus:    "pending",
			},
		}

		mockDSStore.On("FindByOrganizationID", ctx, orgID).Return(testSources, nil).Once()

		result, err := svc.List(ctx, orgID)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "Blog 1", result[0].Name)
		assert.Equal(t, "Blog 2", result[1].Name)
		mockDSStore.AssertExpectations(t)
	})

	t.Run("returns empty list when no sources found", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		orgID := uuid.New()
		mockDSStore.On("FindByOrganizationID", ctx, orgID).Return([]types.DataSource{}, nil).Once()

		result, err := svc.List(ctx, orgID)

		assert.NoError(t, err)
		assert.Len(t, result, 0)
		mockDSStore.AssertExpectations(t)
	})
}

func TestService_ListByUserID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns data sources for user", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		userID := uuid.New()
		testSources := []types.DataSource{
			{
				ID:             uuid.New(),
				UserID:         &userID,
				Name:           "User Blog 1",
				URL:            "https://userblog1.com",
				SourceType:     "blog",
				CrawlFrequency: "daily",
				IsEnabled:      true,
				CrawlStatus:    "completed",
			},
		}

		mockDSStore.On("FindByUserID", ctx, userID).Return(testSources, nil).Once()

		result, err := svc.ListByUserID(ctx, userID)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "User Blog 1", result[0].Name)
		mockDSStore.AssertExpectations(t)
	})
}

func TestService_ListAll(t *testing.T) {
	ctx := context.Background()

	t.Run("returns paginated data sources", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		testSources := []types.DataSource{
			{
				ID:             uuid.New(),
				Name:           "Source 1",
				URL:            "https://source1.com",
				SourceType:     "blog",
				CrawlFrequency: "daily",
			},
			{
				ID:             uuid.New(),
				Name:           "Source 2",
				URL:            "https://source2.com",
				SourceType:     "news",
				CrawlFrequency: "hourly",
			},
		}

		// page=1, limit=20 -> offset=0
		mockDSStore.On("List", ctx, 0, 20).Return(testSources, int64(50), nil).Once()

		result, total, err := svc.ListAll(ctx, 1, 20)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(50), total)
		mockDSStore.AssertExpectations(t)
	})

	t.Run("handles default pagination values", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		// page < 1 becomes 1, limit < 1 becomes 20
		mockDSStore.On("List", ctx, 0, 20).Return([]types.DataSource{}, int64(0), nil).Once()

		result, total, err := svc.ListAll(ctx, 0, 0)

		assert.NoError(t, err)
		assert.Len(t, result, 0)
		assert.Equal(t, int64(0), total)
		mockDSStore.AssertExpectations(t)
	})

	t.Run("caps limit at 100", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		// limit > 100 becomes 20
		mockDSStore.On("List", ctx, 0, 20).Return([]types.DataSource{}, int64(0), nil).Once()

		_, _, err := svc.ListAll(ctx, 1, 150)

		assert.NoError(t, err)
		mockDSStore.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates data source successfully with organization", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		orgID := uuid.New()
		req := dto.DataSourceCreateRequest{
			Name:           "New Blog",
			URL:            "https://newblog.com",
			SourceType:     "blog",
			CrawlFrequency: "daily",
		}

		mockDSStore.On("FindByURL", ctx, "https://newblog.com").Return(nil, core.ErrNotFound).Once()
		mockDSStore.On("Save", ctx, mock.AnythingOfType("*types.DataSource")).Return(nil).Once()

		result, err := svc.Create(ctx, &orgID, nil, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Blog", result.Name)
		assert.Equal(t, "https://newblog.com", result.URL)
		assert.Equal(t, "blog", result.SourceType)
		assert.Equal(t, "daily", result.CrawlFrequency)
		assert.True(t, result.IsEnabled)
		mockDSStore.AssertExpectations(t)
	})

	t.Run("creates data source successfully with user", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		userID := uuid.New()
		req := dto.DataSourceCreateRequest{
			Name: "User Blog",
			URL:  "https://userblog.com",
		}

		mockDSStore.On("FindByURL", ctx, "https://userblog.com").Return(nil, core.ErrNotFound).Once()
		mockDSStore.On("Save", ctx, mock.AnythingOfType("*types.DataSource")).Return(nil).Once()

		result, err := svc.Create(ctx, nil, &userID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "User Blog", result.Name)
		// Defaults should be applied
		assert.Equal(t, "blog", result.SourceType)
		assert.Equal(t, "daily", result.CrawlFrequency)
		mockDSStore.AssertExpectations(t)
	})

	t.Run("returns error when URL already exists", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		orgID := uuid.New()
		req := dto.DataSourceCreateRequest{
			Name: "Duplicate Blog",
			URL:  "https://existing.com",
		}

		existingDS := &types.DataSource{
			ID:  uuid.New(),
			URL: "https://existing.com",
		}
		mockDSStore.On("FindByURL", ctx, "https://existing.com").Return(existingDS, nil).Once()

		result, err := svc.Create(ctx, &orgID, nil, req)

		assert.ErrorIs(t, err, core.ErrAlreadyExists)
		assert.Nil(t, result)
		mockDSStore.AssertExpectations(t)
	})

	t.Run("returns error when neither orgID nor userID provided", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		req := dto.DataSourceCreateRequest{
			Name: "Orphan Blog",
			URL:  "https://orphan.com",
		}

		result, err := svc.Create(ctx, nil, nil, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		// Store methods should not be called
		mockDSStore.AssertExpectations(t)
	})
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("updates data source successfully", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dsID := uuid.New()
		orgID := uuid.New()
		existingDS := &types.DataSource{
			ID:             dsID,
			OrganizationID: &orgID,
			Name:           "Old Name",
			URL:            "https://oldurl.com",
			SourceType:     "blog",
			CrawlFrequency: "daily",
			IsEnabled:      true,
			CrawlStatus:    "pending",
		}

		newName := "Updated Name"
		newURL := "https://newurl.com"
		req := dto.DataSourceUpdateRequest{
			Name: &newName,
			URL:  &newURL,
		}

		mockDSStore.On("FindByID", ctx, dsID).Return(existingDS, nil).Once()
		mockDSStore.On("FindByURL", ctx, newURL).Return(nil, core.ErrNotFound).Once()
		mockDSStore.On("Update", ctx, mock.AnythingOfType("*types.DataSource")).Return(nil).Once()

		result, err := svc.Update(ctx, dsID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Updated Name", result.Name)
		assert.Equal(t, "https://newurl.com", result.URL)
		mockDSStore.AssertExpectations(t)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dsID := uuid.New()
		req := dto.DataSourceUpdateRequest{}

		mockDSStore.On("FindByID", ctx, dsID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.Update(ctx, dsID, req)

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockDSStore.AssertExpectations(t)
	})

	t.Run("returns error when new URL already exists", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dsID := uuid.New()
		existingDS := &types.DataSource{
			ID:   dsID,
			Name: "My Blog",
			URL:  "https://myblog.com",
		}

		newURL := "https://taken.com"
		req := dto.DataSourceUpdateRequest{
			URL: &newURL,
		}

		otherDS := &types.DataSource{
			ID:  uuid.New(),
			URL: "https://taken.com",
		}

		mockDSStore.On("FindByID", ctx, dsID).Return(existingDS, nil).Once()
		mockDSStore.On("FindByURL", ctx, newURL).Return(otherDS, nil).Once()

		result, err := svc.Update(ctx, dsID, req)

		assert.ErrorIs(t, err, core.ErrAlreadyExists)
		assert.Nil(t, result)
		mockDSStore.AssertExpectations(t)
	})

	t.Run("allows updating to same URL", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dsID := uuid.New()
		existingDS := &types.DataSource{
			ID:             dsID,
			Name:           "My Blog",
			URL:            "https://myblog.com",
			SourceType:     "blog",
			CrawlFrequency: "daily",
		}

		sameURL := "https://myblog.com"
		req := dto.DataSourceUpdateRequest{
			URL: &sameURL,
		}

		mockDSStore.On("FindByID", ctx, dsID).Return(existingDS, nil).Once()
		// URL unchanged, no URL lookup needed
		mockDSStore.On("Update", ctx, mock.AnythingOfType("*types.DataSource")).Return(nil).Once()

		result, err := svc.Update(ctx, dsID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockDSStore.AssertExpectations(t)
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes data source successfully", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dsID := uuid.New()
		mockDSStore.On("Delete", ctx, dsID).Return(nil).Once()

		err := svc.Delete(ctx, dsID)

		assert.NoError(t, err)
		mockDSStore.AssertExpectations(t)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dsID := uuid.New()
		mockDSStore.On("Delete", ctx, dsID).Return(core.ErrNotFound).Once()

		err := svc.Delete(ctx, dsID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockDSStore.AssertExpectations(t)
	})
}

func TestService_GetContent(t *testing.T) {
	ctx := context.Background()

	t.Run("returns paginated content for data source", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dsID := uuid.New()
		title1 := "Article 1"
		title2 := "Article 2"
		testContent := []types.CrawledContent{
			{
				ID:           uuid.New(),
				DataSourceID: dsID,
				URL:          "https://example.com/post1",
				Title:        &title1,
				Content:      "Content of article 1",
				CreatedAt:    time.Now(),
			},
			{
				ID:           uuid.New(),
				DataSourceID: dsID,
				URL:          "https://example.com/post2",
				Title:        &title2,
				Content:      "Content of article 2",
				CreatedAt:    time.Now(),
			},
		}

		// page=1, limit=20 -> offset=0
		mockContentStore.On("FindByDataSourceID", ctx, dsID, 0, 20).Return(testContent, int64(25), nil).Once()

		result, total, err := svc.GetContent(ctx, dsID, 1, 20)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(25), total)
		assert.Equal(t, "Article 1", *result[0].Title)
		assert.Equal(t, "Article 2", *result[1].Title)
		mockContentStore.AssertExpectations(t)
	})

	t.Run("handles default pagination values", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dsID := uuid.New()

		// page < 1 becomes 1, limit < 1 becomes 20
		mockContentStore.On("FindByDataSourceID", ctx, dsID, 0, 20).Return([]types.CrawledContent{}, int64(0), nil).Once()

		result, total, err := svc.GetContent(ctx, dsID, 0, 0)

		assert.NoError(t, err)
		assert.Len(t, result, 0)
		assert.Equal(t, int64(0), total)
		mockContentStore.AssertExpectations(t)
	})
}

func TestService_GetDueToCrawl(t *testing.T) {
	ctx := context.Background()

	t.Run("returns data sources due for crawling", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dueSources := []types.DataSource{
			{
				ID:             uuid.New(),
				Name:           "Blog 1",
				URL:            "https://blog1.com",
				CrawlFrequency: "daily",
				CrawlStatus:    "pending",
			},
			{
				ID:             uuid.New(),
				Name:           "Blog 2",
				URL:            "https://blog2.com",
				CrawlFrequency: "hourly",
				CrawlStatus:    "pending",
			},
		}

		mockDSStore.On("FindDueToCrawl", ctx, 10).Return(dueSources, nil).Once()

		result, err := svc.GetDueToCrawl(ctx, 10)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "Blog 1", result[0].Name)
		assert.Equal(t, "Blog 2", result[1].Name)
		mockDSStore.AssertExpectations(t)
	})
}

func TestService_TriggerCrawl(t *testing.T) {
	ctx := context.Background()

	t.Run("triggers crawl successfully", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dsID := uuid.New()
		existingDS := &types.DataSource{
			ID:          dsID,
			Name:        "Test Blog",
			URL:         "https://test.com",
			CrawlStatus: "completed",
		}

		mockDSStore.On("FindByID", ctx, dsID).Return(existingDS, nil).Once()
		mockDSStore.On("Update", ctx, mock.AnythingOfType("*types.DataSource")).Return(nil).Once()

		err := svc.TriggerCrawl(ctx, dsID)

		assert.NoError(t, err)
		mockDSStore.AssertExpectations(t)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dsID := uuid.New()
		mockDSStore.On("FindByID", ctx, dsID).Return(nil, core.ErrNotFound).Once()

		err := svc.TriggerCrawl(ctx, dsID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockDSStore.AssertExpectations(t)
	})
}

func TestService_UpdateCrawlStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("updates crawl status successfully", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dsID := uuid.New()
		mockDSStore.On("UpdateCrawlStatus", ctx, dsID, "completed", (*string)(nil)).Return(nil).Once()

		err := svc.UpdateCrawlStatus(ctx, dsID, "completed", nil)

		assert.NoError(t, err)
		mockDSStore.AssertExpectations(t)
	})

	t.Run("updates crawl status with error message", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dsID := uuid.New()
		errMsg := "Connection timeout"
		mockDSStore.On("UpdateCrawlStatus", ctx, dsID, "failed", &errMsg).Return(nil).Once()

		err := svc.UpdateCrawlStatus(ctx, dsID, "failed", &errMsg)

		assert.NoError(t, err)
		mockDSStore.AssertExpectations(t)
	})
}

func TestService_SetNextCrawlTime(t *testing.T) {
	ctx := context.Background()

	t.Run("sets next crawl time successfully", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		dsID := uuid.New()
		mockDSStore.On("UpdateNextCrawlAt", ctx, dsID, mock.AnythingOfType("time.Time")).Return(nil).Once()

		err := svc.SetNextCrawlTime(ctx, dsID, "daily")

		assert.NoError(t, err)
		mockDSStore.AssertExpectations(t)
	})
}

func TestService_CreateDiscoveredSource(t *testing.T) {
	ctx := context.Background()

	t.Run("creates discovered source successfully with organization", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		orgID := uuid.New()
		discoveredFromID := uuid.New()

		mockDSStore.On("FindByURL", ctx, "https://discovered.com").Return(nil, core.ErrNotFound).Once()
		mockDSStore.On("Save", ctx, mock.AnythingOfType("*types.DataSource")).Return(nil).Once()

		result, err := svc.CreateDiscoveredSource(ctx, &orgID, nil, discoveredFromID, "Discovered Blog", "https://discovered.com")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Discovered Blog", result.Name)
		assert.Equal(t, "https://discovered.com", result.URL)
		assert.True(t, result.IsDiscovered)
		assert.False(t, result.IsEnabled) // Disabled by default
		mockDSStore.AssertExpectations(t)
	})

	t.Run("creates discovered source successfully with user", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		userID := uuid.New()
		discoveredFromID := uuid.New()

		mockDSStore.On("FindByURL", ctx, "https://userdisc.com").Return(nil, core.ErrNotFound).Once()
		mockDSStore.On("Save", ctx, mock.AnythingOfType("*types.DataSource")).Return(nil).Once()

		result, err := svc.CreateDiscoveredSource(ctx, nil, &userID, discoveredFromID, "User Discovered", "https://userdisc.com")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsDiscovered)
		mockDSStore.AssertExpectations(t)
	})

	t.Run("returns error when URL already exists", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		orgID := uuid.New()
		discoveredFromID := uuid.New()
		existingDS := &types.DataSource{
			ID:  uuid.New(),
			URL: "https://existing.com",
		}

		mockDSStore.On("FindByURL", ctx, "https://existing.com").Return(existingDS, nil).Once()

		result, err := svc.CreateDiscoveredSource(ctx, &orgID, nil, discoveredFromID, "Duplicate", "https://existing.com")

		assert.ErrorIs(t, err, core.ErrAlreadyExists)
		assert.Nil(t, result)
		mockDSStore.AssertExpectations(t)
	})

	t.Run("returns error when neither orgID nor userID provided", func(t *testing.T) {
		mockDSStore := new(mocks.MockDataSourceRepository)
		mockContentStore := new(mocks.MockCrawledContentRepository)
		svc := datasource.NewService(mockDSStore, mockContentStore)

		discoveredFromID := uuid.New()

		result, err := svc.CreateDiscoveredSource(ctx, nil, nil, discoveredFromID, "Orphan", "https://orphan.com")

		assert.Error(t, err)
		assert.Nil(t, result)
		mockDSStore.AssertExpectations(t)
	})
}
