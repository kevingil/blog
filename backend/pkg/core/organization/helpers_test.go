package organization

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSlug(t *testing.T) {
	t.Run("converts simple name to slug", func(t *testing.T) {
		result := generateSlug("My Company")

		assert.Equal(t, "my-company", result)
	})

	t.Run("converts to lowercase", func(t *testing.T) {
		result := generateSlug("UPPERCASE")

		assert.Equal(t, "uppercase", result)
	})

	t.Run("replaces spaces with hyphens", func(t *testing.T) {
		result := generateSlug("hello world test")

		assert.Equal(t, "hello-world-test", result)
	})

	t.Run("removes special characters", func(t *testing.T) {
		result := generateSlug("Company! @#$%^&*()")

		assert.Equal(t, "company", result)
	})

	t.Run("handles multiple spaces", func(t *testing.T) {
		result := generateSlug("hello    world")

		assert.Equal(t, "hello-world", result)
	})

	t.Run("handles leading and trailing spaces", func(t *testing.T) {
		result := generateSlug("  hello world  ")

		assert.Equal(t, "hello-world", result)
	})

	t.Run("handles numbers", func(t *testing.T) {
		result := generateSlug("Company 123")

		assert.Equal(t, "company-123", result)
	})

	t.Run("handles unicode characters", func(t *testing.T) {
		result := generateSlug("Café Münich")

		// Unicode characters are removed by the regex
		assert.Equal(t, "caf-mnich", result)
	})

	t.Run("handles empty string", func(t *testing.T) {
		result := generateSlug("")

		assert.Equal(t, "", result)
	})

	t.Run("handles string with only special characters", func(t *testing.T) {
		result := generateSlug("!@#$%^&*()")

		assert.Equal(t, "", result)
	})

	t.Run("collapses multiple hyphens", func(t *testing.T) {
		result := generateSlug("hello - - world")

		assert.Equal(t, "hello-world", result)
	})

	t.Run("removes leading hyphens", func(t *testing.T) {
		result := generateSlug("-hello")

		assert.Equal(t, "hello", result)
	})

	t.Run("removes trailing hyphens", func(t *testing.T) {
		result := generateSlug("hello-")

		assert.Equal(t, "hello", result)
	})
}

func TestStringValue(t *testing.T) {
	t.Run("returns string value when pointer is not nil", func(t *testing.T) {
		value := "hello"
		ptr := &value

		result := stringValue(ptr)

		assert.Equal(t, "hello", result)
	})

	t.Run("returns empty string when pointer is nil", func(t *testing.T) {
		var ptr *string = nil

		result := stringValue(ptr)

		assert.Equal(t, "", result)
	})

	t.Run("returns empty string for pointer to empty string", func(t *testing.T) {
		value := ""
		ptr := &value

		result := stringValue(ptr)

		assert.Equal(t, "", result)
	})

	t.Run("handles pointer to string with whitespace", func(t *testing.T) {
		value := "  hello world  "
		ptr := &value

		result := stringValue(ptr)

		assert.Equal(t, "  hello world  ", result)
	})
}
