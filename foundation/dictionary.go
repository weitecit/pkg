package foundation

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"
)

type Dictionaries []Dictionary

func (m Dictionaries) ToRaw() []byte {
	raw, _ := json.Marshal(m)
	return raw
}

type Dictionary struct {
	Tag   `bson:",inline"`
	Group int `json:"group" bson:"group"`
}

func NewDictionary(tag Tag, group int) (*Dictionary, error) {
	m := &Dictionary{}
	m.Tag = tag
	m.Group = group

	err := m.Validate()
	if err != nil {
		return m, err
	}

	return m, nil
}

func (m *Dictionary) Validate() error {
	if utils.IsEmptyStr(m.Key) {
		return errors.New("Dictionary.Validate: key can not be empty")
	}

	return nil
}

func (m *Dictionary) ToJSON() string {
	o, err := json.MarshalIndent(&m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func (m *Dictionary) ToRaw() []byte {
	raw, _ := json.Marshal(m)
	return raw
}

type AuditType utils.Enum

const (
	AuditTypeNone     AuditType = ""
	AuditTypeInternal AuditType = "audit_type_internal"
	AuditTypeExternal AuditType = "audit_type_external"
)

func GetAuditType(name string) (AuditType, error) {
	switch name {
	case "audit_type_internal":
		return AuditTypeInternal, nil
	case "audit_type_external":
		return AuditTypeExternal, nil
	default:
		return AuditTypeNone, errors.New("AuditTypeNone.GetAuditType: invalid audit type: " + name)
	}
}

type DictionaryType utils.Enum

const (
	DictionaryTypeNone         DictionaryType = ""
	DictionaryTypeImpact       DictionaryType = "dictionary_type_impact"
	DictionaryTypeProbability  DictionaryType = "dictionary_type_probability"
	DictionaryTypeRating       DictionaryType = "dictionary_type_rating"
	DictionaryTypeRI           DictionaryType = "dictionary_type_ri"
	DictionaryTypeRC           DictionaryType = "dictionary_type_rc"
	DictionaryTypeRcStrength   DictionaryType = "dictionary_type_rc_strength"
	DictionaryTypeRcEvaluation DictionaryType = "dictionary_type_rc_evaluation"
	DictionaryTypeRcEfficiency DictionaryType = "dictionary_type_rc_efficiency"
)

func GetDictionaryType(name string) (DictionaryType, error) {
	switch name {
	case "dictionary_type_impact":
		return DictionaryTypeImpact, nil
	case "dictionary_type_probability":
		return DictionaryTypeProbability, nil
	case "dictionary_type_rating":
		return DictionaryTypeRating, nil
	case "dictionary_type_ri":
		return DictionaryTypeRI, nil
	case "dictionary_type_rc":
		return DictionaryTypeRC, nil
	case "dictionary_type_rc_strength":
		return DictionaryTypeRcStrength, nil
	case "dictionary_type_rc_evaluation":
		return DictionaryTypeRcEvaluation, nil
	case "dictionary_type_rc_efficiency":
		return DictionaryTypeRcEfficiency, nil
	default:
		return DictionaryTypeNone, errors.New("DictionaryType.GetDictionaryType: invalid dictionary type: " + name)
	}
}

type DictionaryRepo struct {
	ConfigType              ConfigType `json:"config_type" bson:"config_type"`
	BaseModel               `bson:",inline"`
	DictionaryType          DictionaryType `json:"dictionary_type" bson:"dictionary_type"`
	DictionarySecondaryType AuditType      `json:"dictionary_secondary_type" bson:"dictionary_secondary_type"`
	Dictionaries            Dictionaries   `json:"dictionaries" bson:"dictionaries"`
}

func NewDictionaryRepo(dictionaryType DictionaryType, dictionaries Dictionaries) (*DictionaryRepo, error) {
	m := &DictionaryRepo{}
	m.ConfigType = ConfigTypeDictionary
	m.DictionaryType = dictionaryType
	m.Dictionaries = dictionaries

	err := m.Validate()
	if err != nil {
		return m, err
	}

	return m, nil
}

func (m *DictionaryRepo) GetCollection() (name string, isGlobal bool) {
	return "config", false
}

func (m *DictionaryRepo) GetRepoType() RepoType {
	return RepoTypeMongoDB
}

func (m *DictionaryRepo) ToJSON() string {
	o, err := json.MarshalIndent(&m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func (m *DictionaryRepo) Validate() error {
	m.ConfigType = ConfigTypeDictionary

	if m.DictionaryType == DictionaryTypeNone {
		return errors.New("DictionaryRepo.Validate: dictionary type can not be empty")
	}

	return nil
}

func (m *DictionaryRepo) GetFindOptions(request *BaseRequest) *FindOptions {
	findOptions := m.GetBaseFindOptions(request)

	findOptions.AddEquals("config_type", ConfigTypeDictionary)

	if m.DictionaryType != DictionaryTypeNone {
		findOptions.AddEquals("dictionary_type", m.DictionaryType)
	}

	if m.DictionarySecondaryType != AuditTypeNone {
		findOptions.AddEquals("dictionary_secondary_type", m.DictionarySecondaryType)
	}

	return findOptions
}

func (m *DictionaryRepo) Find(request *BaseRequest) BaseResponse {
	request.SetFindOptions(m.GetFindOptions(request))
	request.Model = m
	request.List = []*DictionaryRepo{}

	return m.BaseFind(request)
}

func (m *DictionaryRepo) FindOne(request *BaseRequest) BaseResponse {
	if m.ID != nil {
		response := m.BaseFindOne(*request)
		return response
	}

	request.List = []*Dictionary{}

	response := m.Find(request)
	if response.Error != nil {
		return response
	}

	if response.TotalRows == 0 {
		return NewBaseResponseFromError(errors.New("DictionaryRepo.FindOne: no record found"))
	}
	if response.TotalRows > 1 {
		return NewBaseResponseFromError(errors.New("DictionaryRepo.FindOne: more than one record found"))
	}

	return response
}

func (m *DictionaryRepo) Update(request *BaseRequest) BaseResponse {

	err := m.Validate()
	if err != nil {
		return NewBaseResponseFromError(err)
	}

	request.Model = m

	return m.BaseUpdate(*request)
}

func (m *DictionaryRepo) FindOrCreate(request *BaseRequest) BaseResponse {
	findRequest := *request

	response := m.Find(&findRequest)
	if response.Error != nil {
		return response
	}

	if response.TotalRows == 1 {
		err := response.GetFirst(m)
		if err != nil {
			return BaseResponse{Error: err}
		}
		return response
	}

	if response.TotalRows > 1 {
		return BaseResponse{Error: errors.New("DictionaryRepo.FindOrCreate: more than one file found")}
	}

	return m.Update(request)
}

func (m *DictionaryRepo) UpdateMany(request *BaseRequest, values map[string]interface{}) BaseResponse {
	if request.Model == nil {
		return NewBaseResponseFromError(errors.New("DictionaryRepo.UpdateMany: detailModel is nil"))
	}

	request.SetFindOptions(m.GetFindOptions(request))

	response := m.BaseUpdateMany(*request, values)

	return response
}

func (m *DictionaryRepo) Delete(request *BaseRequest) BaseResponse {
	request.SetFindOptions(m.GetFindOptions(request))
	request.Model = m

	return m.BaseDelete(*request)
}

func (m *DictionaryRepo) GetDictionaryByTag(tag Tag) (dictionary Dictionary, err error) {
	for _, d := range m.Dictionaries {
		if d.HasSameKey(tag) {

			return dictionary, nil
		}
	}

	return dictionary, errors.New("DictionaryRepo.GetDictionaryByTag: no dictionary found")
}

func (m *DictionaryRepo) GetDictionaryByTagValueOrNumber(value string, number int) (dictionary *Dictionary, err error) {
	if utils.IsEmptyStr(value) && number == 0 {
		return dictionary, nil
	}

	if !utils.IsEmptyStr(value) && number > 0 {
		for _, d := range m.Dictionaries {
			if utils.NormalizeForTag(d.Value) == utils.NormalizeForTag(value) && d.Group == number {
				return &d, nil
			}
		}
	}

	if !utils.IsEmptyStr(value) {
		for _, d := range m.Dictionaries {
			if utils.NormalizeForTag(d.Value) == utils.NormalizeForTag(value) {
				return &d, nil
			}
		}
	}

	if number > 0 {
		for _, d := range m.Dictionaries {
			if d.Group == number {
				return &d, nil
			}
		}
	}

	return dictionary, errors.New("DictionaryRepo.GetDictionaryByTagValueOrNumber: no dictionary found")
}

func GetDictionaryRepo(request *BaseRequest, dictionaryType DictionaryType, auditType AuditType, language Language) (*DictionaryRepo, error) {
	dictionaryRepo := &DictionaryRepo{
		ConfigType:              ConfigTypeDictionary,
		DictionaryType:          dictionaryType,
		DictionarySecondaryType: auditType,
	}

	if language == "" {
		return dictionaryRepo, errors.New("DictionaryRepo.GetDictionaryRepo: language can not be empty")
	}

	language, err := language.Normalize()
	if err != nil {
		return dictionaryRepo, err
	}

	dictionaryRepo.Language = language

	request, err = request.Clone(dictionaryRepo)
	if err != nil {
		return dictionaryRepo, err
	}
	response := dictionaryRepo.Find(request)
	if response.Error != nil {
		return dictionaryRepo, response.Error
	}
	if response.TotalRows == 0 {
		dictionry_err := fmt.Sprintf("DictionaryRepo.GetDictionaryRepo: no dictionary found for %s", dictionaryType)
		return dictionaryRepo, errors.New(dictionry_err)
	}
	err = response.GetFirst(dictionaryRepo)

	return dictionaryRepo, err
}
