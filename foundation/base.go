package foundation

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type JSON map[string]interface{}

func (m JSON) Has(field string) bool {
	_, ok := m[field]
	return ok
}

func (m JSON) Get(field string) interface{} {
	return m[field]
}

func (m JSON) Set(field string, value interface{}) {
	m[field] = value
}

type TagKey string

func ParseMapToStruct(data map[string]interface{}, model interface{}) error {
	raw, err := bson.Marshal(data)
	if err != nil {
		return err
	}
	return bson.Unmarshal(raw, model)
}

type ConfigType utils.Enum

const (
	ConfigTypeNone       ConfigType = ""
	ConfigTypeTag        ConfigType = "config_type_tag"
	ConfigTypeDictionary ConfigType = "config_type_dictionary"
)

type Action utils.Enum

const (
	ActionUpdate Action = "ActionUpdate"
	ActionDelete Action = "ActionDelete"
	ActionNone   Action = ""
)

type ListItem struct {
	ID       *primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	ParentID *primitive.ObjectID `json:"parent_id" bson:"parent_id"`
	UserLogs `bson:",inline"`
}

type ListItemModel interface {
	GetID() string
	SetID(*primitive.ObjectID)
	GetParentID() *primitive.ObjectID
	Validate() error
	Calculate(dictionaries Dictionaries)
	SetUserLogs(User)
}

func (m *ListItem) GetID() string {
	if m.ID == nil {
		return ""
	}
	return m.ID.Hex()
}

func (m *ListItem) Validate() error {

	return nil
}

func UpdateBaseList(model ListItemModel, list interface{}) error {
	resultsVal := reflect.ValueOf(list)
	if resultsVal.Kind() != reflect.Ptr {
		return fmt.Errorf("UpdateBaseList: results argument must be a pointer to a slice, but was a %s", resultsVal.Kind())
	}

	sliceVal := resultsVal.Elem()
	if sliceVal.Kind() == reflect.Interface {
		sliceVal = sliceVal.Elem()
	}

	if sliceVal.Kind() != reflect.Slice {
		return fmt.Errorf("UpdateBaseList: results argument must be a pointer to a slice, but was a pointer to %s", sliceVal.Kind())
	}

	len := sliceVal.Len()

	for i := 0; i < len; i++ {
		item := sliceVal.Index(i).Interface().(ListItemModel)
		if utils.HaveSameStrIDs(item.GetID(), model.GetID()) {
			// replace
			sliceVal.Index(i).Set(reflect.ValueOf(model))
			resultsVal.Elem().Set(sliceVal.Slice(0, len))
			return nil
		}
	}

	sliceVal = reflect.Append(sliceVal, reflect.ValueOf(model))
	len++
	resultsVal.Elem().Set(sliceVal.Slice(0, len))

	return nil
}

type Synchronizable interface {
	LastSync() *time.Time
	Sync(model interface{}) error
	GetExternalID() string
	Delete() RepoResponse
}

type SourceType utils.Enum

const (
	SourceTypeEnumNone     SourceType = ""
	SourceTypeImportedData SourceType = "imported_data"
	// Same ID as DomainID Cloned
	SourceTypeDomainClone SourceType = "domain_clone"
)

type UserLogs struct {
	CreatedBy  *UserLog `json:"created_by,omitempty" bson:"created_by,omitempty"`
	UpdatedBy  *UserLog `json:"updated_by,omitempty" bson:"updated_by,omitempty"`
	DeletedBy  *UserLog `json:"deleted_by,omitempty" bson:"deleted_by,omitempty"`
	LastAccess *UserLog `json:"last_access,omitempty" bson:"last_access,omitempty"`
	LockedBy   *UserLog `json:"locked_by,omitempty" bson:"locked_by,omitempty"`
}

type MiniModel struct {
	ID   string `json:"id" bson:"id"`
	Name string `json:"name" bson:"name"`
}

type BaseModel struct {
	ID       *primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	UserLogs `bson:",inline"`
	Language Language `json:"language,omitempty" bson:"language,omitempty"`
	// Refers to other system for synchronization
	ExternalID string     `json:"external_id,omitempty" bson:"external_id,omitempty"`
	SyncAt     *time.Time `json:"sync_at,omitempty" bson:"sync_at,omitempty"`
	Labels     *Labels    `json:"labels" bson:"labels,omitempty"`
	Tags       *Tags      `json:"tags" bson:"tags,omitempty"`
	// Relations with others in same collection
	FamilyID string `json:"family_id,omitempty" bson:"family_id,omitempty"`
	// Store domain of the model
	RepoID string `json:"repo_id,omitempty" bson:"repo_id,omitempty"`
	// REMOVE
	OriginRepoID string `json:"origin_repo_id,omitempty" bson:"origin_repo_id,omitempty"`
	// Interface connection with other collections
	SourceID   string     `json:"source_id,omitempty" bson:"source_id"`
	SourceType SourceType `json:"source_tlabype,omitempty" bson:"source_type,omitempty"`
	Version    int        `json:"version,omitempty" bson:"version,omitempty"`
	Touched    bool       `json:"touched,omitempty" bson:"-"`
	ParentID   string     `json:"parent_id" bson:"parent_id,omitempty"`
	LabelsNot  *Labels    `json:"labels_not,omitempty" bson:"-"`
}

func (m *BaseModel) CompareLabels(labels Labels) bool {

	if m.Labels == nil {
		m.Labels = &Labels{}
	}

	if m.Labels.Total() != labels.Total() {
		return false
	}

	for _, label := range labels {
		if !m.Labels.Has(label) {
			return false
		}
	}
	return true

}

func (m *BaseModel) GetIDStr() string {
	if m.ID == nil {
		return ""
	}
	return m.ID.Hex()
}

func (m *BaseModel) GetID() (interface{}, error) {
	if m.ID == nil {
		return nil, errors.New("BaseModel.GetID: ID is nil")
	}
	return m.ID, nil
}

func (m *BaseModel) SetID(id string) error {
	if id == "" {
		return errors.New("BaseModel.SetID: ID is nil")
	}
	var err error
	m.ID, err = utils.GetObjectIdFromString(id)
	if err != nil {
		return err
	}
	return nil
}

func (m *BaseModel) IsNew() bool {
	return m.CreatedBy == nil
}

func (m *BaseModel) IsDeleted() bool {
	return m.DeletedBy != nil
}

func (m *BaseModel) SetDeleted(user User) {
	m.DeletedBy = user.GetUserLog()
}

func (m *BaseModel) SetRecover(user User) {
	m.DeletedBy = nil
}

func (m BaseModel) ToJSON(model interface{}) string {
	o, err := json.MarshalIndent(model, "", "\t")
	if err != nil {
		return "Error in conversion"
	}
	return string(o)
}

func (m BaseModel) ToRaw(model interface{}) []byte {
	raw, _ := json.Marshal(model)
	return raw
}

func (m *BaseModel) SetCreated(user User) {

	if m.ID == nil {
		m.ID = utils.NewID()
	}

	m.CreatedBy = user.GetUserLog()
}

func (m *BaseModel) SetUpdated(user User) {
	m.UpdatedBy = user.GetUserLog()
}

func (m *BaseModel) ValidateBase() error {
	// if m.FirmID == nil {
	// 	return errors.New("ValidateBase: FirmID is empty")
	// }
	return nil
}

// For nil booleans
func (m *BaseModel) True() *bool {
	b := true
	return &b
}

func (m *BaseModel) False() *bool {
	b := false
	return &b
}
func (m *BaseModel) IsTrue(condition *bool) bool {
	if condition == nil {
		return false
	}
	return condition == m.True()
}

func (m *BaseModel) IsFalse(condition *bool) bool {
	if condition == nil {
		return true
	}
	return condition == m.False()
}

func (m *BaseModel) NewBool(condition bool) *bool {
	if condition {
		return m.True()
	}
	return m.False()
}

type Model interface {
	ToJSON()
	ToRaw()
}

type BulkList struct {
	NewList    []interface{}
	UpdateList []interface{}
}

// newbulks
func NewBulkList() *BulkList {
	return &BulkList{
		NewList:    make([]interface{}, 0),
		UpdateList: make([]interface{}, 0),
	}
}

func (m *BaseModel) GetBaseFindOptions(request *BaseRequest) *FindOptions {
	findOptions := NewFindOptions()
	findOptions.Order = request.Order

	if m.ID != nil {
		findOptions.AddEquals("_id", m.ID)
	}

	if m.RepoID != "" {
		findOptions.AddEquals("repo_id", m.RepoID)
	}

	if m.ExternalID != "" {
		if m.ExternalID == "-1" {
			findOptions.AddEquals("external_id", bson.M{"$exists": false})
		} else if m.ExternalID == "0" {
			findOptions.AddEquals("external_id", bson.M{"$exists": true})
		} else {
			findOptions.AddEquals("external_id", m.ExternalID)
		}
	}

	if request.ExcludedIDs != nil {
		findOptions.AddComplex("_id", FilterOperatorNotIn, request.ExcludedIDs)
	}

	if m.Language != "" {
		lang, err := m.Language.Normalize()
		if err != nil {
			log.Err(err)
		}
		findOptions.AddEquals("language", lang)
	}

	if m.SourceID != "" {
		findOptions.AddEquals("source_id", m.SourceID)
	}

	if m.FamilyID != "" {
		findOptions.AddEquals("family_id", m.FamilyID)
	}

	if request.HasIDs() && m.ID == nil {
		queryField := "_id"
		if request.QueryField != "" {
			queryField = request.QueryField
			request.QueryField = ""
		}
		// check ids are string or ObjectID and convert to ObjectID
		if _, ok := request.IDs.([]*primitive.ObjectID); !ok {
			if _, ok := request.IDs.([]string); ok {
				ids := []*primitive.ObjectID{}
				for _, id := range request.IDs.([]string) {
					oid, err := primitive.ObjectIDFromHex(id)
					if err != nil {
						log.Err(err)
						continue
					}
					ids = append(ids, &oid)
				}
				request.IDs = ids
			}
		}

		findOptions.AddComplex(queryField, FilterOperatorIn, request.IDs)
	}

	if request.QueryField != "" && !request.HasIDs() && !utils.IsEmptyIntefaceList(request.List) {
		findOptions.AddComplex(request.QueryField, FilterOperatorIn, request.List)
		request.QueryField = ""
	}

	if m.IsLabeled() {
		findOptions.AddComplex("labels", FilterOperatorIn, m.GetLabels())
	}

	if m.LabelsNot != nil {
		findOptions.AddComplex("labels", FilterOperatorNotIn, m.LabelsNot.ToArray())
	}

	return findOptions
}

func (m *BaseModel) Trace(err error) error {
	if err == nil {
		return errors.New("Models.Err: err is nil")
	}
	log.Err(err)
	_, err = LogTrace(m, err)
	return err
}

func (m *BaseModel) Err(err error) error {
	if err == nil {
		return errors.New("Models.Err: err is nil")
	}
	log.Err(err)
	_, err = LogErr(m, err)
	return err
}

func (m *BaseModel) PrepareForSync(externalID interface{}) error {
	if externalID == nil {
		err := errors.New("BaseModel.PrepareForSync: No external ID")
		return err
	}
	m.ExternalID = fmt.Sprintf("%v", externalID)
	now := time.Now()
	m.SyncAt = &now
	return nil
}

func (m *BaseModel) LastSync() *time.Time {
	return m.SyncAt
}

func (m *BaseModel) GetExternalID() string {
	return m.ExternalID
}

func (m *BaseModel) BecomeNew() {
	m.ID = nil
	m.CreatedBy = nil
	m.UpdatedBy = nil
	m.DeletedBy = nil
}

func (m *BaseModel) BecomeNewButKeepID() {
	m.CreatedBy = nil
	m.UpdatedBy = nil
	m.DeletedBy = nil
}

// Esta función es válida para modelos NO globales
func (m *BaseModel) IsEmpty() bool {

	println("•••••••••••••••••••••••••••••••••")
	println("IsEmpty Not implemented")
	println("•••••••••••••••••••••••••••••••••")

	return false
}
func (m *BaseModel) Label(labels ...Label) {

	if m.Labels == nil {
		m.Labels = &Labels{}
	}

	for _, label := range labels {
		if label == LabelNone {
			continue
		}
		if !m.HasLabel(label) {
			*m.Labels = append(*m.Labels, label)
		}
	}
}

func (m *BaseModel) LabelIf(label Label, b interface{}) {
	if utils.Bool(b) {
		m.Label(label)
	}
}

func (m *BaseModel) LabelFromStrings(strings ...string) {

	if m.Labels == nil {
		m.Labels = &Labels{}
	}

	for _, stringValue := range strings {
		if !m.HasLabel(Label(stringValue)) {
			m.Label(Label(stringValue))
		}
	}
}

func (m *BaseModel) HasLabels(labels []Label) bool {
	if m.Labels == nil {
		m.Labels = &Labels{}
	}

	if len(labels) == 0 {
		return true
	}

	for _, item := range labels {
		matched := false
		for _, label := range *m.Labels {
			if item == label {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

func (m *BaseModel) HasLabel(label Label) bool {
	if m.Labels == nil {
		m.Labels = &Labels{}
	}

	for _, item := range *m.Labels {
		if item == label {
			return true
		}
	}

	return false
}

func (m *BaseModel) UnLabel(labels ...Label) {
	if m.Labels == nil {
		m.Labels = &Labels{}
		return
	}

	for _, label := range labels {
		for i, item := range *m.Labels {
			if item == label {
				*m.Labels = append((*m.Labels)[:i], (*m.Labels)[i+1:]...)
			}
		}
	}
}

func (m *BaseModel) IsLabeled() bool {
	if m.Labels == nil {
		m.Labels = &Labels{}
		return false
	}
	return m.LabelTotal() > 0
}

func (m *BaseModel) LabelTotal() int {
	if m.Labels == nil {
		m.Labels = &Labels{}
		return 0
	}
	return len(*m.Labels)
}

func (m *BaseModel) GetLabels() []Label {
	if m.Labels == nil {
		m.Labels = &Labels{}
	}
	return *m.Labels
}

func (m *BaseModel) BaseFind(request *BaseRequest) BaseResponse {

	err := request.Validate()
	if err != nil {
		return NewBaseResponseFromError(err)
	}

	repoRequest := request.GetRepoRequest()

	result := request.Repo.Find(repoRequest)

	// Enable for logs bbdd indexes
	// println("•••••••••••••••••••••••••••••••••")
	// fmt.Println("repoRequest.ToJSON()", repoRequest.ToJSON())
	// println("•••••••••••••••••••••••••••••••••")

	response := NewBaseResponseFromRepoResponse(result)

	return response
}

func (m *BaseModel) BaseCount(request *BaseRequest) BaseResponse {

	err := request.Validate()
	if err != nil {
		return NewBaseResponseFromError(err)
	}

	request.PageSize = 0

	repoRequest := request.GetRepoRequest()

	result := request.Repo.Count(repoRequest)

	response := NewBaseResponseFromRepoResponse(result)

	return response
}

func (m *BaseModel) BaseFindOne(request BaseRequest) BaseResponse {

	response := NewBaseResponse()

	if m.ID == nil {
		response.Error = errors.New("BaseModel.BaseFindOne: ID is nil")
		return *response
	}

	_, err := request.Model.GetID()
	if err != nil {
		request.Model.SetID(m.GetIDStr())
	}

	err = request.Validate()
	if err != nil {
		response.Error = err
		return *response
	}

	repoRequest := request.GetRepoRequest()

	result := request.Repo.FindOne(repoRequest)

	response.CurrentPage = result.CurrentPage
	response.Error = result.Error
	response.List = []interface{}{request.Model}
	response.PageSize = result.PageSize
	response.TotalPages = result.TotalPages
	response.TotalRows = result.TotalRows

	return *response
}

func (m *BaseModel) BaseUpdate(request BaseRequest) BaseResponse {

	if request.Repo == nil {
		return BaseResponse{Error: errors.New("BaseModel.BaseUpdate: Repo is required")}
	}

	repoRequest := RepoRequest{
		Model: request.Model,
		User:  request.User,
	}

	m.Version++

	repoResponse := request.Repo.Update(repoRequest)

	response := NewBaseResponseFromRepoResponse(repoResponse)

	return response
}

func (m *BaseModel) BaseUpdateMany(request BaseRequest, values map[string]interface{}) BaseResponse {

	if request.Repo == nil {
		return BaseResponse{Error: errors.New("BaseModel.BaseUpdateMany: Repo is required")}
	}

	repoRequest := RepoRequest{
		Model:       request.Model,
		User:        request.User,
		FindOptions: request.findOptions,
	}

	repoResponse := request.Repo.UpdateMany(repoRequest, values)

	response := NewBaseResponseFromRepoResponse(repoResponse)

	return response
}

func (m *BaseModel) BaseUpdateField(request BaseRequest, field string, value interface{}) BaseResponse {

	if request.Repo == nil {
		return BaseResponse{Error: errors.New("BaseModel.BaseUpdateField: Repo is required")}
	}

	repoRequest := RepoRequest{
		Model:       request.Model,
		User:        request.User,
		FindOptions: request.findOptions,
	}

	repoResponse := request.Repo.UpdateField(repoRequest, field, value)

	response := NewBaseResponseFromRepoResponse(repoResponse)

	return response
}

func (m *BaseModel) BaseSwitchItemInArray(request *BaseRequest, field string, value string) BaseResponse {

	if request.Repo == nil {
		return BaseResponse{Error: errors.New("BaseModel.BaseSwitchItemInArray: Repo is required")}
	}

	repoRequest := RepoRequest{
		User: request.User,
		ID:   m.GetIDStr(),
	}

	result := request.Repo.SwitchItemInArray(repoRequest, field, value)
	if result.Error != nil {
		return NewBaseResponseFromError(result.Error)
	}

	response := NewBaseResponseFromRepoResponse(result)

	return response
}

func (m *BaseModel) BaseRemoveItemInArray(request *BaseRequest, field string, value string) BaseResponse {

	if request.Repo == nil {
		return BaseResponse{Error: errors.New("BaseModel.BaseRemoveItemInArray: Repo is required")}
	}

	repoRequest := RepoRequest{
		User: request.User,
		ID:   m.GetIDStr(),
	}

	result := request.Repo.RemoveItemInArray(repoRequest, field, value)
	if result.Error != nil {
		return NewBaseResponseFromError(result.Error)
	}

	response := NewBaseResponseFromRepoResponse(result)

	return response
}

func (m *BaseModel) BaseAddItemInArray(request *BaseRequest) BaseResponse {

	if request.Repo == nil {
		return BaseResponse{Error: errors.New("BaseModel.BaseAddItemInArray: Repo is required")}
	}

	repoRequest := RepoRequest{
		User: request.User,
		ID:   m.GetIDStr(),
	}

	result := request.Repo.AddItemInArray(repoRequest, "notifications", request.ID.Hex())
	if result.Error != nil {
		return NewBaseResponseFromError(result.Error)
	}

	response := NewBaseResponseFromRepoResponse(result)

	return response
}

func (m *BaseModel) BaseDelete(request BaseRequest) BaseResponse {

	response := NewBaseResponse()

	err := request.Validate()
	if err != nil {
		response.Error = err
		return *response
	}

	repoRequest := request.GetRepoRequest()

	result := request.Repo.Delete(repoRequest)

	response.Error = result.Error
	response.TotalRows = result.TotalRows

	return *response
}

func (m *BaseModel) BaseDeleteSoft(request BaseRequest) BaseResponse {

	response := NewBaseResponse()

	err := request.Validate()
	if err != nil {
		response.Error = err
		return *response
	}

	repoRequest := request.GetRepoRequest()

	result := request.Repo.DeleteSoft(repoRequest)

	response.Error = result.Error
	response.TotalRows = result.TotalRows

	return *response

}

func (m *BaseModel) BaseRemoveField(request BaseRequest, field string) BaseResponse {

	response := NewBaseResponse()

	err := request.Validate()
	if err != nil {
		response.Error = err
		return *response
	}

	repoRequest := request.GetRepoRequest()

	result := request.Repo.RemoveField(repoRequest, field)

	response.Error = result.Error
	response.TotalRows = result.TotalRows

	return *response
}

func (m *BaseModel) GetRepoID() string {
	return m.RepoID
}

func (m *BaseModel) SetRepoID(value string) {
	m.RepoID = value
}

func (m *BaseResponse) AddID(id *primitive.ObjectID) error {
	// for _, x := range m.IDs {
	// 	if utils.HaveSameIDs(x, id) {
	// 		return errors.New("BaseResponse.AddID: id already exists")
	// 	}
	// }
	// m.IDs = append(m.IDs, id)
	println("•••••••••••••••••••••••••••••••••")
	println("BaseResponse.AddID: Not implemented")
	println("•••••••••••••••••••••••••••••••••")
	return nil
}

// REFACTOR: generalización de Find y otros pero peta

// func (m *BaseResponse) GetIDs() []*primitive.ObjectID {

// 	result := []*primitive.ObjectID{}

// 	if m.List == nil {
// 		return result
// 	}

// 	for _, item := range m.List.([]*BaseModel) {
// 		result = append(result, item.ID)
// 	}

// 	return result
// }

// func (m *BaseResponse) Find(item interface{}) (interface{}, error) {

// 	if item == nil {
// 		return item, errors.New("BaseResponse.Find: item is nil")
// 	}

// 	idValue, err := model.Get(item, "ID")
// 	if err != nil {
// 		return item, err
// 	}

// 	id := idValue.(*primitive.ObjectID)

// 	for _, i := range m.List.([]*BaseModel) {
// 		idValue2, err := model.Get(i, "ID")
// 		if err != nil {
// 			continue
// 		}
// 		id2 := idValue2.(*primitive.ObjectID)
// 		if utils.HaveSameIDs(id, id2) {
// 			return i, nil
// 		}
// 	}
// 	return item, errors.New("BaseResponse.Find: item not found in list")
// }

// TESTING +++++++++++++++++++++++++++++++++
type SendEmail struct {
	From string
}

func (sender *SendEmail) Send(to, subject, body string) error {
	// It sends an email here, and perhaps returns an error.

	return nil
}

// type CustomerWelcome struct{}

// func (welcomer *CustomerWelcome) Welcome(name, email string) error {
// 	body := fmt.Sprintf("Hi, %s!", name)
// 	subject := "Welcome"
// 	emailer := &SendEmail{
// 		From: "hi@welcome.com",
// 	}
// 	return emailer.Send(email, subject, body)
// }

type EmailSender interface {
	Send(to, subject, body string) error
}

type CustomerWelcome struct {
	Email EmailSender
}

func (welcomer *CustomerWelcome) Welcome(name, email string) error {
	body := fmt.Sprintf("Hi, %s!", name)
	subject := "Welcome"

	return welcomer.Email.Send(email, subject, body)
}

func test() {
	emailer := &SendEmail{
		From: "hi@welcome.com",
	}
	welcomer := &CustomerWelcome{
		Email: emailer,
	}
	err := welcomer.Welcome("Bob", "bob@smith.com")
	// check error...
	if err != nil {

	}
}
