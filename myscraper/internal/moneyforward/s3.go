package moneyforward

import (
	"context"

	"github.com/azuki774/myscrapers/myscraper/internal/storage"
)

// Uploader uploads a local file to S3. See storage.Uploader.
type Uploader = storage.Uploader

// S3Store is the S3-backed object store used by MoneyForward. It is a
// thin alias over storage.Store so the shared implementation lives in one
// place.
type S3Store = storage.Store

// NewS3Store builds an S3Store from the required environment variables.
func NewS3Store(ctx context.Context) (*S3Store, error) {
	return storage.New(ctx)
}
