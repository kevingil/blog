// Package s3 provides a client for S3-compatible storage (including Cloudflare R2)
package s3

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// FileData represents a file in storage
type FileData struct {
	Key          string    `json:"key"`
	LastModified time.Time `json:"last_modified"`
	Size         string    `json:"size"`
	SizeRaw      int64     `json:"size_raw"`
	URL          string    `json:"url"`
	IsImage      bool      `json:"is_image"`
}

// FolderData represents a folder in storage
type FolderData struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	IsHidden     bool      `json:"is_hidden"`
	LastModified time.Time `json:"last_modified"`
	FileCount    int       `json:"file_count"`
}

// Client wraps the S3 client with convenience methods
type Client struct {
	s3Client  *s3.Client
	bucket    string
	urlPrefix string
}

// NewClient creates a new S3 client
func NewClient(s3Client *s3.Client, bucket string, urlPrefix string) *Client {
	return &Client{
		s3Client:  s3Client,
		bucket:    bucket,
		urlPrefix: urlPrefix,
	}
}

// ListFiles lists files and folders at the given prefix
func (c *Client) ListFiles(ctx context.Context, prefix string) ([]FileData, []FolderData, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:    &c.bucket,
		Prefix:    &prefix,
		Delimiter: aws.String("/"),
	}

	result, err := c.s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, nil, err
	}

	files := make([]FileData, 0, len(result.Contents))
	for _, item := range result.Contents {
		size := int64(0)
		if item.Size != nil {
			size = *item.Size
		}
		files = append(files, FileData{
			Key:          *item.Key,
			LastModified: *item.LastModified,
			Size:         formatByteSize(size),
			SizeRaw:      size,
			URL:          c.urlPrefix + "/" + *item.Key,
			IsImage:      isImageFile(*item.Key),
		})
	}

	folders := make([]FolderData, 0, len(result.CommonPrefixes))
	for _, prefix := range result.CommonPrefixes {
		path := *prefix.Prefix
		name := path
		if len(path) > 0 && path[len(path)-1] == '/' {
			name = path[:len(path)-1]
		}
		if lastSlash := strings.LastIndex(name, "/"); lastSlash != -1 {
			name = name[lastSlash+1:]
		}

		folders = append(folders, FolderData{
			Name:         name,
			Path:         path,
			IsHidden:     folderIsHidden(name),
			LastModified: time.Now(),
			FileCount:    0,
		})
	}

	return files, folders, nil
}

// UploadFile uploads a file to storage
func (c *Client) UploadFile(ctx context.Context, key string, data []byte) error {
	input := &s3.PutObjectInput{
		Bucket: &c.bucket,
		Key:    &key,
		Body:   bytes.NewReader(data),
	}

	_, err := c.s3Client.PutObject(ctx, input)
	return err
}

// DeleteFile deletes a file from storage
func (c *Client) DeleteFile(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: &c.bucket,
		Key:    &key,
	}

	_, err := c.s3Client.DeleteObject(ctx, input)
	return err
}

// CreateFolder creates a folder (empty object with trailing slash)
func (c *Client) CreateFolder(ctx context.Context, folderPath string) error {
	if !strings.HasSuffix(folderPath, "/") {
		folderPath += "/"
	}

	input := &s3.PutObjectInput{
		Bucket: &c.bucket,
		Key:    &folderPath,
		Body:   bytes.NewReader([]byte{}),
	}

	_, err := c.s3Client.PutObject(ctx, input)
	return err
}

// UpdateFolder renames/moves a folder
func (c *Client) UpdateFolder(ctx context.Context, oldPath string, newPath string) error {
	// List all objects in the old path
	listInput := &s3.ListObjectsV2Input{
		Bucket: &c.bucket,
		Prefix: &oldPath,
	}

	result, err := c.s3Client.ListObjectsV2(ctx, listInput)
	if err != nil {
		return err
	}

	// Copy each object to the new path and delete the old one
	for _, item := range result.Contents {
		newKey := strings.Replace(*item.Key, oldPath, newPath, 1)

		// Copy object
		copyInput := &s3.CopyObjectInput{
			Bucket:     &c.bucket,
			CopySource: aws.String(c.bucket + "/" + *item.Key),
			Key:        &newKey,
		}

		_, err := c.s3Client.CopyObject(ctx, copyInput)
		if err != nil {
			return err
		}

		// Delete old object
		deleteInput := &s3.DeleteObjectInput{
			Bucket: &c.bucket,
			Key:    item.Key,
		}

		_, err = c.s3Client.DeleteObject(ctx, deleteInput)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetURLPrefix returns the URL prefix for constructing file URLs
func (c *Client) GetURLPrefix() string {
	return c.urlPrefix
}

// NewR2Client creates an S3 client configured for Cloudflare R2
func NewR2Client() *s3.Client {
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKey := os.Getenv("S3_ACCESS_KEY_ID")
	secretKey := os.Getenv("S3_ACCESS_KEY_SECRET")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		fmt.Println("Error loading config:", err)
		panic(err)
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true // Required for R2
	})
}

// Helper functions

func formatByteSize(size int64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"}
	i := 0
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	return strconv.FormatFloat(float64(size), 'f', 2, 64) + " " + units[i]
}

func isImageFile(key string) bool {
	imageExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"}
	for _, ext := range imageExtensions {
		if len(key) >= len(ext) && key[len(key)-len(ext):] == ext {
			return true
		}
	}
	return false
}

func folderIsHidden(folderName string) bool {
	return len(folderName) > 0 && folderName[0] == '.'
}
