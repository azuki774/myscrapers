package moneyforward

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Uploader interface {
	Upload(ctx context.Context, localPath string) error
}

type s3PutObjectClient interface {
	PutObject(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type s3GetObjectClient interface {
	GetObject(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type s3ReadWriteClient interface {
	s3PutObjectClient
	s3GetObjectClient
}

func newS3ClientFromEnv(ctx context.Context) (*s3.Client, string, string, error) {
	required := []struct {
		name string
		val  string
	}{
		{"BUCKET_URL", os.Getenv("BUCKET_URL")},
		{"BUCKET_NAME", os.Getenv("BUCKET_NAME")},
		{"BUCKET_DIR", os.Getenv("BUCKET_DIR")},
		{"AWS_REGION", os.Getenv("AWS_REGION")},
		{"AWS_ACCESS_KEY_ID", os.Getenv("AWS_ACCESS_KEY_ID")},
		{"AWS_SECRET_ACCESS_KEY", os.Getenv("AWS_SECRET_ACCESS_KEY")},
	}
	for _, r := range required {
		if r.val == "" {
			return nil, "", "", fmt.Errorf("S3Store: %s is required", r.name)
		}
	}
	endpoint := required[0].val
	bucket := required[1].val
	prefix := required[2].val
	region := required[3].val
	accessKey := required[4].val
	secretKey := required[5].val
	cfg, err := awscfg.LoadDefaultConfig(ctx,
		awscfg.WithRegion(region),
		awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, "", "", fmt.Errorf("load aws config: %w", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = awsv2.String(endpoint)
		o.UsePathStyle = true
	})
	return client, bucket, prefix, nil
}

type S3Store struct {
	client s3ReadWriteClient
	bucket string
	prefix string
}

func NewS3Store(ctx context.Context) (*S3Store, error) {
	client, bucket, prefix, err := newS3ClientFromEnv(ctx)
	if err != nil {
		return nil, err
	}
	return &S3Store{client: client, bucket: bucket, prefix: prefix}, nil
}

func (s *S3Store) KeyFor(localPath string) string {
	return s.prefix + "/" + filepath.Base(localPath)
}

func (s *S3Store) Upload(ctx context.Context, localPath string) error {
	key := s.KeyFor(localPath)
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", localPath, err)
	}
	defer f.Close()
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: awsv2.String(s.bucket),
		Key:    awsv2.String(key),
		Body:   f,
	})
	if err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	return nil
}

func (s *S3Store) Download(ctx context.Context, key, destPath string) error {
	fullKey := s.prefix + "/" + key
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: awsv2.String(s.bucket),
		Key:    awsv2.String(fullKey),
	})
	if err != nil {
		return fmt.Errorf("get %s: %w", fullKey, err)
	}
	defer out.Body.Close()
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", destPath, err)
	}
	if _, err := io.Copy(f, out.Body); err != nil {
		f.Close()
		return fmt.Errorf("write %s: %w", destPath, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %s: %w", destPath, err)
	}
	return nil
}
