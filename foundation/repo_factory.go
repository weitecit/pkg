package foundation

// foundation/repository.go
type RepoFactory interface {
	FromModel(model RepositoryModel, connection string) (Repository, error)
	Clone(repo Repository, model RepositoryModel) (Repository, error)
}

// Implementación concreta (producción)
type MongoRepoFactory struct{}

func (f *MongoRepoFactory) FromModel(model RepositoryModel, connection string) (Repository, error) {
	return NewRepositoryFromModel(model, connection)
}
func (f *MongoRepoFactory) Clone(repo Repository, model RepositoryModel) (Repository, error) {
	return CloneRepository(repo, model)
}
