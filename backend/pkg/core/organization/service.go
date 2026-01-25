package organization

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"backend/pkg/core"
	"backend/pkg/database"
	"backend/pkg/database/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// CreateRequest represents a request to create an organization
type CreateRequest struct {
	Name            string             `json:"name" validate:"required,min=2,max=255"`
	Slug            string             `json:"slug" validate:"omitempty,min=2,max=100"`
	Bio             *string            `json:"bio"`
	LogoURL         *string            `json:"logo_url"`
	WebsiteURL      *string            `json:"website_url"`
	EmailPublic     *string            `json:"email_public"`
	SocialLinks     *map[string]string `json:"social_links"`
	MetaDescription *string            `json:"meta_description"`
}

// UpdateRequest represents a request to update an organization
type UpdateRequest struct {
	Name            *string            `json:"name"`
	Slug            *string            `json:"slug"`
	Bio             *string            `json:"bio"`
	LogoURL         *string            `json:"logo_url"`
	WebsiteURL      *string            `json:"website_url"`
	EmailPublic     *string            `json:"email_public"`
	SocialLinks     *map[string]string `json:"social_links"`
	MetaDescription *string            `json:"meta_description"`
}

// OrganizationResponse is the response for an organization
type OrganizationResponse struct {
	ID              uuid.UUID         `json:"id"`
	Name            string            `json:"name"`
	Slug            string            `json:"slug"`
	Bio             string            `json:"bio"`
	LogoURL         string            `json:"logo_url"`
	WebsiteURL      string            `json:"website_url"`
	EmailPublic     string            `json:"email_public"`
	SocialLinks     map[string]string `json:"social_links"`
	MetaDescription string            `json:"meta_description"`
}

// GetByID retrieves an organization by its ID
func GetByID(ctx context.Context, id uuid.UUID) (*OrganizationResponse, error) {
	db := database.DB()

	var model models.Organization
	if err := db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return modelToResponse(&model), nil
}

// GetBySlug retrieves an organization by its slug
func GetBySlug(ctx context.Context, slug string) (*OrganizationResponse, error) {
	db := database.DB()

	var model models.Organization
	if err := db.WithContext(ctx).Where("slug = ?", slug).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return modelToResponse(&model), nil
}

// List retrieves all organizations
func List(ctx context.Context) ([]OrganizationResponse, error) {
	db := database.DB()

	var orgModels []models.Organization
	if err := db.WithContext(ctx).Order("name ASC").Find(&orgModels).Error; err != nil {
		return nil, err
	}

	result := make([]OrganizationResponse, len(orgModels))
	for i, model := range orgModels {
		result[i] = *modelToResponse(&model)
	}
	return result, nil
}

// Create creates a new organization
func Create(ctx context.Context, req CreateRequest) (*OrganizationResponse, error) {
	db := database.DB()

	// Generate slug if not provided
	slug := req.Slug
	if slug == "" {
		slug = generateSlug(req.Name)
	}

	// Check if slug already exists
	var count int64
	db.WithContext(ctx).Model(&models.Organization{}).Where("slug = ?", slug).Count(&count)
	if count > 0 {
		return nil, core.ErrAlreadyExists
	}

	// Marshal social links
	var socialLinksJSON datatypes.JSON
	if req.SocialLinks != nil {
		data, err := json.Marshal(*req.SocialLinks)
		if err != nil {
			return nil, err
		}
		socialLinksJSON = datatypes.JSON(data)
	} else {
		socialLinksJSON = datatypes.JSON([]byte("{}"))
	}

	model := models.Organization{
		ID:              uuid.New(),
		Name:            req.Name,
		Slug:            slug,
		Bio:             req.Bio,
		LogoURL:         req.LogoURL,
		WebsiteURL:      req.WebsiteURL,
		EmailPublic:     req.EmailPublic,
		SocialLinks:     socialLinksJSON,
		MetaDescription: req.MetaDescription,
	}

	if err := db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, err
	}

	return modelToResponse(&model), nil
}

// Update updates an existing organization
func Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*OrganizationResponse, error) {
	db := database.DB()

	var model models.Organization
	if err := db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	// Check if new slug is unique
	if req.Slug != nil && *req.Slug != model.Slug {
		var count int64
		db.WithContext(ctx).Model(&models.Organization{}).Where("slug = ? AND id != ?", *req.Slug, id).Count(&count)
		if count > 0 {
			return nil, core.ErrAlreadyExists
		}
		model.Slug = *req.Slug
	}

	// Apply updates
	if req.Name != nil {
		model.Name = *req.Name
	}
	if req.Bio != nil {
		model.Bio = req.Bio
	}
	if req.LogoURL != nil {
		model.LogoURL = req.LogoURL
	}
	if req.WebsiteURL != nil {
		model.WebsiteURL = req.WebsiteURL
	}
	if req.EmailPublic != nil {
		model.EmailPublic = req.EmailPublic
	}
	if req.SocialLinks != nil {
		data, err := json.Marshal(*req.SocialLinks)
		if err != nil {
			return nil, err
		}
		model.SocialLinks = datatypes.JSON(data)
	}
	if req.MetaDescription != nil {
		model.MetaDescription = req.MetaDescription
	}

	if err := db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, err
	}

	return modelToResponse(&model), nil
}

// Delete removes an organization by its ID
func Delete(ctx context.Context, id uuid.UUID) error {
	db := database.DB()
	result := db.WithContext(ctx).Delete(&models.Organization{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// JoinOrganization sets the organization for a user account
func JoinOrganization(ctx context.Context, accountID, orgID uuid.UUID) error {
	db := database.DB()

	// Verify organization exists
	var orgCount int64
	db.WithContext(ctx).Model(&models.Organization{}).Where("id = ?", orgID).Count(&orgCount)
	if orgCount == 0 {
		return core.ErrNotFound
	}

	// Update account
	result := db.WithContext(ctx).Model(&models.Account{}).Where("id = ?", accountID).Update("organization_id", orgID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// LeaveOrganization clears the organization for a user account
func LeaveOrganization(ctx context.Context, accountID uuid.UUID) error {
	db := database.DB()

	result := db.WithContext(ctx).Model(&models.Account{}).Where("id = ?", accountID).Update("organization_id", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// Helper functions

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	reg := regexp.MustCompile("[^a-z0-9-]")
	slug = reg.ReplaceAllString(slug, "")
	reg = regexp.MustCompile("-+")
	slug = reg.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

func modelToResponse(model *models.Organization) *OrganizationResponse {
	socialLinks := make(map[string]string)
	if model.SocialLinks != nil {
		_ = json.Unmarshal(model.SocialLinks, &socialLinks)
	}

	return &OrganizationResponse{
		ID:              model.ID,
		Name:            model.Name,
		Slug:            model.Slug,
		Bio:             stringValue(model.Bio),
		LogoURL:         stringValue(model.LogoURL),
		WebsiteURL:      stringValue(model.WebsiteURL),
		EmailPublic:     stringValue(model.EmailPublic),
		SocialLinks:     socialLinks,
		MetaDescription: stringValue(model.MetaDescription),
	}
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Legacy Service type for backward compatibility

type Service struct {
	store        OrganizationStore
	accountStore interface {
		FindByID(ctx context.Context, id uuid.UUID) (interface{}, error)
		Update(ctx context.Context, account interface{}) error
	}
}

func NewService(store OrganizationStore, accountStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (interface{}, error)
	Update(ctx context.Context, account interface{}) error
}) *Service {
	return &Service{
		store:        store,
		accountStore: accountStore,
	}
}
