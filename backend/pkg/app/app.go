// Package app provides the main application container for dependency injection
package app

import (
	"backend/pkg/config"
	"backend/pkg/core/article"
	"backend/pkg/core/auth"
	"backend/pkg/core/chat"
	"backend/pkg/core/image"
	"backend/pkg/core/organization"
	"backend/pkg/core/page"
	"backend/pkg/core/profile"
	"backend/pkg/core/project"
	"backend/pkg/core/source"
	"backend/pkg/core/tag"
	"backend/pkg/database"
	"backend/pkg/database/repository"

	"gorm.io/gorm"
)

// App is the main application container that holds all dependencies
type App struct {
	// Infrastructure
	DB        *gorm.DB
	DBService database.Service
	Config    *config.Config

	// Core Services
	Articles      *article.Service
	Auth          *auth.Service
	Pages         *page.Service
	Projects      *project.Service
	Tags          *tag.Service
	Sources       *source.Service
	Images        *image.Service
	Organizations *organization.Service
	Profiles      *profile.Service
	Chat          *chat.MessageService
}

// New creates a new App with all dependencies initialized
func New(dbService database.Service, cfg *config.Config) *App {
	db := dbService.GetDB()

	app := &App{
		DB:        db,
		DBService: dbService,
		Config:    cfg,
	}

	// Initialize repositories
	articleRepo := repository.NewArticleRepository(db)
	pageRepo := repository.NewPageRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	tagRepo := repository.NewTagRepository(db)
	sourceRepo := repository.NewSourceRepository(db)
	imageRepo := repository.NewImageRepository(db)
	orgRepo := repository.NewOrganizationRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	settingsRepo := repository.NewSiteSettingsRepository(db)
	profileRepo := repository.NewProfileRepository(db)

	// Initialize core services with their repositories
	app.Tags = tag.NewService(tagRepo)
	app.Articles = article.NewService(articleRepo, tagRepo)
	app.Auth = auth.NewService(accountRepo, cfg.Auth.SecretKey)
	app.Pages = page.NewService(pageRepo)
	app.Projects = project.NewService(projectRepo, tagRepo)
	app.Sources = source.NewService(sourceRepo)
	app.Images = image.NewService(imageRepo)
	app.Organizations = organization.NewService(orgRepo, accountRepo)
	app.Profiles = profile.NewService(settingsRepo, profileRepo)
	app.Chat = chat.NewMessageService(dbService)

	return app
}
