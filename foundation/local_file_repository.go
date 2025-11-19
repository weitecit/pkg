package foundation

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/weitecit/pkg/log"
)

type LocalFileRepository struct {
	FileID         string
	FileName       string
	Path           string
	File           io.Reader
	RepositoryPath string
}

func (m *LocalFileRepository) ToJSON() string {
	o, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func NewLocalFileRepository(request FileRepoRequest) (LocalFileRepository, error) {
	repo := LocalFileRepository{
		FileID:         request.FileID,
		FileName:       request.FileName,
		Path:           request.Folder,
		File:           request.File,
		RepositoryPath: "file_repository",
	}

	if repo.Path == "" {
		return repo, errors.New("NewFileRepository: Invalid path")
	}

	return repo, nil
}

func (m *LocalFileRepository) GetPath() string {
	return m.RepositoryPath + "/" + m.Path + "/" + m.FileID
}

func (m *LocalFileRepository) Save() FileRepoResponse {

	if m.FileName == "" {
		return FileRepoResponse{Error: errors.New("FileRepository.Save: FileName is empty")}
	}

	if m.File == nil {
		return FileRepoResponse{Error: errors.New("FileRepository.Save: File is nil")}
	}

	response := &FileRepoResponse{}

	path := m.RepositoryPath + "/" + m.Path

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			response.Error = err
			return *response
		}
	}

	path += "/" + m.FileID

	destination, err := os.Create(path)
	if err != nil {
		response.Error = err
		return *response
	}

	_, err = io.Copy(destination, m.File)
	if err != nil {
		log.Err(err)
		response.Error = err
		return *response
	}

	return FileRepoResponse{}

}

func (m *LocalFileRepository) Open() FileRepoResponse {

	response := &FileRepoResponse{}

	if m.FileID == "" {
		return FileRepoResponse{Error: errors.New("FileRepository.GetFile: FileID is empty")}
	}

	if m.FileName == "" {
		m.FileName = m.FileID
	}

	path := m.RepositoryPath + "/" + m.Path + "/" + m.FileID

	file, err := os.Open(path)
	if err != nil {
		response.Error = err
		return *response
	}

	m.File = file

	return FileRepoResponse{}
}

func (m *LocalFileRepository) Delete() FileRepoResponse {

	response := &FileRepoResponse{}

	path := m.RepositoryPath + "/" + m.Path + "/" + m.FileID

	err := os.Remove(path)
	if err != nil {
		response.Error = err
		return *response
	}

	return FileRepoResponse{}

}

func (m *LocalFileRepository) GetAccess() FileRepoResponse {

	return FileRepoResponse{}
}

func (m *LocalFileRepository) EmptyBin() FileRepoResponse {
	return FileRepoResponse{}
}

func (m *LocalFileRepository) Download() FileRepoResponse {

	path := m.GetPath()
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return FileRepoResponse{
			Error: errors.New("File does not exist: " + path),
		}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return FileRepoResponse{
			Error: err,
		}
	}

	return FileRepoResponse{
		Content: content,
	}
}

func (m *LocalFileRepository) GetFile() io.Reader {
	return m.File
}

func (m *LocalFileRepository) Close() {
	m.File = nil
}
