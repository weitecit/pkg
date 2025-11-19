package foundation

import (
	"errors"

	"github.com/weitecit/pkg/utils"
)

type TagType utils.Enum

const (
	TagTypeNone    TagType = ""
	TagTypeUser    TagType = "tag_user"
	TagTypeDomain  TagType = "tag_domain"
	TagTypeContact TagType = "tag_contact"
	TagTypeFile    TagType = "tag_file"
	TagTypeTask    TagType = "tag_task"
)

func GetTagType(name string) (TagType, error) {
	switch name {
	case "tag_user":
		return TagTypeUser, nil
	case "tag_domain":
		return TagTypeDomain, nil
	case "tag_contact":
		return TagTypeContact, nil
	case "tag_file":
		return TagTypeFile, nil
	case "tag_task":
		return TagTypeTask, nil
	default:
		return TagTypeNone, errors.New("TagType.GetTagType: invalid tag type: " + name)
	}
}

// Esta clase contiene todo tipo de configuraciones... tags, monedas, etc.
type TagRepo struct {
	TagType TagType `json:"tag_type" bson:"tag_type"`
	// System tags are not editable by users at the moment and has always the same key
	System bool `json:"system" bson:"system"`
	Tag    `bson:",inline"`
}

// Must be used instead of TagRepo
type TagRepoList struct {
	List     []*TagRepo `bson:",inline"`
	TagList  []Tag
	TagType  TagType
	Language Language
	Touched  bool
	RepoID   string
	User     User
}

// func NewTagRepoList(tagType TagType, language Language, repoID string, user User) (*TagRepoList, error) {

// 	if tagType == TagTypeNone {
// 		return nil, errors.New("TagRepos.NewTagRepos: tag type is required")
// 	}

// 	language, err := language.Validate()
// 	if err != nil {
// 		language = user.Language
// 	}

// 	language, err = language.Validate()
// 	if err != nil {
// 		return nil, err
// 	}

// 	if utils.IsEmptyStr(repoID) {
// 		return nil, errors.New("TagRepos.NewTagRepos: repo id is required")
// 	}

// 	result := &TagRepoList{
// 		TagType:  tagType,
// 		Language: language,
// 		List:     []*TagRepo{},
// 		RepoID:   repoID,
// 		User:     user,
// 	}

// 	return result, nil
// }

// func (m *TagRepoList) Compare(tag Tag) Tag {
// 	for _, repoTag := range m.List {
// 		if repoTag.Tag.HasSameKey(tag) {
// 			return repoTag.Tag
// 		}
// 	}

// 	newRepoTag := &TagRepo{
// 		Tag:     tag,
// 		TagType: m.TagType,
// 	}
// 	newRepoTag.Language = m.Language
// 	newRepoTag.Touched = true

// 	m.List = append(m.List, newRepoTag)
// 	m.Touched = true
// 	return tag
// }

// func (m *TagRepoList) AddList(list interface{}) error {
// 	jsonModel, err := json.Marshal(list)
// 	if err != nil {
// 		return err
// 	}
// 	modelRepo := []*TagRepo{}
// 	json.Unmarshal(jsonModel, &modelRepo)
// 	m.List = append(m.List, modelRepo...)
// 	return nil
// }

// func (m *TagRepoList) AddTags(tags []Tag) {
// 	for _, tag := range tags {
// 		m.Compare(tag)
// 	}
// }

// func (m *TagRepoList) Update(request BaseRequest) BaseResponse {

// 	if !m.Touched {
// 		return BaseResponse{}
// 	}

// 	if utils.IsEmptyStr(m.RepoID) {
// 		return BaseResponse{Error: errors.New("TagRepoList.Update: repo id is required")}
// 	}

// 	model := &TagRepo{}
// 	model.RepoID = m.RepoID

// 	var err error
// 	request.Repo, err = NewRepositoryFromModel(model, m.User.Connection)

// 	if err != nil {
// 		return BaseResponse{Error: err}
// 	}

// 	m.Touched = false

// 	response := BaseResponse{}

// 	for _, tagRepo := range m.List {
// 		if !tagRepo.Touched {
// 			continue
// 		}
// 		tagRepo.Touched = false
// 		var resp BaseResponse
// 		if tagRepo.IsNew() {
// 			resp = tagRepo.FindOrCreate(&request)
// 		} else {
// 			resp = tagRepo.Update(&request)
// 		}
// 		if resp.Error != nil {
// 			return resp
// 		}
// 		response.TotalRows++
// 	}

// 	return response
// }

// func (m *TagRepoList) GetTagsByValue(list []string) (returnable []Tag, errs []error) {
// 	for _, str := range list {
// 		if utils.IsEmptyStr(str) {
// 			continue
// 		}
// 		needAddErr := true
// 		for _, tagRepo := range m.List {
// 			if str == "*" && len(list) == 1 {
// 				returnable = append(returnable, tagRepo.Tag)
// 				needAddErr = false
// 				continue
// 			}
// 			if utils.NormalizeForTag(str) == utils.NormalizeForTag(tagRepo.Tag.Value) {
// 				returnable = append(returnable, tagRepo.Tag)
// 				needAddErr = false
// 				break
// 			}
// 		}
// 		if needAddErr {
// 			errs = append(errs, errors.New("tag "+str+" does not exist"))
// 		}
// 	}

// 	return returnable, errs
// }

// func (m *TagRepoList) GetTagsByKey(list []string) (returnable []Tag, errs []error) {
// 	for _, str := range list {
// 		if utils.IsEmptyStr(str) {
// 			continue
// 		}
// 		needAddErr := true
// 		for _, tagRepo := range m.List {
// 			if str == "*" && len(list) == 1 {
// 				returnable = append(returnable, tagRepo.Tag)
// 				needAddErr = false
// 				continue
// 			}
// 			if strings.ToLower(str) == tagRepo.Tag.Key {
// 				returnable = append(returnable, tagRepo.Tag)
// 				needAddErr = false
// 				break
// 			}
// 		}
// 		if needAddErr {
// 			errs = append(errs, errors.New("tag "+str+" does not exist"))
// 		}
// 	}

// 	return returnable, errs
// }

// func NewIsolatedTagRepo() *TagRepo {
// 	return &TagRepo{
// 		ConfigType: ConfigTypeTag,
// 	}
// }

// func (m *TagRepo) Validate() (*TagRepo, error) {
// 	m.ConfigType = ConfigTypeTag

// 	if m.TagType == TagTypeNone {
// 		return m, errors.New("TagRepo.Validate: tagrepo type is required")
// 	}

// 	if m.Language == "" {
// 		return m, errors.New("TagRepo.Validate: language is required")
// 	}

// 	return m, nil
// }

// func (m *TagRepo) GetFindOptions(request *BaseRequest) *FindOptions {
// 	findOptions := m.GetBaseFindOptions(request)

// 	findOptions.AddEquals("config_type", ConfigTypeTag)

// 	if m.Tag.Key != "" {
// 		findOptions.AddEquals("key", m.Tag.Key)
// 	}

// 	if m.Tag.Value != "" {
// 		findOptions.AddEquals("value", m.Tag.Value)
// 	}

// 	if m.TagType != TagTypeNone {
// 		findOptions.AddEquals("tag_type", m.TagType)
// 	}

// 	return findOptions
// }

// func (m *TagRepo) FindOrCreate(request *BaseRequest) BaseResponse {
// 	response := m.FindByKey(request)
// 	if response.TotalRows == 1 {
// 		err := response.GetFirst(m)
// 		if err != nil {
// 			return BaseResponse{Error: err}
// 		}
// 		return response
// 	}

// 	if response.TotalRows > 1 {
// 		return BaseResponse{Error: errors.New("TagRepo.FindOrCreate: more than one tagrepo found")}
// 	}

// 	return m.Update(request)
// }

// func (m *TagRepo) FindByKey(request *BaseRequest) BaseResponse {

// 	findOptions := m.GetBaseFindOptions(request)

// 	findOptions.AddEquals("config_type", ConfigTypeTag)

// 	if m.Tag.Key != "" {
// 		findOptions.AddEquals("key", m.Tag.Key)
// 	}

// 	if m.TagType != TagTypeNone {
// 		findOptions.AddEquals("tag_type", m.TagType)
// 	}

// 	request.findOptions = *findOptions
// 	request.Model = m
// 	request.List = []*TagRepo{}

// 	response := m.BaseFind(request)

// 	return response
// }
