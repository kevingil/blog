package validation_test

import (
	"testing"

	"backend/pkg/api/validation"
	"backend/pkg/core"

	"github.com/stretchr/testify/assert"
)

// Test structs for validation
type TestLoginRequest struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required,min=6"`
}

type TestRegisterRequest struct {
	Name     string `validate:"required,min=2,max=100"`
	Email    string `validate:"required,email"`
	Password string `validate:"required,min=8,max=128"`
}

type TestSlugRequest struct {
	Slug string `validate:"required,slug"`
}

type TestOptionalSlugRequest struct {
	Slug string `validate:"omitempty,slug"`
}

type TestURLRequest struct {
	Website string `validate:"required,url"`
}

type TestOneOfRequest struct {
	Status string `validate:"required,oneof=draft published archived"`
}

func TestValidateStruct_Required(t *testing.T) {
	t.Run("returns error when required field is empty", func(t *testing.T) {
		req := TestLoginRequest{
			Email:    "",
			Password: "password123",
		}

		err := validation.ValidateStruct(req)

		assert.Error(t, err)
		validationErrs, ok := err.(core.ValidationErrors)
		assert.True(t, ok)
		assert.Len(t, validationErrs, 1)
		assert.Equal(t, "Email", validationErrs[0].Field)
		assert.Contains(t, validationErrs[0].Message, "required")
	})

	t.Run("returns multiple errors for multiple missing fields", func(t *testing.T) {
		req := TestLoginRequest{
			Email:    "",
			Password: "",
		}

		err := validation.ValidateStruct(req)

		assert.Error(t, err)
		validationErrs, ok := err.(core.ValidationErrors)
		assert.True(t, ok)
		assert.Len(t, validationErrs, 2)
	})

	t.Run("returns nil for valid struct", func(t *testing.T) {
		req := TestLoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		err := validation.ValidateStruct(req)

		assert.NoError(t, err)
	})
}

func TestValidateStruct_Email(t *testing.T) {
	t.Run("accepts valid email", func(t *testing.T) {
		req := TestLoginRequest{
			Email:    "user@example.com",
			Password: "password123",
		}

		err := validation.ValidateStruct(req)

		assert.NoError(t, err)
	})

	t.Run("accepts email with subdomain", func(t *testing.T) {
		req := TestLoginRequest{
			Email:    "user@mail.example.com",
			Password: "password123",
		}

		err := validation.ValidateStruct(req)

		assert.NoError(t, err)
	})

	t.Run("rejects invalid email without @", func(t *testing.T) {
		req := TestLoginRequest{
			Email:    "invalid-email",
			Password: "password123",
		}

		err := validation.ValidateStruct(req)

		assert.Error(t, err)
		validationErrs, ok := err.(core.ValidationErrors)
		assert.True(t, ok)
		assert.Contains(t, validationErrs[0].Message, "email")
	})

	t.Run("rejects invalid email without domain", func(t *testing.T) {
		req := TestLoginRequest{
			Email:    "user@",
			Password: "password123",
		}

		err := validation.ValidateStruct(req)

		assert.Error(t, err)
	})
}

func TestValidateStruct_MinMax(t *testing.T) {
	t.Run("accepts value at minimum length", func(t *testing.T) {
		req := TestLoginRequest{
			Email:    "test@example.com",
			Password: "123456", // Exactly 6 characters
		}

		err := validation.ValidateStruct(req)

		assert.NoError(t, err)
	})

	t.Run("rejects value below minimum length", func(t *testing.T) {
		req := TestLoginRequest{
			Email:    "test@example.com",
			Password: "12345", // 5 characters, needs 6
		}

		err := validation.ValidateStruct(req)

		assert.Error(t, err)
		validationErrs, ok := err.(core.ValidationErrors)
		assert.True(t, ok)
		assert.Contains(t, validationErrs[0].Message, "at least")
	})

	t.Run("rejects value above maximum length", func(t *testing.T) {
		req := TestRegisterRequest{
			Name:     "A very long name that exceeds the maximum allowed length for the name field which is one hundred characters total",
			Email:    "test@example.com",
			Password: "password123",
		}

		err := validation.ValidateStruct(req)

		assert.Error(t, err)
		validationErrs, ok := err.(core.ValidationErrors)
		assert.True(t, ok)
		assert.Contains(t, validationErrs[0].Message, "at most")
	})
}

func TestValidateStruct_Slug(t *testing.T) {
	t.Run("accepts valid slug", func(t *testing.T) {
		req := TestSlugRequest{
			Slug: "my-blog-post",
		}

		err := validation.ValidateStruct(req)

		assert.NoError(t, err)
	})

	t.Run("accepts slug with numbers", func(t *testing.T) {
		req := TestSlugRequest{
			Slug: "post-123",
		}

		err := validation.ValidateStruct(req)

		assert.NoError(t, err)
	})

	t.Run("accepts single word slug", func(t *testing.T) {
		req := TestSlugRequest{
			Slug: "hello",
		}

		err := validation.ValidateStruct(req)

		assert.NoError(t, err)
	})

	t.Run("rejects slug with uppercase", func(t *testing.T) {
		req := TestSlugRequest{
			Slug: "My-Blog-Post",
		}

		err := validation.ValidateStruct(req)

		assert.Error(t, err)
		validationErrs, ok := err.(core.ValidationErrors)
		assert.True(t, ok)
		assert.Contains(t, validationErrs[0].Message, "slug")
	})

	t.Run("rejects slug with spaces", func(t *testing.T) {
		req := TestSlugRequest{
			Slug: "my blog post",
		}

		err := validation.ValidateStruct(req)

		assert.Error(t, err)
	})

	t.Run("rejects slug with special characters", func(t *testing.T) {
		req := TestSlugRequest{
			Slug: "my_blog_post",
		}

		err := validation.ValidateStruct(req)

		assert.Error(t, err)
	})

	t.Run("rejects slug starting with hyphen", func(t *testing.T) {
		req := TestSlugRequest{
			Slug: "-my-blog",
		}

		err := validation.ValidateStruct(req)

		assert.Error(t, err)
	})

	t.Run("rejects slug ending with hyphen", func(t *testing.T) {
		req := TestSlugRequest{
			Slug: "my-blog-",
		}

		err := validation.ValidateStruct(req)

		assert.Error(t, err)
	})

	t.Run("rejects slug with consecutive hyphens", func(t *testing.T) {
		req := TestSlugRequest{
			Slug: "my--blog",
		}

		err := validation.ValidateStruct(req)

		assert.Error(t, err)
	})

	t.Run("allows empty slug when optional", func(t *testing.T) {
		req := TestOptionalSlugRequest{
			Slug: "",
		}

		err := validation.ValidateStruct(req)

		assert.NoError(t, err)
	})
}

func TestValidateStruct_URL(t *testing.T) {
	t.Run("accepts valid HTTP URL", func(t *testing.T) {
		req := TestURLRequest{
			Website: "http://example.com",
		}

		err := validation.ValidateStruct(req)

		assert.NoError(t, err)
	})

	t.Run("accepts valid HTTPS URL", func(t *testing.T) {
		req := TestURLRequest{
			Website: "https://example.com/path?query=value",
		}

		err := validation.ValidateStruct(req)

		assert.NoError(t, err)
	})

	t.Run("rejects invalid URL", func(t *testing.T) {
		req := TestURLRequest{
			Website: "not-a-url",
		}

		err := validation.ValidateStruct(req)

		assert.Error(t, err)
		validationErrs, ok := err.(core.ValidationErrors)
		assert.True(t, ok)
		assert.Contains(t, validationErrs[0].Message, "URL")
	})
}

func TestValidateStruct_OneOf(t *testing.T) {
	t.Run("accepts valid oneof value", func(t *testing.T) {
		req := TestOneOfRequest{
			Status: "draft",
		}

		err := validation.ValidateStruct(req)

		assert.NoError(t, err)
	})

	t.Run("accepts another valid oneof value", func(t *testing.T) {
		req := TestOneOfRequest{
			Status: "published",
		}

		err := validation.ValidateStruct(req)

		assert.NoError(t, err)
	})

	t.Run("rejects invalid oneof value", func(t *testing.T) {
		req := TestOneOfRequest{
			Status: "invalid",
		}

		err := validation.ValidateStruct(req)

		assert.Error(t, err)
		validationErrs, ok := err.(core.ValidationErrors)
		assert.True(t, ok)
		assert.Contains(t, validationErrs[0].Message, "one of")
	})
}
