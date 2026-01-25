package organization

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"backend/pkg/core"
	"backend/pkg/database"
	"backend/pkg/database/repository"
	"backend/pkg/types"

	"github.com/google/uuid"
)

// getRepo returns an organization repository instance
func getRepo() *repository.OrganizationRepository {
	return repository.NewOrganizationRepository(database.DB())
}

// getAccountRepo returns an account repository instance
func getAccountRepo() *repository.AccountRepository {
	return repository.NewAccountRepository(database.DB())
}

// GetByID retrieves an organization by its ID
func GetByID(ctx context.Context, id uuid.UUID) (*OrganizationResponse, error) {
	repo := getRepo()
	org, err := repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toResponse(org), nil
}

// GetBySlug retrieves an organization by its slug
func GetBySlug(ctx context.Context, slug string) (*OrganizationResponse, error) {
	repo := getRepo()
	org, err := repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return toResponse(org), nil
}

// List retrieves all organizations
func List(ctx context.Context) ([]OrganizationResponse, error) {
	repo := getRepo()
	orgs, err := repo.List(ctx)
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
func Create(ctx context.Context, req CreateRequest) (*OrganizationResponse, error) {
	repo := getRepo()

	// Generate slug if not provided
	slug := req.Slug
	if slug == "" {
		slug = generateSlug(req.Name)
	}

	// Check if slug already exists
	existing, err := repo.FindBySlug(ctx, slug)
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

	if err := repo.Save(ctx, org); err != nil {
		return nil, err
	}

	return toResponse(org), nil
}

// Update updates an existing organization
func Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*OrganizationResponse, error) {
	repo := getRepo()

	org, err := repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if new slug is unique
	if req.Slug != nil && *req.Slug != org.Slug {
		existing, err := repo.FindBySlug(ctx, *req.Slug)
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

	if err := repo.Update(ctx, org); err != nil {
		return nil, err
	}

	return toResponse(org), nil
}

// Delete removes an organization by its ID
func Delete(ctx context.Context, id uuid.UUID) error {
	repo := getRepo()
	return repo.Delete(ctx, id)
}

// JoinOrganization sets the organization for a user account
func JoinOrganization(ctx context.Context, accountID, orgID uuid.UUID) error {
	orgRepo := getRepo()
	accountRepo := getAccountRepo()

	// Verify organization exists
	_, err := orgRepo.FindByID(ctx, orgID)
	if err != nil {
		return err
	}

	// Get and update account
	account, err := accountRepo.FindByID(ctx, accountID)
	if err != nil {
		return err
	}

	account.OrganizationID = &orgID
	return accountRepo.Update(ctx, account)
}

// LeaveOrganization clears the organization for a user account
func LeaveOrganization(ctx context.Context, accountID uuid.UUID) error {
	accountRepo := getAccountRepo()

	account, err := accountRepo.FindByID(ctx, accountID)
	if err != nil {
		return err
	}

	account.OrganizationID = nil
	return accountRepo.Update(ctx, account)
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
