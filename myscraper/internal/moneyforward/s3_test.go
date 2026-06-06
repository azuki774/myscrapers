package moneyforward

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestNewS3UploaderRequiresAllEnvVars(t *testing.T) {
	full := map[string]string{
		"BUCKET_URL":            "https://example.test",
		"BUCKET_NAME":           "bucket",
		"BUCKET_DIR":            "prefix",
		"AWS_REGION":            "us-east-1",
		"AWS_ACCESS_KEY_ID":     "AKIA",
		"AWS_SECRET_ACCESS_KEY": "SECRET",
	}
	names := []string{
		"BUCKET_URL",
		"BUCKET_NAME",
		"BUCKET_DIR",
		"AWS_REGION",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
	}
	for _, missing := range names {
		missing := missing
		t.Run("missing_"+missing, func(t *testing.T) {
			for k, v := range full {
				if k == missing {
					t.Setenv(k, "")
				} else {
					t.Setenv(k, v)
				}
			}
			_, err := NewS3Uploader(context.Background())
			if err == nil {
				t.Fatalf("NewS3Uploader() error = nil, want error naming %s", missing)
			}
			if !strings.Contains(err.Error(), missing) {
				t.Fatalf("NewS3Uploader() error = %q, want it to name %s", err.Error(), missing)
			}
		})
	}
}

func TestS3UploaderKeyFor(t *testing.T) {
	u := &S3Uploader{prefix: "myscrapers/moneyforward"}
	cases := []struct {
		in   string
		want string
	}{
		{"/data/cf.csv", "myscrapers/moneyforward/cf.csv"},
		{"cf_lastmonth.csv", "myscrapers/moneyforward/cf_lastmonth.csv"},
		{"/tmp/nested/dir/asset_history.csv", "myscrapers/moneyforward/asset_history.csv"},
	}
	for _, tc := range cases {
		if got := u.KeyFor(tc.in); got != tc.want {
			t.Fatalf("KeyFor(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

type recordingS3Client struct {
	input *s3.PutObjectInput
	body  []byte
	err   error
}

func (r *recordingS3Client) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	r.input = input
	if input.Body != nil {
		b, err := io.ReadAll(input.Body)
		if err != nil {
			return nil, err
		}
		r.body = b
	}
	return &s3.PutObjectOutput{}, r.err
}

func TestS3UploaderUpload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cf.csv")
	want := []byte("date,amount\n2024/12/09,-110\n")
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	rec := &recordingS3Client{}
	u := &S3Uploader{client: rec, bucket: "my-bucket", prefix: "myscrapers/moneyforward"}

	if err := u.Upload(context.Background(), path); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if rec.input == nil {
		t.Fatalf("PutObject not called")
	}
	if rec.input.Bucket == nil || *rec.input.Bucket != "my-bucket" {
		t.Fatalf("Bucket = %v, want %q", rec.input.Bucket, "my-bucket")
	}
	if rec.input.Key == nil || *rec.input.Key != "myscrapers/moneyforward/cf.csv" {
		t.Fatalf("Key = %v, want %q", rec.input.Key, "myscrapers/moneyforward/cf.csv")
	}
	if rec.input.Body == nil {
		t.Fatalf("Body = nil, want non-nil io.Reader")
	}
	if string(rec.body) != string(want) {
		t.Fatalf("body = %q, want %q", string(rec.body), string(want))
	}
}
