package foundation

import (
	"strings"

	"github.com/weitecit/pkg/utils"
)

type TagCloud struct {
	TagKeys   []string `json:"tag_keys" bson:"tag_keys"`
	TagValues []string `json:"tag_values" bson:"tag_values"`
	Tags      Tags     `json:"tags" bson:"tags"`
}

func NewTagCloud() *TagCloud {
	return &TagCloud{
		TagKeys:   []string{},
		TagValues: []string{},
		Tags:      []Tag{},
	}
}

func (m *TagCloud) HasTag(tag Tag) int {
	for i, item := range m.Tags {
		if item.Key == tag.Key {
			return i
		}
	}
	return -1
}

func (m *TagCloud) AddTag(tag Tag) (bool, string) {
	index := m.HasTag(tag)

	if index != -1 {
		return false, m.TagValues[index]
	}

	m.Tags = append(m.Tags, tag)
	m.TagKeys = append(m.TagKeys, tag.Key)
	m.TagValues = append(m.TagValues, tag.Value)

	return true, tag.Value
}

func (m *TagCloud) AddTagCloud(tagCloud TagCloud) {
	for _, tag := range tagCloud.Tags {
		m.AddTag(tag)
	}
}

func (m *TagCloud) RemoveTag(tag Tag) bool {
	index := m.HasTag(tag)

	if index == -1 {
		return false
	}

	m.Tags = append(m.Tags[:index], m.Tags[index+1:]...)
	m.TagKeys = append(m.TagKeys[:index], m.TagKeys[index+1:]...)
	m.TagValues = append(m.TagValues[:index], m.TagValues[index+1:]...)
	return true
}

type Tags []Tag

type Tag struct {
	Key   string `json:"key" bson:"key"`
	Value string `json:"value" bson:"value"`
	Group string `json:"group,omitempty" bson:"group,omitempty"`
	Check bool   `json:"check,omitempty" bson:"check,omitempty"`
}

func NewTag(key string, value string) *Tag {
	return &Tag{
		Key:   key,
		Value: value,
	}
}

func NewTagByValue(value string) *Tag {
	tag := &Tag{
		Value: value,
	}
	tag.Normalize()
	return tag
}

func (m *Tag) Normalize() {
	value := strings.TrimSpace(m.Value)
	key := utils.Normalize(value)
	m.Key = utils.RemovePunctuation(key)
}

func (m *Tag) HasSameKey(tag Tag) bool {
	return m.Key == tag.Key
}

func (m *Tag) IsEmpty() bool {
	return m.Key == "" && m.Value == ""
}

func (m *Tag) IsEqual(tag Tag) bool {
	return m.Key == tag.Key && m.Value == tag.Value
}

func (m *Tags) Total() int {
	return len(*m)
}

func (m *Tags) GetKeys() []string {
	result := []string{}
	if m.Total() < 0 {
		return result
	}
	for _, tag := range m.ToArray() {
		result = append(result, tag.Key)
	}
	return result
}

func (m *Tags) AddValue(value string) (bool, Tag) {

	tag := NewTagByValue(value)

	return m.AddTag(*tag)
}

func (m *Tags) AddTag(tag Tag) (bool, Tag) {

	hasTag, rTag := m.HasTagByKeyAndGetTag(tag.Key)

	if hasTag {
		return false, rTag
	}

	*m = append(*m, tag)

	return true, tag
}

func (m *Tags) AddTags(tags Tags) {

	for _, tag := range tags {
		m.AddTag(tag)
	}
}

func (m *Tags) HasTag(tag Tag) bool {
	for _, item := range *m {
		if item.HasSameKey(tag) {
			return true
		}
	}
	return false
}

func (m *Tags) HasTagAndGet(tag Tag) (bool, Tag) {
	for _, item := range *m {
		if item.HasSameKey(tag) {
			return true, item
		}
	}
	return false, Tag{}
}

func (m *Tags) HasTagByKeyAndGetTag(key string) (bool, Tag) {
	for _, item := range *m {
		if item.Key == key {
			return true, item
		}
	}
	return false, Tag{}
}

func (m *Tags) HasTagByKey(key string) bool {

	has, _ := m.HasTagByKeyAndGetTag(key)
	return has
}

func (m *Tags) ToArray() []Tag {
	return *m
}

func (m *Tags) ToArrayByValues() []string {
	var returnable []string
	for _, tag := range m.ToArray() {
		returnable = append(returnable, tag.Value)
	}

	return returnable
}

func (m *Tags) Normalize(tagRepos *TagRepoList) (*Tags, error) {

	if tagRepos == nil {
		return m, nil
	}

	model := &TagRepo{}
	// model.RepoID = tagRepos.RepoID
	// model.Language = tagRepos.Language
	model.TagType = tagRepos.TagType

	// request, err := NewBaseRequestWithModel(model, tagRepos.User)
	// if err != nil {
	// 	return m, err
	// }

	// response := model.Find(request)
	// if response.Error != nil {
	// 	return m, response.Error
	// }
	// tagRepos.AddList(response.List)

	// m.Compare(tagRepos)
	result := m.RemoveDuplicates()

	return result, nil
}

func (m *Tags) Compare(tagRepos *TagRepoList) {

	// for i, tag := range *m {
	// 	tag = tagRepos.Compare(tag)
	// 	(*m)[i] = tag
	// }
}

func (m *Tags) RemoveDuplicates() *Tags {
	keys := make(map[string]bool)
	var uniqueTags Tags
	for _, tag := range *m {
		if _, value := keys[tag.Key]; !value {
			keys[tag.Key] = true
			uniqueTags = append(uniqueTags, tag)
		}
	}
	return &uniqueTags
}

// AddOrReplace agrega un nuevo tag o reemplaza uno existente si ya existe uno con la misma clave
func (m *Tags) AddOrReplace(tag Tag) {
	for i, t := range *m {
		if t.Key == tag.Key {
			(*m)[i] = tag
			return
		}
	}
	*m = append(*m, tag)
}

func (m *Tags) Update(request *BaseRequest, language Language) (*Tags, error) {

	// tagRepo := &TagRepo{}
	// repoID := tagRepo.GetRepoID()
	// tagRepoList, err := NewTagRepoList(TagTypeMark, language, repoID, request.User)
	// if err != nil {
	// 	return m, err
	// }

	// tagRepoList.AddTags(m.Tags)

	println("•••••••••••••••••••••••••••••••••")
	println("Tags.Update Not implemented")
	println("•••••••••••••••••••••••••••••••••")
	return m, nil
}
