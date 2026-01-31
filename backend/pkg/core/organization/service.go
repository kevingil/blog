package organization

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"backend/pkg/core"
	"backend/pkg/types"

	"github.com/google/uuid"
)

// Service provides business logic for organizations
type Service struct {
	orgStore     OrganizationStore
	accountStore AccountStore
}

// NewService creates a new organization service with the provided stores
func NewService(orgStore OrganizationStore, accountStore AccountStore) *Service {
	return &Service{
		orgStore:     orgStore,
		accountStore: accountStore,
	}
}

// GetByID retrieves an organization by its ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*OrganizationResponse, error) {
	org, err := s.orgStore.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toResponse(org), nil
}

// GetBySlug retrieves an organization by its slug
func (s *Service) GetBySlug(ctx context.Context, slug string) (*OrganizationResponse, error) {
	org, err := s.orgStore.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return toResponse(org), nil
}

// List retrieves all organizations
func (s *Service) List(ctx context.Context) ([]OrganizationResponse, error) {
	orgs, err := s.orgStore.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]OrganizationResponse, len(orgs))
	for i, org := range orgs {
		result[i] = *toResponse(&org)
	}
	return result, nil
}

// Create creates a new organization
func (s *Service) Create(ctx context.Context, req CreateRequest) (*OrganizationResponse, error) {
	// Generate slug if not provided
	slug := req.Slug
	if slug == "" {
		slug = generateSlug(req.Name)
	}

	// Check if slug already exists
	existing, err := s.orgStore.FindBySlug(ctx, slug)
	if err != nil && err != core.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, core.ErrAlreadyExists
	}

	// Convert social links
	var socialLinks map[string]interface{}
	if req.SocialLinks != nil {
		socialLinks = make(map[string]interface{})
		for k, v := range *req.SocialLinks {
			socialLinks[k] = v
		}
	}

	org := &types.Organization{
		ID:              uuid.New(),
		Name:            req.Name,
		Slug:            slug,
		Bio:             req.Bio,
		LogoURL:         req.LogoURL,
		WebsiteURL:      req.WebsiteURL,
		EmailPublic:     req.EmailPublic,
		SocialLinks:     socialLinks,
		MetaDescription: req.MetaDescription,
	}

	if err := s.orgStore.Save(ctx, org); err != nil {
		return nil, err
	}

	return toResponse(org), nil
}

// Update updates an existing organization
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*OrganizationResponse, error) {
	org, err := s.orgStore.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if new slug is unique
	if req.Slug != nil && *req.Slug != org.Slug {
		existing, err := s.orgStore.FindBySlug(ctx, *req.Slug)
		if err != nil && err != core.ErrNotFound {
			return nil, err
		}
		if existing != nil {
			return nil, core.ErrAlreadyExists
		}
		org.Slug = *req.Slug
	}

	// Apply updates
	if req.Name != nil {
		org.Name = *req.Name
	}
	if req.Bio != nil {
		org.Bio = req.Bio
	}
	if req.LogoURL != nil {
		org.LogoURL = req.LogoURL
	}
	if req.WebsiteURL != nil {
		org.WebsiteURL = req.WebsiteURL
	}
	if req.EmailPublic != nil {
		org.EmailPublic = req.EmailPublic
	}
	if req.SocialLinks != nil {
		socialLinks := make(map[string]interface{})
		for k, v := range *req.SocialLinks {
			socialLinks[k] = v
		}
		org.SocialLinks = socialLinks
	}
	if req.MetaDescription != nil {
		org.MetaDescription = req.MetaDescription
	}

	if err := s.orgStore.Update(ctx, org); err != nil {
		return nil, err
	}

	return toResponse(org), nil
}

// Delete removes an organization by its ID
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.orgStore.Delete(ctx, id)
}

// JoinOrganization sets the organization for a user account
func (s *Service) JoinOrganization(ctx context.Context, accountID, orgID uuid.UUID) error {
	// Verify organization exists
	_, err := s.orgStore.FindByID(ctx, orgID)
	if err != nil {
		return err
	}

	// Get and update account
	account, err := s.accountStore.FindByID(ctx, accountID)
	if err != nil {
		return err
	}

	account.OrganizationID = &orgID
	return s.accountStore.Update(ctx, account)
}

// LeaveOrganization clears the organization for a user account
func (s *Service) LeaveOrganization(ctx context.Context, accountID uuid.UUID) error {
	account, err := s.accountStore.FindByID(ctx, accountID)
	if err != nil {
		return err
	}

	account.OrganizationID = nil
	return s.accountStore.Update(ctx, account)
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

func toResponse(org *types.Organization) *OrganizationResponse {
	socialLinks := make(map[string]string)
	if org.SocialLinks != nil {
		data, err := json.Marshal(org.SocialLinks)
		if err == nil {
			json.Unmarshal(data, &socialLinks)
		}
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

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
