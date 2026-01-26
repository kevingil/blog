package auth_test

import (
	"testing"

	"backend/pkg/core/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	t.Run("hashes password successfully", func(t *testing.T) {
		password := "mySecurePassword123"

		hash, err := auth.HashPassword(password)

		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash) // Hash should be different from plain text
	})

	t.Run("different passwords produce different hashes", func(t *testing.T) {
		hash1, err1 := auth.HashPassword("password1")
		hash2, err2 := auth.HashPassword("password2")

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("same password produces different hashes (bcrypt salt)", func(t *testing.T) {
		password := "samePassword"

		hash1, err1 := auth.HashPassword(password)
		hash2, err2 := auth.HashPassword(password)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		// bcrypt adds random salt, so same password produces different hashes
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("handles empty password", func(t *testing.T) {
		hash, err := auth.HashPassword("")

		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
	})
}

func TestComparePasswords(t *testing.T) {
	t.Run("returns true for correct password", func(t *testing.T) {
		password := "correctPassword123"
		hash, _ := auth.HashPassword(password)

		result := auth.ComparePasswords(password, hash)

		assert.True(t, result)
	})

	t.Run("returns false for incorrect password", func(t *testing.T) {
		password := "correctPassword123"
		hash, _ := auth.HashPassword(password)

		result := auth.ComparePasswords("wrongPassword", hash)

		assert.False(t, result)
	})

	t.Run("returns false for empty password against hash", func(t *testing.T) {
		hash, _ := auth.HashPassword("somePassword")

		result := auth.ComparePasswords("", hash)

		assert.False(t, result)
	})

	t.Run("returns false for invalid hash", func(t *testing.T) {
		result := auth.ComparePasswords("anyPassword", "invalidhash")

		assert.False(t, result)
	})

	t.Run("returns false for empty hash", func(t *testing.T) {
		result := auth.ComparePasswords("anyPassword", "")

		assert.False(t, result)
	})
}

func TestGetUserIDFromToken(t *testing.T) {
	t.Run("extracts user ID from valid token claims", func(t *testing.T) {
		userID := uuid.New()
		token := &jwt.Token{
			Valid: true,
			Claims: jwt.MapClaims{
				"sub": userID.String(),
			},
		}

		result, err := auth.GetUserIDFromToken(token)

		assert.NoError(t, err)
		assert.Equal(t, userID, result)
	})

	t.Run("returns error when sub claim is missing", func(t *testing.T) {
		token := &jwt.Token{
			Valid: true,
			Claims: jwt.MapClaims{
				"exp": 12345,
			},
		}

		result, err := auth.GetUserIDFromToken(token)

		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, result)
	})

	t.Run("returns error when sub claim is not a string", func(t *testing.T) {
		token := &jwt.Token{
			Valid: true,
			Claims: jwt.MapClaims{
				"sub": 12345, // Not a string
			},
		}

		result, err := auth.GetUserIDFromToken(token)

		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, result)
	})

	t.Run("returns error when claims are not MapClaims", func(t *testing.T) {
		token := &jwt.Token{
			Valid:  true,
			Claims: nil,
		}

		result, err := auth.GetUserIDFromToken(token)

		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, result)
	})

	t.Run("returns error when sub is invalid UUID", func(t *testing.T) {
		token := &jwt.Token{
			Valid: true,
			Claims: jwt.MapClaims{
				"sub": "not-a-valid-uuid",
			},
		}

		result, err := auth.GetUserIDFromToken(token)

		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, result)
	})
}
