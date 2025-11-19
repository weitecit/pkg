package foundation

import (
	"encoding/json"
	"errors"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BaseRequest struct {
	ID               *primitive.ObjectID
	SourceID         string
	PageSize         int64
	CurrentPage      int64
	Order            *Orders
	SearchTerms      SearchTerms
	findOptions      FindOptions
	Repo             Repository
	SpaceID          string
	IncludeDeleted   bool
	NotNilFields     []string
	Language         Language
	IDs              interface{}
	ExcludedIDs      []*primitive.ObjectID
	User             User
	Model            RepositoryModel
	List             interface{}
	TargetCollection string
	// Puede cambiar el campo ID que por defecto es "_id"
	QueryField  string
	QueryValues interface{}
	Target      string
	DateRange
	HTTPClient interface{}
	Max        int
	Min        int
	Total      int
	Message    string
}

type SearchTerms []string

func (m SearchTerms) Add(term string) SearchTerms {
	return utils.FindOrAppendStrRaw(m, term)
}

func (m *BaseRequest) ToJSON() string {
	o, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func NewBaseRequest(model RepositoryModel, repo Repository, user User) (*BaseRequest, error) {

	baseRequest := &BaseRequest{
		User:  user,
		Repo:  repo,
		Order: &Orders{},
		Model: model,
	}

	if !utils.HasValidID(user.ID) {
		return baseRequest, errors.New("NewBaseRequest: User is id required")
	}

	if baseRequest.Repo == nil {
		return baseRequest, errors.New("NewBaseRequest: Repository is required")
	}

	return baseRequest, nil
}

func NewBaseRequestWithModel(model RepositoryModel, user User) (*BaseRequest, error) {
	repo, err := NewRepositoryFromModel(model, user.Connection)
	if err != nil {
		return &BaseRequest{}, err
	}

	request, err := NewBaseRequest(model, repo, user)
	if err != nil {
		return request, err
	}

	request.Model = model

	return request, nil
}

func (m *BaseRequest) HasIDs() bool {
	if m.IDs == nil {
		return false
	}

	if len(m.GetObjectsIDs()) == 0 {
		return false
	}
	return true
}

func (m *BaseRequest) GetObjectsIDs() []*primitive.ObjectID {
	if m.IDs == nil {
		return []*primitive.ObjectID{}
	}

	// check if is a slice of string and convert to slice of ObjectID
	switch m.IDs.(type) {
	case []*primitive.ObjectID:
		return m.IDs.([]*primitive.ObjectID)
	case []string:
		ids := []*primitive.ObjectID{}
		for _, id := range m.IDs.([]string) {
			oid, err := primitive.ObjectIDFromHex(id)
			if err != nil {
				log.Err(err)
				continue
			}
			ids = append(ids, &oid)
		}
		return ids
	default:
		log.Err(errors.New("BaseRequest.GetObjectsIDs: Invalid type"))
		return []*primitive.ObjectID{}
	}

}

func (m *BaseRequest) Validate() error {
	if m.Model == nil {
		return errors.New("BaseRequest.Validate: Model is required")
	}
	if m.Repo == nil {
		return errors.New("BaseRequest.Validate: Repository is required")
	}
	if m.User.Username == "" && m.User.ID == nil {
		return errors.New("BaseRequest.Validate: User is required")
	}

	return nil
}

func (m *BaseRequest) AddOrderAsc(fields ...string) {

	if m.Order == nil {
		m.Order = &Orders{}
	}

	for _, field := range fields {
		m.addOrder(field, true)
	}

}

func (m *BaseRequest) AddOrderDesc(fields ...string) {

	if m.Order == nil {
		m.Order = &Orders{}
	}

	for _, field := range fields {
		m.addOrder(field, false)
	}

}

func (m *BaseRequest) addOrder(field string, asc bool) {

	if m.Order == nil {
		m.Order = &Orders{}
	}

	order := &Order{}
	order.Field = field
	order.Direction = 1

	if !asc {
		order.Direction = -1
	}
	m.Order.Add(*order)

}

func (m *BaseRequest) SetFindOptions(findOptions *FindOptions) {
	m.findOptions = *findOptions
}

func (m *BaseRequest) GetFindOptions() *FindOptions {
	return &m.findOptions
}

func (m *BaseRequest) GetRepoRequest() RepoRequest {

	return RepoRequest{
		PageSize:         m.PageSize,
		CurrentPage:      m.CurrentPage,
		User:             m.User,
		Model:            m.Model,
		FindOptions:      m.findOptions,
		List:             m.List,
		Pipeline:         m.findOptions.Pipeline,
		TargetCollection: m.TargetCollection,
	}
}

func (m *BaseRequest) Clone(model RepositoryModel) (*BaseRequest, error) {

	repo, err := CloneRepository(m.Repo, model)
	if err != nil {
		return &BaseRequest{}, err

	}

	cloneRequest, err := NewBaseRequest(model, repo, m.User)

	if err != nil {
		return cloneRequest, err
	}
	cloneRequest.IDs = m.IDs
	cloneRequest.ExcludedIDs = m.ExcludedIDs
	cloneRequest.IncludeDeleted = m.IncludeDeleted
	cloneRequest.Language = m.Language
	cloneRequest.Order = m.Order
	cloneRequest.PageSize = m.PageSize
	cloneRequest.CurrentPage = m.CurrentPage
	cloneRequest.SearchTerms = m.SearchTerms
	cloneRequest.SourceID = m.SourceID
	cloneRequest.findOptions = m.findOptions
	cloneRequest.QueryField = m.QueryField
	cloneRequest.QueryValues = m.QueryValues
	cloneRequest.Total = m.Total
	cloneRequest.Max = m.Max
	cloneRequest.Min = m.Min
	cloneRequest.DateRange = m.DateRange
	cloneRequest.HTTPClient = m.HTTPClient

	return cloneRequest, nil
}

func (m *BaseRequest) CloneModelToNewDomain(domain string) (*BaseRequest, error) {

	request, err := m.Clone(m.Model)
	if err != nil {
		return request, err
	}
	request.Model.BecomeNewButKeepID()
	request.Model.SetRepoID(domain)
	err = request.Repo.SetRepoID(domain)
	if err != nil {
		return request, err
	}

	return request, nil
}
