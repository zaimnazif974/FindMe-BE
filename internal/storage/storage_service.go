package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Service struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
	urlTTL    time.Duration
}

func NewService(client *s3.Client, bucket string, urlTTL time.Duration) *Service {
	return &Service{
		client:    client,
		presigner: s3.NewPresignClient(client),
		bucket:    bucket,
		urlTTL:    urlTTL,
	}
}

func (s *Service) Bucket() string {
	return s.bucket
}

func (s *Service) Upload(ctx context.Context, key, contentType string, size int64, body io.Reader) error {
	if s.bucket == "" {
		return fmt.Errorf("AWS_S3_BUCKET is not configured")
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        String(s.bucket),
		Key:           String(key),
		Body:          body,
		ContentType:   String(contentType),
		ContentLength: &size,
	})
	if err != nil {
		return fmt.Errorf("upload S3 object: %w", err)
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: String(s.bucket), Key: String(key)})
	return err
}

func (s *Service) PresignedDownloadURL(ctx context.Context, key string) (string, error) {
	result, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: String(s.bucket),
		Key:    String(key),
	}, s3.WithPresignExpires(s.urlTTL))
	if err != nil {
		return "", fmt.Errorf("presign S3 object: %w", err)
	}
	return result.URL, nil
}
