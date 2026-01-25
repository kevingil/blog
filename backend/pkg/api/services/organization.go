package services

import (
	"backend/pkg/database"
	"backend/pkg/core"
	"backend/pkg/models"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// OrganizationService provides methods to interact with organizations
type OrganizationService struct {
	db database.Service
}

func NewOrganizationService(db database.Service) *OrganizationService {
	return &OrganizationService{db: db}
}

// OrganizationCreateRequest is the request to create an organization
type OrganizationCreateRequest struct {
	Name            string             `json:"name" validate:"required,min=2,max=255"`
	Slug            string             `json:"slug" validate:"omitempty,min=2,max=100"`
	Bio             *string            `json:"bio"`
	LogoURL         *string            `json:"logo_url"`
	WebsiteURL      *string            `json:"website_url"`
	EmailPublic     *string            `json:"email_public"`
	SocialLinks     *map[string]string `json:"social_links"`
	MetaDescription *string            `json:"meta_description"`
}

// OrganizationUpdateRequest is the request to update an organization
type OrganizationUpdateRequest struct {
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

// CreateOrganization creates a new organization
func (s *OrganizationService) CreateOrganization(req OrganizationCreateRequest) (*OrganizationResponse, error) {
	db := s.db.GetDB()

	// Generate slug if not provided
	slug := req.Slug
	if slug == "" {
		slug = generateOrgSlug(req.Name)
	}

	// Check if slug already exists
	var existing models.Organization
	if err := db.Where("slug = ?", slug).First(&existing).Error; err == nil {
		return nil, core.AlreadyExistsError("Organization with this slug")
	}

	var socialLinksJSON datatypes.JSON
	if req.SocialLinks != nil {
		data, err := json.Marshal(*req.SocialLinks)
		if err != nil {
			return nil, core.InternalError("Failed to marshal social_links")
		}
		socialLinksJSON = datatypes.JSON(data)
	} else {
		socialLinksJSON = datatypes.JSON([]byte("{}"))
	}

	org := models.Organization{
		Name:            req.Name,
		Slug:            slug,
		Bio:             req.Bio,
		LogoURL:         req.LogoURL,
		WebsiteURL:      req.WebsiteURL,
		EmailPublic:     req.EmailPublic,
		SocialLinks:     socialLinksJSON,
		MetaDescription: req.MetaDescription,
	}

	if err := db.Create(&org).Error; err != nil {
		return nil, core.InternalError("Failed to create organization")
	}

	return s.toResponse(&org), nil
}

// GetOrganizationByID returns an organization by ID
func (s *OrganizationService) GetOrganizationByID(id uuid.UUID) (*OrganizationResponse, error) {
	db := s.db.GetDB()

	var org models.Organization
	if err := db.First(&org, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Organization")
		}
		return nil, core.InternalError("Failed to fetch organization")
	}

	return s.toResponse(&org), nil
}

// GetOrganizationBySlug returns an organization by slug
func (s *OrganizationService) GetOrganizationBySlug(slug string) (*OrganizationResponse, error) {
	db := s.db.GetDB()

	var org models.Organization
	if err := db.First(&org, "slug = ?", slug).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Organization")
		}
		return nil, core.InternalError("Failed to fetch organization")
	}

	return s.toResponse(&org), nil
}

// ListOrganizations returns all organizations
func (s *OrganizationService) ListOrganizations() ([]OrganizationResponse, error) {
	db := s.db.GetDB()

	var orgs []models.Organization
	if err := db.Order("name ASC").Find(&orgs).Error; err != nil {
		return nil, core.InternalError("Failed to fetch organizations")
	}

	result := make([]OrganizationResponse, len(orgs))
	for i, org := range orgs {
		result[i] = *s.toResponse(&org)
	}

	return result, nil
}

// UpdateOrganization updates an organization
func (s *OrganizationService) UpdateOrganization(id uuid.UUID, req OrganizationUpdateRequest) (*OrganizationResponse, error) {
	db := s.db.GetDB()

	var org models.Organization
	if err := db.First(&org, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Organization")
		}
		return nil, core.InternalError("Failed to fetch organization")
	}

	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Slug != nil {
		// Check if new slug is unique
		var existing models.Organization
		if err := db.Where("slug = ? AND id != ?", *req.Slug, id).First(&existing).Error; err == nil {
			return nil, core.AlreadyExistsError("Organization with this slug")
		}
		updates["slug"] = *req.Slug
	}
	if req.Bio != nil {
		updates["bio"] = *req.Bio
	}
	if req.LogoURL != nil {
		updates["logo_url"] = *req.LogoURL
	}
	if req.WebsiteURL != nil {
		updates["website_url"] = *req.WebsiteURL
	}
	if req.EmailPublic != nil {
		updates["email_public"] = *req.EmailPublic
	}
	if req.MetaDescription != nil {
		updates["meta_description"] = *req.MetaDescription
	}
	if req.SocialLinks != nil {
		data, err := json.Marshal(*req.SocialLinks)
		if err != nil {
			return nil, core.InternalError("Failed to marshal social_links")
		}
		updates["social_links"] = datatypes.JSON(data)
	}

	if len(updates) > 0 {
		if err := db.Model(&org).Updates(updates).Error; err != nil {
			return nil, core.InternalError("Failed to update organization")
		}
	}

	// Reload the organization
	if err := db.First(&org, "id = ?", id).Error; err != nil {
		return nil, core.InternalError("Failed to reload organization")
	}

	return s.toResponse(&org), nil
}

// DeleteOrganization deletes an organization
func (s *OrganizationService) DeleteOrganization(id uuid.UUID) error {
	db := s.db.GetDB()

	result := db.Delete(&models.Organization{}, "id = ?", id)
	if result.Error != nil {
		return core.InternalError("Failed to delete organization")
	}

	if result.RowsAffected == 0 {
		return core.NotFoundError("Organization")
	}

	return nil
}

// JoinOrganization sets the organization for a user account
func (s *OrganizationService) JoinOrganization(accountID uuid.UUID, orgID uuid.UUID) error {
	db := s.db.GetDB()

	// Verify organization exists
	var org models.Organization
	if err := db.First(&org, "id = ?", orgID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return core.NotFoundError("Organization")
		}
		return core.InternalError("Failed to fetch organization")
	}

	// Update account
	result := db.Model(&models.Account{}).Where("id = ?", accountID).Update("organization_id", orgID)
	if result.Error != nil {
		return core.InternalError("Failed to join organization")
	}

	if result.RowsAffected == 0 {
		return core.NotFoundError("Account")
	}

	return nil
}

// LeaveOrganization clears the organization for a user account
func (s *OrganizationService) LeaveOrganization(accountID uuid.UUID) error {
	db := s.db.GetDB()

	result := db.Model(&models.Account{}).Where("id = ?", accountID).Update("organization_id", nil)
	if result.Error != nil {
		return core.InternalError("Failed to leave organization")
	}

	if result.RowsAffected == 0 {
		return core.NotFoundError("Account")
	}

	return nil
}

// Helper functions

func (s *OrganizationService) toResponse(org *models.Organization) *OrganizationResponse {
	socialLinks := make(map[string]string)
	if org.SocialLinks != nil {
		_ = json.Unmarshal(org.SocialLinks, &socialLinks)
	}

	return &OrganizationResponse{
		ID:              org.ID,
		Name:            org.Name,
		Slug:            org.Slug,
		Bio:             stringValue(org.Bio),
		LogoURL:         stringValue(org.LogoURL),
		WebsiteURL:      stringValue(org.WebsiteURL),
		EmailPublic:     stringValue(org.EmailPublic),
		SocialLinks:     socialLinks,
		MetaDescription: stringValue(org.MetaDescription),
	}
}

func generateOrgSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)
	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove non-alphanumeric characters except hyphens
	reg := regexp.MustCompile("[^a-z0-9-]")
	slug = reg.ReplaceAllString(slug, "")
	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile("-+")
	slug = reg.ReplaceAllString(slug, "-")
	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")
	return slug
}
