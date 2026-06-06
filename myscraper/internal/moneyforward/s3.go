package moneyforward

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Uploader is the minimum interface the Fetch orchestrator needs to
// ship a CSV file off-host. The production implementation is
// S3Uploader; tests use a fake that records the keys it was asked to
// upload.
type Uploader interface {
	Upload(ctx context.Context, localPath string) error
}

// s3PutObjectClient is the subset of *s3.Client that S3Uploader needs.
// The concrete *s3.Client from aws-sdk-go-v2 satisfies it via Go's
// structural typing, and tests inject a fake to assert on the
// PutObjectInput without contacting AWS.
type s3PutObjectClient interface {
	PutObject(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// S3Uploader uploads each local file to "<prefix>/<basename>" on the
// configured bucket via the supplied S3 client. The bucket and prefix
// are captured at construction time so the Fetch orchestrator only
// needs to know about Uploader.
type S3Uploader struct {
	client s3PutObjectClient
	bucket string
	prefix string
}

// NewS3Uploader reads the AWS_* and BUCKET_* environment variables and
// returns an S3Uploader ready to upload to the configured endpoint.
// Required env vars: BUCKET_URL, BUCKET_NAME, BUCKET_DIR,
// AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION. The returned
// error names the first missing variable so callers can fix env
// configuration without trial and error.
func NewS3Uploader(ctx context.Context) (*S3Uploader, error) {
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
			return nil, fmt.Errorf("S3Uploader: %s is required", r.name)
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
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = awsv2.String(endpoint)
		o.UsePathStyle = true
	})
	return &S3Uploader{client: client, bucket: bucket, prefix: prefix}, nil
}

// KeyFor returns the S3 object key for a local file path. The key is
// "<prefix>/<basename>" so callers know the upload destination without
// reading the file. The base name is computed with filepath.Base which
// strips any directory component.
func (u *S3Uploader) KeyFor(localPath string) string {
	return u.prefix + "/" + filepath.Base(localPath)
}

// Upload sends a single file to "<prefix>/<basename>" on the bucket.
func (u *S3Uploader) Upload(ctx context.Context, localPath string) error {
	key := u.KeyFor(localPath)
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", localPath, err)
	}
	defer f.Close()
	_, err = u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: awsv2.String(u.bucket),
		Key:    awsv2.String(key),
		Body:   f,
	})
	if err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	return nil
}
