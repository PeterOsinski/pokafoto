package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/drive/drive/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type StorageService struct {
	cfg     *config.Config
	s3      *minio.Client
	client  *minio.Client
}

func NewStorageService(cfg *config.Config) (*StorageService, error) {
	s := &StorageService{cfg: cfg}

	if !cfg.Storage.S3.Enabled {
		return s, nil
	}

	client, err := minio.New(cfg.Storage.S3.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Storage.S3.AccessKey, cfg.Storage.S3.SecretKey, ""),
		Secure: cfg.Storage.S3.UseSSL,
		Region: cfg.Storage.S3.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}

	s.client = client

	exists, err := client.BucketExists(context.Background(), cfg.Storage.S3.Bucket)
	if err != nil {
		return s, fmt.Errorf("check s3 bucket: %w", err)
	}
	if !exists {
		return s, fmt.Errorf("s3 bucket %s does not exist", cfg.Storage.S3.Bucket)
	}

	return s, nil
}

func (s *StorageService) PutObject(key string, filePath string) error {
	if s.client == nil {
		return nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file for s3 upload: %w", err)
	}
	defer f.Close()

	stat, _ := f.Stat()
	_, err = s.client.PutObject(context.Background(), s.cfg.Storage.S3.Bucket, key, f, stat.Size(), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("s3 put object: %w", err)
	}

	return nil
}

func (s *StorageService) GetObject(key string, destPath string) error {
	if s.client == nil {
		return fmt.Errorf("s3 not configured")
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	obj, err := s.client.GetObject(context.Background(), s.cfg.Storage.S3.Bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("s3 get object: %w", err)
	}
	defer obj.Close()

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, obj); err != nil {
		os.Remove(destPath)
		return fmt.Errorf("s3 download: %w", err)
	}

	return nil
}

func (s *StorageService) PutOriginals(userID, filename string, filePath string) error {
	key := fmt.Sprintf("originals/%s/%s", userID, filename)
	return s.PutObject(key, filePath)
}

func (s *StorageService) PutThumbnail(fileID, size, format string, filePath string) error {
	key := fmt.Sprintf("thumbnails/%s/%s.%s", fileID, size, format)
	return s.PutObject(key, filePath)
}

func (s *StorageService) IsConnected() bool {
	return s.client != nil
}
