package storage

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type FileData struct {
	Key          string    `json:"key"`
	LastModified time.Time `json:"last_modified"`
	Size         string    `json:"size"`
	SizeRaw      int64     `json:"size_raw"`
	URL          string    `json:"url"`
	IsImage      bool      `json:"is_image"`
}

type FolderData struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	IsHidden     bool      `json:"is_hidden"`
	LastModified time.Time `json:"last_modified"`
	FileCount    int       `json:"file_count"`
}

type StorageService struct {
	s3Client  *s3.Client
	bucket    string
	urlPrefix string
}

func NewStorageService(s3Client *s3.Client, bucket string, urlPrefix string) *StorageService {
	return &StorageService{
		s3Client:  s3Client,
		bucket:    bucket,
		urlPrefix: urlPrefix,
	}
}

func (s *StorageService) ListFiles(ctx context.Context, prefix string) ([]FileData, []FolderData, error) {
	// TODO: Implement S3 list objects
	return nil, nil, nil
}

func (s *StorageService) UploadFile(ctx context.Context, key string, data []byte) error {
	// TODO: Implement S3 upload
	return nil
}

func (s *StorageService) DeleteFile(ctx context.Context, key string) error {
	// TODO: Implement S3 delete
	return nil
}

func (s *StorageService) CreateFolder(ctx context.Context, folderPath string) error {
	// TODO: Implement S3 folder creation
	return nil
}

func (s *StorageService) UpdateFolder(ctx context.Context, oldPath string, newPath string) error {
	// TODO: Implement S3 folder update
	return nil
}

func formatByteSize(size int64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"}
	i := 0
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	return formatFloat(float64(size)) + " " + units[i]
}

func formatFloat(f float64) string {
	return formatFloat64(f, 2)
}

func formatFloat64(f float64, prec int) string {
	// TODO: Implement float formatting
	return ""
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
