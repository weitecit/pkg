package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/weitecit/pkg/foundation"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"

	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ServiceRequest struct {
	Errors []error
	ID     *primitive.ObjectID
	IDs    interface{}
	// Puede cambiar el campo ID que por defecto es "_id"
	QueryField  string
	RepoID      string
	Language    foundation.Language
	Location    string
	PageSize    int64
	CurrentPage int64
	List        interface{}
	Date        *time.Time
	Order       *foundation.Orders
	Collation   *options.Collation
	SearchTerms foundation.SearchTerms
	Labels      []string
	Connection  string
	foundation.DateRange
	Repo         foundation.Repository
	RepoModel    foundation.RepositoryModel
	ParseModel   interface{}
	Token        string
	RefreshToken string
	User         foundation.User
	SpaceID      string
}

type ExerciseClusterRequest struct {
	ServiceRequest *ServiceRequest
	Level          int
	AccountCode    string
	EntryNumber    string
	ParentID       *primitive.ObjectID
}

func NewExerciseClusterRequest(request *ServiceRequest, level int) *ExerciseClusterRequest {
	exerciseClusterRequest := &ExerciseClusterRequest{}
	exerciseClusterRequest.ServiceRequest = request
	exerciseClusterRequest.Level = level
	return exerciseClusterRequest
}

func NewServiceRequest() *ServiceRequest {
	serviceRequest := &ServiceRequest{}
	serviceRequest.Order = &foundation.Orders{}
	serviceRequest.SearchTerms = foundation.SearchTerms{}
	return serviceRequest
}

func NewServiceRequestWithSpaceID(id string, repoID string, user foundation.User, language foundation.Language) (*ServiceRequest, error) {
	serviceRequest := NewServiceRequest()

	if id == "" {
		return serviceRequest, errors.New("Services.NewServiceRequestWithSpaceID: ID not found")
	}

	if repoID == "" {
		return serviceRequest, errors.New("Services.NewServiceRequestWithSpaceID: BaseModel RepoID not found")
	}

	serviceRequest.SpaceID = id
	serviceRequest.RepoID = repoID
	serviceRequest.User = user
	serviceRequest.Language = language
	return serviceRequest, nil
}

func NewBaseRequestFromServiceRequestWithIDs(request *ServiceRequest, baseModel *foundation.BaseModel) *foundation.BaseRequest {
	// baseRequest, err := NewBaseRequestFromServiceRequest(request)
	// if err != nil {
	// 	log.Err(err)
	// 	return baseRequest
	// }
	// baseModel.IDs, baseRequest.Error = utils.GetObjectIdsFromInterface(request.IDs)
	// baseModel.QueryField = request.QueryField
	// return baseRequest
	println("•••••••••••••••••••••••••••••••••")
	log.Log("Not implemented")
	fmt.Println("Not implemented")
	println("•••••••••••••••••••••••••••••••••")
	return nil
}

func NewBaseRequestFromServiceRequest(request *ServiceRequest) (*foundation.BaseRequest, error) {

	// baseRequest, err := foundation.NewBaseRequest()
	// if err != nil {
	// 	return baseRequest, err
	// }
	// baseRequest.Order = request.Order
	// baseRequest.CurrentPage = request.CurrentPage
	// baseRequest.PageSize = request.PageSize
	// baseRequest.SearchTerms = request.SearchTerms
	// if utils.HasValidID(request.ID) {
	// 	baseRequest.ID = request.ID
	// }
	// baseRequest.User = request.User
	// return baseRequest, nil
	println("•••••••••••••••••••••••••••••••••")
	log.Log("Not implemented")
	fmt.Println("Not implemented")
	println("•••••••••••••••••••••••••••••••••")
	return nil, nil
}

func NewFoundationBaseRequestWithRepository(request *ServiceRequest) (*foundation.BaseRequest, error) {

	if request.RepoModel != nil {
		if request.RepoModel.GetRepoID() == "" {
			request.RepoModel.SetRepoID(request.RepoID)
		}
		repo, err := foundation.NewRepositoryFromModel(request.RepoModel, request.Connection)
		if err != nil {
			return &foundation.BaseRequest{}, err
		}
		request.Repo = repo
	}

	return NewFoundationBaseRequest(request)
}

func NewFoundationBaseRequest(request *ServiceRequest) (*foundation.BaseRequest, error) {

	if request.RepoModel == nil {
		return &foundation.BaseRequest{}, errors.New("Services.NewFoundationBaseRequest: RepoModel not found")
	}

	repoID := request.RepoModel.GetRepoID()
	if repoID == "" {
		repoID = request.RepoID
		request.RepoModel.SetRepoID(request.RepoID)
	}

	if repoID == "" {
		return &foundation.BaseRequest{}, errors.New("Services.NewFoundationBaseRequest: Repo not found")
	}

	user := request.GetUser()
	if user == nil {
		return &foundation.BaseRequest{}, errors.New("Services.NewFoundationBaseRequest: User not found")
	}

	baseRequest, err := foundation.NewBaseRequest(request.RepoModel, request.Repo, *user)
	if err != nil {
		return baseRequest, err
	}
	baseRequest.Order = request.Order
	baseRequest.Language = request.Language
	baseRequest.CurrentPage = request.CurrentPage
	baseRequest.PageSize = request.PageSize
	baseRequest.Model.SetRepoID(repoID)
	baseRequest.Model.LabelFromStrings(request.Labels...)
	baseRequest.SearchTerms = request.SearchTerms
	if utils.HasValidID(request.ID) {
		baseRequest.ID = request.ID
	}

	baseRequest.IDs = request.IDs
	baseRequest.QueryField = request.QueryField
	baseRequest.SpaceID = request.SpaceID

	return baseRequest, nil

}

func NewServiceRequestFromBaseModel(m *foundation.BaseModel, user *foundation.User) *ServiceRequest {
	request := NewServiceRequest()
	request.RepoID = m.RepoID
	request.Language = m.Language
	if user != nil {
		request.User = *user
	}
	return request
}

func NewServiceRequestFromUser(user *foundation.User) (*ServiceRequest, error) {
	request := NewServiceRequestFromBaseModel(&user.BaseModel, user)
	if user == nil {
		return request, errors.New("Services.NewServiceRequestFromUser: User not found")
	}
	if !utils.HasValidID(user.ID) {
		return request, errors.New("Services.NewServiceRequestFromUser: User ID not found")
	}
	return request, nil

}

func (m *ServiceRequest) AddID(id interface{}) {

	if id == nil {
		return
	}
	switch id.(type) {
	case string:
		if m.IDs == nil {
			m.IDs = []string{}
		}
		m.IDs = append(m.IDs.([]string), id.(string))
	case *primitive.ObjectID:
		if m.IDs == nil {
			m.IDs = []*primitive.ObjectID{}
		}
		m.IDs = append(m.IDs.([]*primitive.ObjectID), id.(*primitive.ObjectID))
	}

}

func (m *ServiceRequest) IsEmpty() bool {
	if m.ID == nil && m.IDs == nil {
		return true
	}
	return false
}

func (m *ServiceRequest) ChangeUser(user *foundation.User) error {
	if user == nil {
		return errors.New("Services.ChangeUser: User not found")
	}
	if !utils.HasValidID(user.ID) {
		return errors.New("Services.ChangeUser: User ID is required")
	}

	m.User = *user
	m.SpaceID = user.SpaceID

	return nil
}

func (m *ServiceRequest) HasValidUser() bool {
	if !utils.HasValidID(m.User.ID) && m.User.Username == "" {
		return false
	}

	return true
}

func (m *ServiceRequest) AddOrderAsc(fields ...string) {

	if m.Order == nil {
		m.Order = &foundation.Orders{}
	}

	for _, field := range fields {
		m.addOrder(field, true)
	}

}

func (m *ServiceRequest) AddOrderDesc(fields ...string) {

	if m.Order == nil {
		m.Order = &foundation.Orders{}
	}

	for _, field := range fields {
		m.addOrder(field, false)
	}

}

func (m *ServiceRequest) HasOrderField(field string) bool {

	if m.Order == nil {
		return false
	}

	for _, order := range *m.Order {
		if order.Field == field {
			return true
		}
	}

	return false

}

func (m *ServiceRequest) addOrder(field string, asc bool) {

	if m.Order == nil {
		m.Order = &foundation.Orders{}
	}

	order := &foundation.Order{}
	order.Field = field
	order.Direction = 1

	if !asc {
		order.Direction = -1
	}
	m.Order.Add(*order)

}

func (m *ServiceRequest) HasUser() bool {

	return utils.HasValidID(m.User.ID)
}

func (m *ServiceRequest) HasBasicInfo() bool {

	if !utils.HasValidID(m.User.ID) {
		return false
	}
	if !m.HasValidUser() {
		return false
	}

	return true
}

func (m *ServiceRequest) ToJSON() string {
	o, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func (m *ServiceRequest) GetUser() *foundation.User {
	if !m.HasValidUser() {
		return nil
	}
	return &m.User
}

type ImportedDataRequest struct {
	ServiceRequest
	ImportedDataID primitive.ObjectID
}

func GetBaseRequest(request *ServiceRequest) (*foundation.BaseRequest, error) {
	// response, err := foundation.NewBaseRequest()
	// if err != nil {
	// 	return response, err
	// }

	// response.CurrentPage = request.CurrentPage
	// response.PageSize = request.PageSize
	// response.Order = request.Order
	// response.SearchTerms = request.SearchTerms
	// response.DateRange = request.DateRange
	// return response, nil

	return nil, nil
}

// func NewBaseResponseFromModelRepoResponse(repoResponse repo.RepoResponse) foundation.BaseResponse {
// 	response := foundation.NewBaseResponse()
// 	response.Error = repoResponse.Error
// 	response.TotalRows = repoResponse.TotalRows
// 	response.TotalPages = repoResponse.TotalPages
// 	response.PageSize = repoResponse.PageSize
// 	response.CurrentPage = repoResponse.CurrentPage

// 	return *response
// }

func GetClaimsFromToken(token string) (jwt.MapClaims, error) {

	result := jwt.MapClaims{}

	mySecret := utils.GetEnv("SECRET_KEY")
	token = strings.Replace(token, "Bearer ", "", 1)

	webToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(mySecret), nil
	})

	if err != nil {
		return result, errors.New("SystemService.GetServiceRequestFromToken: " + err.Error())
	}

	if !webToken.Valid {
		return result, errors.New("SystemService.GetServiceRequestFromToken: invalid webtoken")
	}

	return webToken.Claims.(jwt.MapClaims), err
}
