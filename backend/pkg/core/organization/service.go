package organization

import (
	"context"
	"regexp"
	"strings"

	"backend/pkg/core"
	"backend/pkg/core/auth"

	"github.com/google/uuid"
)

// Service provides business logic for organizations
type Service struct {
	store        OrganizationStore
	accountStore auth.AccountStore
}

// NewService creates a new organization service
func NewService(store OrganizationStore, accountStore auth.AccountStore) *Service {
	return &Service{
		store:        store,
		accountStore: accountStore,
	}
}

// CreateRequest represents a request to create an organization
type CreateRequest struct {
	Name            string
	Slug            string
	Bio             *string
	LogoURL         *string
	WebsiteURL      *string
	EmailPublic     *string
	SocialLinks     map[string]interface{}
	MetaDescription *string
}

// UpdateRequest represents a request to update an organization
type UpdateRequest struct {
	Name            *string
	Slug            *string
	Bio             *string
	LogoURL         *string
	WebsiteURL      *string
	EmailPublic     *string
	SocialLinks     *map[string]interface{}
	MetaDescription *string
}

// GetByID retrieves an organization by its ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	return s.store.FindByID(ctx, id)
}

// GetBySlug retrieves an organization by its slug
func (s *Service) GetBySlug(ctx context.Context, slug string) (*Organization, error) {
	return s.store.FindBySlug(ctx, slug)
}

// List retrieves all organizations
func (s *Service) List(ctx context.Context) ([]Organization, error) {
	return s.store.List(ctx)
}

// Create creates a new organization
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Organization, error) {
	// Generate slug if not provided
	slug := req.Slug
	if slug == "" {
		slug = generateSlug(req.Name)
	}

	// Check if slug already exists
	existing, err := s.store.FindBySlug(ctx, slug)
	if err != nil && err != core.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, core.ErrAlreadyExists
	}

	org := &Organization{
		ID:              uuid.New(),
		Name:            req.Name,
		Slug:            slug,
		Bio:             req.Bio,
		LogoURL:         req.LogoURL,
		WebsiteURL:      req.WebsiteURL,
		EmailPublic:     req.EmailPublic,
		SocialLinks:     req.SocialLinks,
		MetaDescription: req.MetaDescription,
	}

	if err := s.store.Save(ctx, org); err != nil {
		return nil, err
	}

	return org, nil
}

// Update updates an existing organization
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*Organization, error) {
	org, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if new slug is unique
	if req.Slug != nil && *req.Slug != org.Slug {
		existing, err := s.store.FindBySlug(ctx, *req.Slug)
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
		org.SocialLinks = *req.SocialLinks
	}
	if req.MetaDescription != nil {
		org.MetaDescription = req.MetaDescription
	}

	if err := s.store.Update(ctx, org); err != nil {
		return nil, err
	}

	return org, nil
}

// Delete removes an organization by its ID
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.store.Delete(ctx, id)
}

// JoinOrganization sets the organization for a user account
func (s *Service) JoinOrganization(ctx context.Context, accountID, orgID uuid.UUID) error {
	// Verify organization exists
	_, err := s.store.FindByID(ctx, orgID)
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

// Helper function to generate slug
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
