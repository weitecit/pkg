package foundation

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"
)

type BaseResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	// Es un error crítico
	StrError string `json:"str_error"`
	Error    error  `json:"error"`
	// Son errores no críticos
	Errors        []error     `json:"errors"`
	TotalRows     int64       `json:"total_rows"`
	TotalPages    int64       `json:"total_pages"`
	PageSize      int64       `json:"page_size"`
	CurrentPage   int64       `json:"current_page"`
	List          interface{} `json:"list"`
	Token         string      `json:"token"`
	ExternalToken string      `json:"external_token"`
	Status        string      `json:"status"`
}

func (m *BaseResponse) ToJSON() string {
	o, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func NewBaseResponseFromRepoResponse(repoResponse RepoResponse) BaseResponse {
	baseResponse := NewBaseResponse()
	baseResponse.Error = repoResponse.Error
	baseResponse.Errors = repoResponse.Errors
	baseResponse.TotalRows = repoResponse.TotalRows
	baseResponse.TotalPages = repoResponse.TotalPages
	baseResponse.PageSize = repoResponse.PageSize
	baseResponse.CurrentPage = repoResponse.CurrentPage
	baseResponse.List = repoResponse.List

	return *baseResponse
}

func NewBaseResponseFromError(err error) BaseResponse {
	baseResponse := NewBaseResponse()
	baseResponse.SetError(err)

	return *baseResponse
}

func NewBaseResponseFromErrorStr(err string) BaseResponse {
	baseResponse := NewBaseResponse()
	baseResponse.SetError(errors.New(err))

	return *baseResponse
}

func NewBaseResponseFromErrorWithCode(err error, code int) BaseResponse {
	baseResponse := NewBaseResponse()
	baseResponse.SetError(err)
	baseResponse.Code = code
	return *baseResponse
}

func NewBaseResponseFromModelAndError(model interface{}, err error) BaseResponse {
	baseResponse := NewBaseResponse()
	baseResponse.AppendToList(model)
	baseResponse.SetError(err)

	return *baseResponse
}

func NewBaseResponseFromModel(model interface{}) BaseResponse {
	baseResponse := NewBaseResponse()
	baseResponse.AppendToList(model)
	return *baseResponse
}

func NewBaseResponse() *BaseResponse {
	return &BaseResponse{}
}

func (m *BaseResponse) SetError(err error) {
	m.Error = err
	if err != nil {
		m.StrError = err.Error()
	}
}

func (m *BaseResponse) Merge(response BaseResponse) BaseResponse {
	m.Errors = append(m.Errors, response.Errors...)
	if m.Error == nil {
		m.Error = response.Error
	}
	m.TotalRows = utils.Max64(m.TotalRows, response.TotalRows)

	return *m
}

func (m *BaseResponse) GetFirst(model interface{}) error {

	if m.TotalRows == 0 {
		return errors.New("BaseResponse.GetFirst: No rows found")
	}

	if m.List == nil {
		return errors.New("BaseResponse.GetFirst: List is nil")
	}

	list := reflect.ValueOf(m.List)
	if list.Kind() != reflect.Slice {
		return errors.New("BaseResponse.GetFirst: List is not a slice")
	}

	if list.IsNil() {
		return errors.New("BaseResponse.GetFirst: list is nil")
	}

	if list.Len() == 0 {
		return errors.New("BaseResponse.GetFirst: List is empty")
	}

	value := list.Index(0).Interface()

	jsonModel, err := json.Marshal(value)
	json.Unmarshal(jsonModel, model)

	return err

}

func (m *BaseResponse) GetAtIndex(index int, model interface{}) error {

	if m.TotalRows == 0 {
		return errors.New("BaseResponse.GetFirst: No rows found")
	}

	if m.List == nil {
		return errors.New("BaseResponse.GetFirst: List is nil")
	}

	list := reflect.ValueOf(m.List)
	if list.Kind() != reflect.Slice {
		return errors.New("BaseResponse.GetFirst: List is not a slice")
	}

	if list.IsNil() {
		return errors.New("BaseResponse.GetFirst: list is nil")
	}

	if list.Len() == 0 {
		return errors.New("BaseResponse.GetFirst: List is empty")
	}

	if index >= list.Len() {
		return errors.New("BaseResponse.GetFirst: Index out of range")
	}

	value := list.Index(index).Interface()

	jsonModel, err := json.Marshal(value)
	json.Unmarshal(jsonModel, model)

	return err

}

func (m *BaseResponse) GetList(list interface{}) error {

	resultsVal := reflect.ValueOf(list)
	if resultsVal.Kind() != reflect.Ptr {
		return fmt.Errorf("BaseResponse.GetList: argument must be a pointer to a slice, but was a %s", resultsVal.Kind())
	}

	sliceVal := resultsVal.Elem()

	if sliceVal.Kind() == reflect.Interface {
		sliceVal = sliceVal.Elem()
	}

	if sliceVal.Kind() != reflect.Slice {
		return fmt.Errorf("BaseResponse.GetList: argument must be a pointer to a slice, but was a pointer to %s", sliceVal.Kind())
	}

	o, err := json.Marshal(m.List)
	if err != nil {
		log.Err(err)
		return errors.New("BaseResponse.GetList: " + err.Error())
	}
	err = json.Unmarshal(o, &list)
	if err != nil {
		return err
	}

	return nil

}

type List struct {
	Error     error         `json:"error"`
	TotalRows int64         `json:"total_rows"`
	Rows      []interface{} `json:"rows"`
}

func (m *BaseResponse) GetListInterface() List {

	result := &List{}

	rows := make([]interface{}, 0)

	if m.List == nil {
		return *result
	}

	s := reflect.ValueOf(m.List)
	if s.Kind() != reflect.Slice {
		result.Error = errors.New("BaseResponse.GetListInterface: List is not a slice")
		return *result
	}

	if s.IsNil() {
		result.Error = errors.New("BaseResponse.GetListInterface: List is nil")
		return *result
	}

	for i := 0; i < s.Len(); i++ {
		rows = append(rows, s.Index(i).Interface())
		result.TotalRows++
	}

	result.Rows = rows

	return *result

}

func (m *BaseResponse) AppendToList(item interface{}) {
	if m.List == nil {
		m.List = []interface{}{}
	}

	s := reflect.ValueOf(item)

	// Handle pointer to slice
	if s.Kind() == reflect.Ptr {
		s = s.Elem()
	}

	// If it's not a slice, append as single item
	if s.Kind() != reflect.Slice {
		m.List = append(m.List.([]interface{}), item)
		m.TotalRows++
		return
	}

	// Handle empty slice
	if s.Len() == 0 {
		return
	}

	// Convert existing list to Value
	existingList := reflect.ValueOf(m.List)

	// Append all elements from the input slice
	for i := 0; i < s.Len(); i++ {
		existingList = reflect.Append(existingList, s.Index(i))
	}

	m.List = existingList.Interface()
	m.TotalRows = int64(existingList.Len())
}
