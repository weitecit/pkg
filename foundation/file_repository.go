package foundation

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"
)

var FileConnectionPool = []FileRepository{}

type FileRepoType utils.Enum

const (
	FileRepoNone      FileRepoType = ""
	FileRepoLocal     FileRepoType = "file_repo_local"
	FileRepoS3        FileRepoType = "file_repo_s3"
	FileRepoGoogle    FileRepoType = "file_repo_google"
	FileRepoOffice365 FileRepoType = "file_repo_office365"
	FileRepoMockup    FileRepoType = "file_repo_mockup"
)

func (m FileRepoType) GetFileRepoType(s string) FileRepoType {
	switch s {
	case "file_repo_mockup":
		return FileRepoMockup
	case "file_repo_local":
		return FileRepoLocal
	case "file_repo_s3":
		return FileRepoS3
	case "file_repo_office365":
		return FileRepoOffice365
	case "file_repo_google":
		return FileRepoGoogle
	default:
		return FileRepoNone
	}
}

type FileRepoRequest struct {
	FileID   string
	Folder   string
	FileName string
	File     io.Reader
	RepoType FileRepoType
	User     User
}

func (m *FileRepoRequest) ToJSON() string {
	o, err := json.MarshalIndent(&m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

type FileRepoResponse struct {
	Error   error
	Content []byte
}

func NewFileRepository(request FileRepoRequest) (FileRepository, error) {

	if request.FileName == "" && request.FileID == "" {
		return nil, errors.New("NewFileRepository: FileName or ID is required")
	}

	if request.FileName == "" {
		request.FileName = request.FileID
	}

	if request.Folder == "" {
		return nil, errors.New("NewFileRepository: Folder is required")
	}

	if request.RepoType == "" {
		return nil, errors.New("NewFileRepository: RepoType is required")
	}

	if !request.User.IsValid() {
		return nil, errors.New("NewFileRepository: User is not valid")
	}

	switch request.RepoType {
	case FileRepoLocal:
		repo, err := NewLocalFileRepository(request)
		return &repo, err
	case FileRepoS3:
		return nil, errors.New("NewFileRepository: S3 is not supported")
		// repo, err := NewS3FileRepository(request)
		// return &repo, err
	case FileRepoGoogle:
		repo, err := NewGCSFileRepository(request)
		return &repo, err
	case FileRepoMockup:
		repo, err := NewMockupFileRepository(request)
		return &repo, err
	default:
		return nil, errors.New("NewFileRepository: RepoType is not supported: " + string(request.RepoType))
	}
}

type FileRepository interface {
	Save() FileRepoResponse
	Delete() FileRepoResponse
	EmptyBin() FileRepoResponse
	GetAccess() FileRepoResponse
	Download() FileRepoResponse
	GetPath() string
	GetFile() io.Reader
	Open() FileRepoResponse
	Close()
}
