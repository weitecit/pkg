package foundation

import (
	"encoding/json"
	"errors"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"
)

var ConnectionPool = []Repository{}

type FilterOperator utils.Enum

const (
	FilterOperatorEquals                    FilterOperator = "="
	FilterOperatorEqualsWithCaseInsensitive FilterOperator = "equal_ci"
	FilterOperatorNotEquals                 FilterOperator = "!="
	FilterOperatorIn                        FilterOperator = "in"
	FilterOperatorNotIn                     FilterOperator = "not_in"
	FilterOperatorSize                      FilterOperator = "size"
	FilterOperatorGreat                     FilterOperator = "greater"
	FilterOperatorLess                      FilterOperator = "less"
	FilterOperatorGreatOrEqual              FilterOperator = "greater_or_equal"
	FilterOperatorLessOrEqual               FilterOperator = "less_or_equal"
	FilterOperatorAll                       FilterOperator = "all"
	FilterOperatorContains                  FilterOperator = "contains"
	FilterOperatorGroupsOfArrays            FilterOperator = "groups_of_arrays"
	FilterOperatorGroupBy                   FilterOperator = "group_by"
	FilterOperatorNotNil                    FilterOperator = "not_nil"
	FilterOperatorNil                       FilterOperator = "nil"
)

type Filter struct {
	Key      string         `json:"key"`
	Operator FilterOperator `json:"operator"`
	Value    interface{}    `json:"value"`
}

type FilterOr []Filter

type Order struct {
	Field     string
	Direction int
}

type Orders []Order

func (m *Orders) Add(orders ...Order) {

	if m == nil {
		m = &Orders{}
	}

	for _, order := range orders {
		if !m.Has(order) {
			*m = append(*m, order)
		}
	}

}

func (m *Orders) Has(order Order) bool {

	if m == nil {
		m = &Orders{}
	}

	for _, item := range *m {
		if item == order {
			return true
		}
	}
	return false
}

func (m *Orders) HasByFields(fields ...string) bool {

	if m == nil {
		m = &Orders{}
	}

	for _, item := range *m {
		if utils.StringInArray(item.Field, fields) {
			return true
		}
	}
	return false
}

func (m *Orders) IsEmpty() bool {

	return m == nil || len(*m) == 0
}

func (m *Orders) List() []Order {
	return *m
}

type FindOptions struct {
	Filters   []Filter    `json:"filters"`
	FiltersOr []FilterOr  `json:"filters_or"`
	Order     *Orders     `json:"order"`
	Pipeline  interface{} `json:"pipeline"`
}

func (m *FindOptions) Remove(key string) {
	if m.Filters == nil {
		return
	}

	for i, item := range m.Filters {
		if item.Key == key {
			m.Filters = append(m.Filters[:i], m.Filters[i+1:]...)
			break
		}
	}
}

func (m *FindOptions) GetTotalOrders() int64 {
	totalOrders := 0
	if m.Order == nil {
		return int64(totalOrders)
	}

	for range *m.Order {
		totalOrders++
	}
	return int64(totalOrders)
}

func (m *FindOptions) AddComplex(name string, operation FilterOperator, value interface{}) {
	if name == "" {
		return
	}
	if m.Filters == nil {
		m.Filters = []Filter{}
	}

	m.Filters = append(m.Filters, Filter{Key: name, Value: value, Operator: operation})
}

func (m *FindOptions) AddNotNil(name string) {
	m.AddComplex(name, FilterOperatorNotNil, nil)
}

func (m *FindOptions) AddNil(name string) {
	m.AddComplex(name, FilterOperatorNil, nil)
}

func (m *FindOptions) AddEquals(name string, value interface{}) {
	m.AddComplex(name, FilterOperatorEquals, value)
}

// AddEqualsCI adds a filter to the FindOptions that matches the specified field with the given value, ignoring case.
func (m *FindOptions) AddEqualsCI(name string, value interface{}) {
	m.AddComplex(name, FilterOperatorEqualsWithCaseInsensitive, value)
}

func (m *FindOptions) AddNotEquals(name string, value interface{}) {
	m.AddComplex(name, FilterOperatorNotEquals, value)
}

func (m *FindOptions) AddLess(name string, value interface{}) {
	m.AddComplex(name, FilterOperatorLess, value)
}

func (m *FindOptions) AddLessOrEqual(name string, value interface{}) {
	m.AddComplex(name, FilterOperatorLessOrEqual, value)
}

func (m *FindOptions) AddGreat(name string, value interface{}) {
	m.AddComplex(name, FilterOperatorGreat, value)
}

func (m *FindOptions) AddGreatOrEqual(name string, value interface{}) {
	m.AddComplex(name, FilterOperatorGreatOrEqual, value)
}

func (m *FindOptions) AddRange(nameFrom string, valueFrom interface{}, nameTo string, valueTo interface{}) {
	m.AddComplex(nameFrom, FilterOperatorGreatOrEqual, valueFrom)
	m.AddComplex(nameTo, FilterOperatorLessOrEqual, valueTo)
}

func (m *FindOptions) AddIn(name string, value interface{}) {
	m.AddComplex(name, FilterOperatorIn, value)
}

func (m *FindOptions) AddAll(name string, value interface{}) {
	m.AddComplex(name, FilterOperatorAll, value)
}

func (m *FindOptions) AddOrderAsc(fields ...string) {

	if m.Order == nil {
		m.Order = &Orders{}
	}

	for _, field := range fields {
		m.addOrder(field, true)
	}

}

func (m *FindOptions) AddOrderDesc(fields ...string) {

	if m.Order == nil {
		m.Order = &Orders{}
	}

	for _, field := range fields {
		m.addOrder(field, false)
	}

}

func (m *FindOptions) addOrder(field string, asc bool) {

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

// func (m *FindOptions) addOrder(name string, ascending bool) {
// 	if m.Orders == nil {
// 		m.Orders = Orders{}
// 	}
// 	m.Add(Order{Key: name, Ascending: ascending})
// }

// func (m *FindOptions) AddOrder(name string) {
// 	m.addOrder(name, true)
// }

// func (m *FindOptions) AddOrderDesc(name string) {
// 	m.addOrder(name, false)
// }

func (m *FindOptions) AddMultiple(value FilterOr) {
	if len(value) == 0 {
		return
	}

	if m.FiltersOr == nil {
		m.FiltersOr = []FilterOr{}
	}

	m.FiltersOr = append(m.FiltersOr, value)
}

// AddSearchTerms adds a search filter for multiple terms across multiple fields.
// By default (combineOr = false): each field has its own OR group with all terms
//
//	Fields: ["title", "reference"], Terms: ["texto", "tres"]
//	Result: group 1: (title contains "texto" OR title contains "tres"), group 2: (reference contains "texto" OR reference contains "tres")
//	Query: (title contains "texto" OR title contains "tres") AND (reference contains "texto" OR reference contains "tres")
//
// If combineOr = true: all terms and fields are combined into a single OR group
//
//	Fields: ["title", "reference"], Terms: ["texto", "tres"]
//	Result: (title contains "texto" OR title contains "tres" OR reference contains "texto" OR reference contains "tres")
//	Query: (title contains "texto" OR title contains "tres" OR reference contains "texto" OR reference contains "tres")
func (m *FindOptions) AddSearchTerms(fields []string, searchTerms []string, operator FilterOperator, combineOr ...bool) {
	if len(fields) == 0 || len(searchTerms) == 0 {
		return
	}

	// Default to Contains if not specified
	if operator == "" {
		operator = FilterOperatorContains
	}

	// Initialize FiltersOr if nil
	if m.FiltersOr == nil {
		m.FiltersOr = []FilterOr{}
	}

	// Check if we should combine all into a single OR group
	shouldCombine := len(combineOr) > 0 && combineOr[0]

	if shouldCombine {
		// Combine all term + field combinations into ONE OR group
		filterOr := FilterOr{}
		for _, term := range searchTerms {
			for _, field := range fields {
				filterOr = append(filterOr, Filter{
					Key:      field,
					Operator: operator,
					Value:    term,
				})
			}
		}
		m.FiltersOr = append(m.FiltersOr, filterOr)
	} else {
		// Default: each field creates its own OR group (one group per field, all terms in that group)
		// fields: ["title", "reference"], terms: ["texto", "tres"]
		// Result: group1: (title texto OR title tres), group2: (reference texto OR reference tres)
		for _, field := range fields {
			filterOr := FilterOr{}
			for _, term := range searchTerms {
				filterOr = append(filterOr, Filter{
					Key:      field,
					Operator: operator,
					Value:    term,
				})
			}
			m.FiltersOr = append(m.FiltersOr, filterOr)
		}
	}
}

func NewFindOptions() *FindOptions {
	return &FindOptions{
		Filters:   []Filter{},
		FiltersOr: []FilterOr{},
		Order:     &Orders{},
	}
}

func (m *FindOptions) ToJSON() string {
	o, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func (m *FindOptions) filterIsEmpty() bool {

	if m.Filters == nil && m.FiltersOr == nil {
		return true
	}
	return false
}

type Sort struct {
	Name       string
	Descending bool
}

type RepositoryModel interface {
	GetID() (interface{}, error)
	SetID(string) error
	IsNew() bool
	BecomeNew()
	BecomeNewButKeepID()
	GetCollection() (name string, isGlobal bool)
	SetCreated(user User)
	SetUpdated(user User)
	SetDeleted(user User)
	GetRepoType() RepoType
	GetRepoID() string
	SetRepoID(value string)
	LabelFromStrings(strings ...string)
}

type RepoRequest struct {
	ID               string
	PageSize         int64
	SortBy           []Sort
	CurrentPage      int64
	User             User
	Model            RepositoryModel
	FindOptions      FindOptions
	List             interface{}
	RepoID           string
	Pipeline         interface{}
	TargetCollection string
}

func (m *RepoRequest) ToJSON() string {
	o, err := json.MarshalIndent(&m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

type RepoResponse struct {
	Error       error
	Errors      []error
	TotalRows   int64
	TotalPages  int64
	PageSize    int64
	CurrentPage int64
	List        interface{}
}

func (m *RepoResponse) ToJSON() string {
	o, err := json.MarshalIndent(&m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func NewRepoResponse() (*RepoResponse, error) {
	return &RepoResponse{}, nil
}

type Repository interface {
	Aggregate(request RepoRequest) RepoResponse
	Find(request RepoRequest) RepoResponse
	Count(request RepoRequest) RepoResponse
	FindOne(request RepoRequest) RepoResponse
	Update(request RepoRequest) RepoResponse
	UpdateMany(request RepoRequest, values map[string]interface{}) RepoResponse
	UpdateField(request RepoRequest, field string, value interface{}) RepoResponse
	SwitchItemInArray(request RepoRequest, field string, value string) RepoResponse
	AddItemInArray(request RepoRequest, field string, value string) RepoResponse
	RemoveItemInArray(request RepoRequest, field string, value string) RepoResponse
	Move(request RepoRequest) RepoResponse
	Delete(request RepoRequest) RepoResponse
	DeleteSoft(request RepoRequest) RepoResponse
	RemoveField(request RepoRequest, field string) RepoResponse
	GetFilter(filterOptions FindOptions) (map[string]interface{}, error)
	GetOrder(filterOptions FindOptions) map[string]interface{}
	GetType() RepoType
	GetRepoID() string
	GetDataBase() string
	GetConnection() string
	SetRepoID(value string) error
	RepoBackup(request RepoRequest, backupID string) RepoResponse
	RepoRestore(request RepoRequest, backupID string) RepoResponse
	DeleteDatabase(connection string, database string) error
}

type RepoType uint64

const (
	RepoTypeUnknown RepoType = iota
	RepoTypeMongoDB
)

func NewRepository(connection string, repoType RepoType, database string, collection string, isGlobal bool) (Repository, error) {

	switch repoType {
	case RepoTypeMongoDB:
		repo := NewMongoRepository(connection, database, collection, isGlobal)
		if repo.Error != nil {
			return nil, repo.Error
		}
		return &repo, nil
	default:
		return nil, errors.New("NewRepository: RepoType is not supported")
	}
}

func NewRepositoryFromModel(model RepositoryModel, connection string) (Repository, error) {
	if model == nil {
		return nil, errors.New("NewRepositoryFromModel: model is nil")
	}

	collection, isGlobal := model.GetCollection()

	return NewRepository(connection, model.GetRepoType(), model.GetRepoID(), collection, isGlobal)
}

func CloneRepository(repo Repository, model RepositoryModel) (Repository, error) {
	if model == nil {
		return nil, errors.New("CloneRepository: model is nil")
	}

	collection, isGlobal := model.GetCollection()

	repoID := model.GetRepoID()
	if utils.IsEmptyStr(repoID) {
		repoID = repo.GetRepoID()
	}

	return NewRepository(repo.GetConnection(), repo.GetType(), repoID, collection, isGlobal)
}
