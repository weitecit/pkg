package foundation

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// MockGCSClient simula storage.Client para tests
type MockGCSClient struct {
	Buckets map[string]*MockBucket
	mu      sync.RWMutex
}

// NewMockGCSClient crea un nuevo cliente GCS mock
func NewMockGCSClient() *MockGCSClient {
	return &MockGCSClient{
		Buckets: make(map[string]*MockBucket),
	}
}

// Bucket retorna o crea un bucket mock
func (m *MockGCSClient) Bucket(name string) GCSBucketInterface {
	m.mu.Lock()
	defer m.mu.Unlock()

	if bucket, exists := m.Buckets[name]; exists {
		return bucket
	}

	bucket := &MockBucket{
		Name:       name,
		ObjectsMap: make(map[string]*MockObject),
		client:     m,
	}
	m.Buckets[name] = bucket
	return bucket
}

// Close simula el cierre del cliente
func (m *MockGCSClient) Close() error {
	return nil
}

// GetBucket retorna el bucket mock real (helper para tests)
func (m *MockGCSClient) GetBucket(name string) *MockBucket {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Buckets[name]
}

// MockBucket simula storage.BucketHandle
type MockBucket struct {
	Name       string
	ObjectsMap map[string]*MockObject
	client     *MockGCSClient
	mu         sync.RWMutex
}

// Object retorna o crea un objeto mock
func (b *MockBucket) Object(path string) GCSObjectInterface {
	b.mu.Lock()
	defer b.mu.Unlock()

	if obj, exists := b.ObjectsMap[path]; exists {
		return obj
	}

	obj := &MockObject{
		Path:    path,
		bucket:  b,
		Deleted: false,
	}
	b.ObjectsMap[path] = obj
	return obj
}

// Objects retorna un iterador de objetos que coinciden con el query
func (b *MockBucket) Objects(ctx context.Context, query *storage.Query) GCSObjectIteratorInterface {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var matchingObjects []*storage.ObjectAttrs

	prefix := ""
	if query != nil && query.Prefix != "" {
		prefix = query.Prefix
	}

	for path, obj := range b.ObjectsMap {
		if !obj.Deleted && strings.HasPrefix(path, prefix) {
			// No incluir el directorio mismo
			if path != prefix && !strings.HasSuffix(path, "/") {
				matchingObjects = append(matchingObjects, &storage.ObjectAttrs{
					Name: path,
					Size: int64(len(obj.Data)),
				})
			}
		}
	}

	return &MockObjectIterator{
		objects: matchingObjects,
		index:   0,
	}
}

// MockObject simula storage.ObjectHandle
type MockObject struct {
	Path    string
	Data    []byte
	Deleted bool
	bucket  *MockBucket
	mu      sync.RWMutex
}

// NewWriter crea un nuevo writer mock para este objeto
func (o *MockObject) NewWriter(ctx context.Context) *MockWriter {
	return &MockWriter{
		Object: o,
		Buffer: new(bytes.Buffer),
		Closed: false,
	}
}

// Delete marca el objeto como eliminado
func (o *MockObject) Delete(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.Deleted {
		return storage.ErrObjectNotExist
	}

	o.Deleted = true
	return nil
}

// Attrs retorna los atributos del objeto
func (o *MockObject) Attrs(ctx context.Context) (*storage.ObjectAttrs, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.Deleted {
		return nil, storage.ErrObjectNotExist
	}

	return &storage.ObjectAttrs{
		Name: o.Path,
		Size: int64(len(o.Data)),
	}, nil
}

// MockWriter simula storage.Writer
type MockWriter struct {
	Object    *MockObject
	Buffer    *bytes.Buffer
	Closed    bool
	ChunkSize int
	mu        sync.Mutex
}

// Write escribe datos al buffer
func (w *MockWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Closed {
		return 0, fmt.Errorf("writer is closed")
	}

	return w.Buffer.Write(p)
}

// Close finaliza la escritura y guarda los datos en el objeto
func (w *MockWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Closed {
		return fmt.Errorf("writer already closed")
	}

	w.Closed = true

	// Guardar los datos en el objeto
	w.Object.mu.Lock()
	w.Object.Data = w.Buffer.Bytes()
	w.Object.Deleted = false
	w.Object.mu.Unlock()

	return nil
}

// MockObjectIterator simula el iterador de objetos
type MockObjectIterator struct {
	objects []*storage.ObjectAttrs
	index   int
	mu      sync.Mutex
}

// Next retorna el siguiente objeto en el iterador
func (it *MockObjectIterator) Next() (*storage.ObjectAttrs, error) {
	it.mu.Lock()
	defer it.mu.Unlock()

	if it.index >= len(it.objects) {
		return nil, iterator.Done
	}

	obj := it.objects[it.index]
	it.index++
	return obj, nil
}

// GetAllObjects retorna todos los objetos del bucket (helper para tests)
func (b *MockBucket) GetAllObjects() map[string]*MockObject {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Retornar una copia para evitar race conditions
	objects := make(map[string]*MockObject)
	for k, v := range b.ObjectsMap {
		objects[k] = v
	}
	return objects
}

// CountNonDeletedObjects cuenta objetos no eliminados (helper para tests)
func (b *MockBucket) CountNonDeletedObjects() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	count := 0
	for _, obj := range b.ObjectsMap {
		if !obj.Deleted {
			count++
		}
	}
	return count
}

// Reset limpia todos los objetos del bucket (helper para tests)
func (b *MockBucket) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ObjectsMap = make(map[string]*MockObject)
}

// ReadObject lee el contenido de un objeto (helper para tests)
func (b *MockBucket) ReadObject(path string) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	obj, exists := b.ObjectsMap[path]
	if !exists {
		return nil, fmt.Errorf("object not found: %s", path)
	}

	obj.mu.RLock()
	defer obj.mu.RUnlock()

	if obj.Deleted {
		return nil, fmt.Errorf("object deleted: %s", path)
	}

	return obj.Data, nil
}
