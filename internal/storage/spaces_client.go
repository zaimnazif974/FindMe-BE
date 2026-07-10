package storage

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func NewSpacesClient(ctx context.Context, region, endpoint, accessKey, secretKey string) (*s3.Client, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("SPACES_ENDPOINT is not configured")
	}
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("SPACES_ACCESS_KEY_ID and SPACES_SECRET_ACCESS_KEY must be configured")
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load DigitalOcean Spaces config: %w", err)
	}

	return s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = false
	}), nil
}

func String(value string) *string {
	return aws.String(value)
}
