package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestNewClientFromEnvRequiresAllEnvVars(t *testing.T) {
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
			_, err := New(context.Background())
			if err == nil {
				t.Fatalf("New() error = nil, want error naming %s", missing)
			}
			if !strings.Contains(err.Error(), missing) {
				t.Fatalf("New() error = %q, want it to name %s", err.Error(), missing)
			}
		})
	}
}

func TestStoreKeyFor(t *testing.T) {
	s := &Store{prefix: "myscrapers/moneyforward"}
	cases := []struct {
		in   string
		want string
	}{
		{"/data/cf.csv", "myscrapers/moneyforward/cf.csv"},
		{"cf_lastmonth.csv", "myscrapers/moneyforward/cf_lastmonth.csv"},
		{"/tmp/nested/dir/asset_history.csv", "myscrapers/moneyforward/asset_history.csv"},
	}
	for _, tc := range cases {
		if got := s.KeyFor(tc.in); got != tc.want {
			t.Fatalf("KeyFor(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestStoreKeyForTime(t *testing.T) {
	s := &Store{prefix: "myscrapers/sbi"}
	now := time.Date(2024, 12, 9, 1, 2, 3, 0, time.UTC)
	if got := s.KeyForTime(now); got != "myscrapers/sbi/2024/12/20241209-100203.json" {
		t.Fatalf("KeyForTime() = %q, want %q", got, "myscrapers/sbi/2024/12/20241209-100203.json")
	}
}

type recordingPutClient struct {
	input *s3.PutObjectInput
	body  []byte
	ctype string
	err   error
}

func (r *recordingPutClient) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	r.input = input
	if input.ContentType != nil {
		r.ctype = *input.ContentType
	}
	if input.Body != nil {
		b, err := io.ReadAll(input.Body)
		if err != nil {
			return nil, err
		}
		r.body = b
	}
	return &s3.PutObjectOutput{}, r.err
}

func (r *recordingPutClient) GetObject(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return nil, nil
}

func TestStoreUpload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cf.csv")
	want := []byte("date,amount\n2024/12/09,-110\n")
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	rec := &recordingPutClient{}
	s := NewFromClient(rec, "my-bucket", "myscrapers/moneyforward")

	if err := s.Upload(context.Background(), path); err != nil {
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
	if rec.ctype != "application/octet-stream" {
		t.Fatalf("ContentType = %q, want %q", rec.ctype, "application/octet-stream")
	}
	if string(rec.body) != string(want) {
		t.Fatalf("body = %q, want %q", string(rec.body), string(want))
	}
}

func TestStorePutJSON(t *testing.T) {
	body := []byte(`{"schema_version":1}`)
	rec := &recordingPutClient{}
	s := NewFromClient(rec, "my-bucket", "myscrapers/sbi")

	if err := s.PutJSON(context.Background(), "myscrapers/sbi/202412/20241209-100203.json", bytes.NewReader(body)); err != nil {
		t.Fatalf("PutJSON() error = %v", err)
	}
	if rec.input == nil {
		t.Fatalf("PutObject not called")
	}
	if rec.input.Key == nil || *rec.input.Key != "myscrapers/sbi/202412/20241209-100203.json" {
		t.Fatalf("Key = %v, want %q", rec.input.Key, "myscrapers/sbi/202412/20241209-100203.json")
	}
	if rec.ctype != "application/json" {
		t.Fatalf("ContentType = %q, want %q", rec.ctype, "application/json")
	}
	if string(rec.body) != string(body) {
		t.Fatalf("body = %q, want %q", string(rec.body), string(body))
	}
}

type recordingGetClient struct {
	input *s3.GetObjectInput
	body  io.ReadCloser
	err   error
}

func (r *recordingGetClient) GetObject(_ context.Context, input *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	r.input = input
	return &s3.GetObjectOutput{Body: r.body}, r.err
}

func (r *recordingGetClient) PutObject(_ context.Context, _ *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return nil, nil
}

func TestStoreDownload(t *testing.T) {
	want := []byte(`[{"name":"sid","value":"abc","domain":".moneyforward.com"}]`)
	rec := &recordingGetClient{body: io.NopCloser(bytes.NewReader(want))}
	store := NewFromClient(rec, "my-bucket", "myscrapers/moneyforward")

	dir := t.TempDir()
	dest := filepath.Join(dir, "cookie.json")
	if err := store.Download(context.Background(), "cookie.json", dest); err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if rec.input == nil {
		t.Fatalf("GetObject not called")
	}
	if rec.input.Bucket == nil || *rec.input.Bucket != "my-bucket" {
		t.Fatalf("Bucket = %v, want %q", rec.input.Bucket, "my-bucket")
	}
	if rec.input.Key == nil || *rec.input.Key != "myscrapers/moneyforward/cookie.json" {
		t.Fatalf("Key = %v, want %q", rec.input.Key, "myscrapers/moneyforward/cookie.json")
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("body = %q, want %q", string(got), string(want))
	}
}
