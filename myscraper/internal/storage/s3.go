package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Uploader uploads a local file to S3. MoneyForward uses this to push the
// scraped CSVs after writing them.
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

// NewClientFromEnv builds an S3 client from the required environment
// variables and returns the client, bucket name, and key prefix. Every
// variable is mandatory; a missing value returns an error naming it.
func NewClientFromEnv(ctx context.Context) (*s3.Client, string, string, error) {
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
			return nil, "", "", fmt.Errorf("storage: %s is required", r.name)
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

// Store is an S3-backed object store targeting a single bucket with a
// fixed key prefix. It serves both file-based uploads (MoneyForward CSVs)
// and raw JSON uploads (SBI assets).
type Store struct {
	client s3ReadWriteClient
	bucket string
	prefix string
}

// New builds a Store from the required environment variables.
func New(ctx context.Context) (*Store, error) {
	client, bucket, prefix, err := NewClientFromEnv(ctx)
	if err != nil {
		return nil, err
	}
	return &Store{client: client, bucket: bucket, prefix: prefix}, nil
}

// NewFromClient builds a Store with an injected client; used by tests.
func NewFromClient(client s3ReadWriteClient, bucket, prefix string) *Store {
	return &Store{client: client, bucket: bucket, prefix: prefix}
}

// KeyFor maps a local file path to its object key, keeping the prefix and
// the file's base name.
func (s *Store) KeyFor(localPath string) string {
	return s.prefix + "/" + filepath.Base(localPath)
}

// Upload reads a local file and puts it under its prefixed key.
func (s *Store) Upload(ctx context.Context, localPath string) error {
	key := s.KeyFor(localPath)
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", localPath, err)
	}
	defer f.Close()
	return s.putObject(ctx, key, "application/octet-stream", f)
}

// PutJSON puts an arbitrary JSON body under the given key with a JSON
// content type. The key is expected to be fully formed (e.g. from
// KeyForTime).
func (s *Store) PutJSON(ctx context.Context, key string, body io.Reader) error {
	return s.putObject(ctx, key, "application/json", body)
}

func (s *Store) putObject(ctx context.Context, key, contentType string, body io.Reader) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      awsv2.String(s.bucket),
		Key:         awsv2.String(key),
		Body:        body,
		ContentType: awsv2.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	return nil
}

// Download gets an object under the prefix and writes it to destPath.
// It writes to a temp file with 0600 perms and renames to destPath on
// success, so a failed or partial download never leaves a corrupt file
// at destPath. destPath's parent directory must already exist.
func (s *Store) Download(ctx context.Context, key, destPath string) error {
	fullKey := s.prefix + "/" + key
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: awsv2.String(s.bucket),
		Key:    awsv2.String(fullKey),
	})
	if err != nil {
		return fmt.Errorf("get %s: %w", fullKey, err)
	}
	defer out.Body.Close()

	dir := filepath.Dir(destPath)
	tmp, err := os.CreateTemp(dir, "myscraper-download-*")
	if err != nil {
		return fmt.Errorf("create temp for %s: %w", destPath, err)
	}
	tmpName := tmp.Name()
	if err := os.Chmod(tmpName, 0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("chmod temp %s: %w", tmpName, err)
	}
	if _, err := io.Copy(tmp, out.Body); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write %s: %w", destPath, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, destPath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename to %s: %w", destPath, err)
	}
	return nil
}

// KeyForTime builds a timestamped object key in JST so each run is kept as
// its own history entry: <prefix>/YYYYMM/YYYYMMDD-HHMMSS.json.
func (s *Store) KeyForTime(now time.Time) string {
	return KeyForTimestamp(s.prefix, now)
}

// KeyForTimestamp builds a timestamped key from an explicit prefix, using
// JST for the date components: <prefix>/YYYY/MM/YYYYMMDD-HHMMSS.json.
func KeyForTimestamp(prefix string, now time.Time) string {
	jst := time.FixedZone("JST", 9*60*60)
	t := now.In(jst)
	return fmt.Sprintf("%s/%s.json", prefix, t.Format("2006/01/20060102-150405"))
}
