// Package storage provides file storage operations
package storage

import (
	"context"
	"sync"

	"backend/pkg/config"
	"backend/pkg/integrations/s3"
)

var (
	client     *s3.Client
	clientOnce sync.Once
)

// getClient returns the S3 client singleton
func getClient() *s3.Client {
	clientOnce.Do(func() {
		cfg := config.Get()
		s3Client := s3.NewR2Client()
		client = s3.NewClient(s3Client, cfg.AWS.S3Bucket, cfg.AWS.S3URLPrefix)
	})
	return client
}

// ListResult contains files and folders from a directory listing
type ListResult struct {
	Files   []s3.FileData   `json:"files"`
	Folders []s3.FolderData `json:"folders"`
}

// ListFiles lists files and folders at the given prefix
func ListFiles(ctx context.Context, prefix string) (*ListResult, error) {
	c := getClient()
	files, folders, err := c.ListFiles(ctx, prefix)
	if err != nil {
		return nil, err
	}
	return &ListResult{
		Files:   files,
		Folders: folders,
	}, nil
}

// UploadFile uploads a file to storage
func UploadFile(ctx context.Context, key string, data []byte) error {
	c := getClient()
	return c.UploadFile(ctx, key, data)
}

// DeleteFile deletes a file from storage
func DeleteFile(ctx context.Context, key string) error {
	c := getClient()
	return c.DeleteFile(ctx, key)
}

// CreateFolder creates a folder in storage
func CreateFolder(ctx context.Context, path string) error {
	c := getClient()
	return c.CreateFolder(ctx, path)
}

// UpdateFolder renames/moves a folder in storage
func UpdateFolder(ctx context.Context, oldPath, newPath string) error {
	c := getClient()
	return c.UpdateFolder(ctx, oldPath, newPath)
}

// GetURLPrefix returns the URL prefix for constructing file URLs
func GetURLPrefix() string {
	c := getClient()
	return c.GetURLPrefix()
}
