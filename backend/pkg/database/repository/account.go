package repository

import (
	"context"

	"backend/pkg/core"
	"backend/pkg/core/auth"
	"backend/pkg/database/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AccountRepository implements auth.AccountStore using GORM
type AccountRepository struct {
	db *gorm.DB
}

// NewAccountRepository creates a new AccountRepository
func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// FindByID retrieves an account by its ID
func (r *AccountRepository) FindByID(ctx context.Context, id uuid.UUID) (*auth.Account, error) {
	var model models.Account
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return model.ToCore(), nil
}

// FindByEmail retrieves an account by its email
func (r *AccountRepository) FindByEmail(ctx context.Context, email string) (*auth.Account, error) {
	var model models.Account
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return model.ToCore(), nil
}

// Save creates a new account
func (r *AccountRepository) Save(ctx context.Context, a *auth.Account) error {
	model := models.AccountFromCore(a)

	if a.ID == uuid.Nil {
		a.ID = uuid.New()
		model.ID = a.ID
	}

	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing account
func (r *AccountRepository) Update(ctx context.Context, a *auth.Account) error {
	model := models.AccountFromCore(a)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete removes an account by its ID
func (r *AccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Account{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}
