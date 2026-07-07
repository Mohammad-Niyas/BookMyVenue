package s3

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bookmyvenue/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type S3Client interface {
	GeneratePresignedURL(ctx context.Context, fileName string, contentType string) (uploadURL string, downloadURL string, err error)
}

type s3Client struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucketName    string
	region        string
}

func NewS3Client(cfg *config.Config) (S3Client, error) {
	if cfg.AWSAccessKeyID == "" || cfg.AWSSecretAccessKey == "" || cfg.AWSS3Bucket == "" {
		return nil, errors.New("missing AWS S3 credentials in configuration (.env)")
	}

	creds := credentials.NewStaticCredentialsProvider(cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey, "")
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.AWSRegion),
		awsconfig.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)
	presignClient := s3.NewPresignClient(client)

	return &s3Client{
		client:        client,
		presignClient: presignClient,
		bucketName:    cfg.AWSS3Bucket,
		region:        cfg.AWSRegion,
	}, nil
}

func (s *s3Client) GeneratePresignedURL(ctx context.Context, fileName string, contentType string) (string, string, error) {
	uniqueKey := "temp/" + uuid.New().String() + "_" + fileName

	req, err := s.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(uniqueKey),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(15*time.Minute))

	if err != nil {
		return "", "", fmt.Errorf("failed to generate presigned PUT url: %w", err)
	}

	downloadURL := "https://" + s.bucketName + ".s3." + s.region + ".amazonaws.com/" + uniqueKey
	return req.URL, downloadURL, nil
}