package foundation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
)

type GCSFileRepository struct {
	FileID         string
	FileName       string
	Path           string
	File           io.Reader
	RepositoryPath string
	client         *storage.Client
	bucket         string
	projectID      string
	User           User
}

func (m *GCSFileRepository) ToJSON() string {
	o, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func NewGCSFileRepository(request FileRepoRequest) (GCSFileRepository, error) {
	repo := GCSFileRepository{
		FileID:   request.FileID,
		FileName: request.FileName,
		Path:     request.Folder,
		File:     request.File,
		User:     request.User,
	}

	if !request.User.IsValid() {
		return repo, errors.New("NewFileRepository: User is not valid")
	}

	if repo.Path == "" {
		return repo, errors.New("NewFileRepository: Invalid path")
	}

	return repo, nil
}

func (m *GCSFileRepository) Save() FileRepoResponse {

	ctx := context.Background()

	response := &FileRepoResponse{}

	if m.FileName == "" {
		return FileRepoResponse{Error: errors.New("FileRepository.Save: FileName is empty")}
	}

	if m.File == nil {
		return FileRepoResponse{Error: errors.New("FileRepository.Save: File is nil")}
	}

	token, err := GetGCPToken(m.User.ExternalToken, "storage", m.User)
	if err != nil {
		err = fmt.Errorf("Error getting GCP token: %v", err)
		return FileRepoResponse{Error: err}
	}

	tokenSource := oauth2.StaticTokenSource(token)

	httpClient, _, err := transport.NewHTTPClient(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		err = fmt.Errorf("Error creating HTTP client: %v", err)
		return FileRepoResponse{Error: err}
	}

	// Create storage client with the custom HTTP client
	client, err := storage.NewClient(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		err = fmt.Errorf("Error creating storage client: %v", err)
		return FileRepoResponse{Error: err}
	}
	defer client.Close()

	m.bucket = utils.GetEnv("BUCKET_NAME")

	path := "data/" + m.Path + "/" + m.FileID

	wc := client.Bucket(m.bucket).Object(path).NewWriter(ctx)
	if _, err = io.Copy(wc, m.File); err != nil {
		err = fmt.Errorf("Error copying file: %v", err)
		return FileRepoResponse{Error: err}
	}
	if err := wc.Close(); err != nil {
		err = fmt.Errorf("Error closing writer: %v", err)
		return FileRepoResponse{Error: err}
	}

	return *response
}

func (m *GCSFileRepository) GetPath() string {
	return m.Path
}

func (m *GCSFileRepository) Download() FileRepoResponse {

	ctx := context.Background()

	if m.FileID == "" {
		return FileRepoResponse{Error: errors.New("FileRepository.GetFile: FileID is empty")}
	}

	if m.FileName == "" {
		m.FileName = m.FileID
	}

	token, err := GetGCPToken(m.User.ExternalToken, "storage", m.User)
	if err != nil {
		err = fmt.Errorf("Error getting GCP token: %v", err)
		return FileRepoResponse{Error: err}
	}

	tokenSource := oauth2.StaticTokenSource(token)

	httpClient, _, err := transport.NewHTTPClient(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		err = fmt.Errorf("Error creating HTTP client: %v", err)
		return FileRepoResponse{Error: err}
	}

	// Create storage client with the custom HTTP client
	client, err := storage.NewClient(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		err = fmt.Errorf("Error creating storage client: %v", err)
		return FileRepoResponse{Error: err}
	}
	defer client.Close()

	m.bucket = utils.GetEnv("BUCKET_NAME")

	path := "data/" + m.Path + "/" + m.FileID
	rc, err := client.Bucket(m.bucket).Object(path).NewReader(ctx)
	if err != nil {
		return FileRepoResponse{Error: err}
	}

	content, err := io.ReadAll(rc)
	if err != nil {
		return FileRepoResponse{Error: err}
	}

	return FileRepoResponse{
		Content: content,
	}

}

func (m *GCSFileRepository) Open() FileRepoResponse {
	response := &FileRepoResponse{}

	if m.FileID == "" {
		return FileRepoResponse{Error: errors.New("FileRepository.GetFile: FileID is empty")}
	}

	if m.FileName == "" {
		m.FileName = m.FileID
	}

	m.bucket = utils.GetEnv("BUCKET_NAME")
	err := m.Connect()
	if err != nil {
		response.Error = err
		return *response
	}

	path := "data/" + m.Path + "/" + m.FileID
	ctx := context.Background()
	rc, err := m.client.Bucket(m.bucket).Object(path).NewReader(ctx)
	if err != nil {
		return FileRepoResponse{Error: err}
	}

	m.File = rc

	return *response
}

func (m *GCSFileRepository) Close() {
	if closer, ok := m.File.(io.Closer); ok {
		closer.Close()
	}
	m.File = nil
}

func (m *GCSFileRepository) Delete() FileRepoResponse {

	ctx := context.Background()

	if m.FileID == "" {
		return FileRepoResponse{Error: errors.New("GCSFileRepository.Delete: FileID is empty")}
	}

	if m.FileName == "" {
		m.FileName = m.FileID
	}

	token, err := GetGCPToken(m.User.ExternalToken, "storage", m.User)
	if err != nil {
		err = fmt.Errorf("Error getting GCP token: %v", err)
		return FileRepoResponse{Error: err}
	}

	tokenSource := oauth2.StaticTokenSource(token)

	httpClient, _, err := transport.NewHTTPClient(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		err = fmt.Errorf("Error creating HTTP client: %v", err)
		return FileRepoResponse{Error: err}
	}

	// Create storage client with the custom HTTP client
	client, err := storage.NewClient(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		err = fmt.Errorf("Error creating storage client: %v", err)
		return FileRepoResponse{Error: err}
	}
	defer client.Close()

	m.bucket = utils.GetEnv("BUCKET_NAME")

	path := "data/" + m.Path + "/" + m.FileID
	err = client.Bucket(m.bucket).Object(path).Delete(ctx)
	if err != nil {
		return FileRepoResponse{Error: err}
	}

	return FileRepoResponse{}
}

func (m *GCSFileRepository) GetAccess() FileRepoResponse {
	// Implementación pendiente
	return FileRepoResponse{}
}

func (m *GCSFileRepository) EmptyBin() FileRepoResponse {
	// Implementación pendiente
	return FileRepoResponse{}
}

func (m *GCSFileRepository) Connect() error {
	for _, connection := range FileConnectionPool {
		con := connection.(*GCSFileRepository)
		if con.bucket != m.bucket {
			continue
		}

		m.client = con.client
		return nil
	}

	ctx := context.Background()
	m.projectID = utils.GetEnv("GCP_PROJECT_ID")
	credentialsFile := utils.GetEnv("GOOGLE_APPLICATION_CREDENTIALS")

	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return err
	}

	m.client = client
	FileConnectionPool = append(FileConnectionPool, m)

	return nil
}

func (m *GCSFileRepository) GetFile() io.Reader {
	return m.File
}

func (m *GCSFileRepository) Mount(fileID string, fileName string, path string, repositoryPath string) {
	m.FileID = fileID
	m.FileName = fileName
	m.Path = path
	m.RepositoryPath = repositoryPath
}
