package foundation

import (
	"context"

	"cloud.google.com/go/storage"
)

// GCSClientInterface permite inyectar un mock para tests
type GCSClientInterface interface {
	Bucket(name string) GCSBucketInterface
	Close() error
}

// GCSBucketInterface representa un bucket de GCS
type GCSBucketInterface interface {
	Object(name string) GCSObjectInterface
	Objects(ctx context.Context, query *storage.Query) GCSObjectIteratorInterface
}

// GCSObjectInterface representa un objeto en GCS
type GCSObjectInterface interface {
	Delete(ctx context.Context) error
	Attrs(ctx context.Context) (*storage.ObjectAttrs, error)
}

// GCSObjectIteratorInterface representa un iterador de objetos
type GCSObjectIteratorInterface interface {
	Next() (*storage.ObjectAttrs, error)
}

// Adaptadores para que storage.Client implemente GCSClientInterface
type GCSClientAdapter struct {
	*storage.Client
}

func (a *GCSClientAdapter) Bucket(name string) GCSBucketInterface {
	return &GCSBucketAdapter{a.Client.Bucket(name)}
}

type GCSBucketAdapter struct {
	*storage.BucketHandle
}

func (a *GCSBucketAdapter) Object(name string) GCSObjectInterface {
	return &GCSObjectAdapter{a.BucketHandle.Object(name)}
}

func (a *GCSBucketAdapter) Objects(ctx context.Context, query *storage.Query) GCSObjectIteratorInterface {
	return a.BucketHandle.Objects(ctx, query)
}

type GCSObjectAdapter struct {
	*storage.ObjectHandle
}

func (a *GCSObjectAdapter) Delete(ctx context.Context) error {
	return a.ObjectHandle.Delete(ctx)
}

func (a *GCSObjectAdapter) Attrs(ctx context.Context) (*storage.ObjectAttrs, error) {
	return a.ObjectHandle.Attrs(ctx)
}
