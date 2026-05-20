package foundation

// MockRepoFactory implements RepoFactory by returning a pre-injected
// Repository from every method. Use it in tests when you need to control
// what Repository comes out of the factory without setting up a real
// MongoRepoFactory.
//
// Usage:
//
//	repo := NewMockRepository()
//	// configure repo behaviour ...
//	factory := NewMockRepoFactory(repo)
//
//	factory.FromModel(someModel, "mongo://...")  // returns repo, nil
//	factory.Clone(otherRepo, someModel)           // returns repo, nil
//	len(factory.FromModelCalls) == 1              // call tracking
//
// If your test needs FromModel to return an error, set InjectErr on the
// factory before exercising the code under test.
type MockRepoFactory struct {
	// Repo is the Repository returned by every method.
	Repo Repository

	// InjectErr, when non-nil, is returned as the error from every method.
	// This lets tests exercise error paths without replacing the mock.
	InjectErr error

	FromModelCalls []MockRepoFactoryFromModelCall
	CloneCalls     []MockRepoFactoryCloneCall
}

// compile-time check: *MockRepoFactory implements RepoFactory
var _ RepoFactory = (*MockRepoFactory)(nil)

// NewMockRepoFactory returns a MockRepoFactory that returns repo from
// every method with a nil error.
func NewMockRepoFactory(repo Repository) *MockRepoFactory {
	return &MockRepoFactory{Repo: repo}
}

// MockRepoFactoryFromModelCall records one call to FromModel.
type MockRepoFactoryFromModelCall struct {
	Model      RepositoryModel
	Connection string
}

// MockRepoFactoryCloneCall records one call to Clone.
type MockRepoFactoryCloneCall struct {
	Repo  Repository
	Model RepositoryModel
}

func (f *MockRepoFactory) FromModel(model RepositoryModel, connection string) (Repository, error) {
	f.FromModelCalls = append(f.FromModelCalls, MockRepoFactoryFromModelCall{
		Model:      model,
		Connection: connection,
	})
	return f.Repo, f.InjectErr
}

func (f *MockRepoFactory) Clone(repo Repository, model RepositoryModel) (Repository, error) {
	f.CloneCalls = append(f.CloneCalls, MockRepoFactoryCloneCall{
		Repo:  repo,
		Model: model,
	})
	return f.Repo, f.InjectErr
}
