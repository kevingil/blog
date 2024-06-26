package database

import (
	"testing"

	"github.com/kevingil/blog/pkg/store"
)

func TestCache(t *testing.T) {
	// Testing in memory cache
	cache := store.NewClient()

	var err error
	err = cache.Set("hello", []byte("world"))
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	value, err := cache.Get("hello")
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if string(value) != "world" {
		t.Errorf("Unexpected value retrieved from cache: %s", value)
	}
}
