package foundation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"
)

type MockFileRepository struct {
	FileID         string
	FileName       string
	Path           string
	RepositoryPath string
}

func (m *MockFileRepository) ToJSON() string {
	o, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func NewMockupFileRepository(request FileRepoRequest) (MockFileRepository, error) {
	repo := MockFileRepository{
		FileID:         request.FileID,
		FileName:       request.FileName,
		Path:           request.Folder,
		RepositoryPath: "file_repository",
	}

	if repo.Path == "" {
		return repo, errors.New("NewFileRepository: Invalid path")
	}

	return repo, nil
}

func (m *MockFileRepository) GetPath() string {
	return m.RepositoryPath + "/" + m.Path + "/" + m.FileName
}

func (m *MockFileRepository) Open() FileRepoResponse {
	return FileRepoResponse{}
}

func (m *MockFileRepository) Close() {
}

func (m *MockFileRepository) Save() FileRepoResponse {

	if m.FileID == "" {
		return FileRepoResponse{Error: errors.New("FileRepository.Save: FileID is empty")}
	}

	response := &FileRepoResponse{}

	repo := &MockIO{}
	response.Error = repo.AddFile(*m)
	if response.Error != nil {
		return *response
	}

	return *response

}

func (m *MockFileRepository) Delete() FileRepoResponse {

	println("•••••••••••••••••••••••••••••••••")
	fmt.Println("hace delete")
	println("•••••••••••••••••••••••••••••••••")
	response := &FileRepoResponse{}

	repo := &MockIO{}
	response.Error = repo.Delete(*m)

	return *response
}

func (m *MockFileRepository) EmptyBin() FileRepoResponse {
	return FileRepoResponse{}
}

func (m *MockFileRepository) GetFile() io.Reader {
	return io.NopCloser(strings.NewReader("test"))
}

func (m *MockFileRepository) GetAccess() FileRepoResponse {
	return FileRepoResponse{}
}

func (m *MockFileRepository) Download() FileRepoResponse {
	return FileRepoResponse{}
}

type MockIO struct {
	Folders []*MockFolder
}

type MockFolder struct {
	ID    string
	Files []string
}

func (m *MockIO) Len(repoID string) int {

	for _, folder := range m.Folders {
		if folder.ID == repoID {
			return len(folder.Files)
		}
	}

	return -1
}

func (m *MockIO) GetFiles(repoID string) ([]string, error) {

	err := m.open()
	if err != nil {
		return []string{}, err
	}

	for _, folder := range m.Folders {
		if folder.ID == repoID {
			return folder.Files, nil
		}
	}

	return []string{}, nil
}

func (m *MockIO) CountFiles(repoID string) int {

	files, err := m.GetFiles(repoID)
	if err != nil {
		return -1
	}

	return len(files)
}

func (m *MockIO) open() error {

	jsonFile, err := os.OpenFile("file_repo.json", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(byteValue, &m.Folders)
	if err != nil && err.Error() != "unexpected end of JSON input" {
		return err
	}

	return nil
}

func (m *MockIO) save() error {
	return os.WriteFile("file_repo.json", []byte(utils.ToJSON(m.Folders)), 0666)
}

func (m *MockIO) addFolder(repo MockFileRepository) error {

	if repo.Path == "" {
		return errors.New("addFolder: Invalid path")
	}

	for _, folder := range m.Folders {
		if folder.ID == repo.Path {
			return nil
		}
	}

	m.Folders = append(m.Folders, &MockFolder{
		ID:    repo.Path,
		Files: []string{},
	})

	return nil
}

func (m *MockIO) AddFile(repo MockFileRepository) error {

	err := m.open()
	if err != nil {
		return err
	}

	err = m.addFolder(repo)
	if err != nil {
		return err
	}

	for _, folder := range m.Folders {
		if folder.ID == repo.Path {
			folder.Files = utils.AddStringsInArray(folder.Files, repo.FileID)
			err = m.save()
			return err
		}
	}

	return errors.New("addFile: Cannot add file")
}

func (m *MockIO) Delete(repo MockFileRepository) error {

	err := m.open()
	if err != nil {
		return err
	}

	for i, folder := range m.Folders {
		if folder.ID == repo.Path {
			m.Folders[i].Files, _ = utils.RemoveStringsFromArray(folder.Files, repo.FileID)
			return m.save()
		}
	}

	return errors.New("Delete: File not found")
}
