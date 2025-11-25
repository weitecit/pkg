package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/weitecit/pkg/services"

	"github.com/weitecit/pkg/foundation"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TODO: Migrate To Environment / ConfigFile / DataBase
const (
	DefaultPageSize    = 0
	DefaultCurrentPage = 1
)

type ControllerRequest struct {
	Order       map[string]interface{}
	SearchTerms []string
}
type CheckedAllDirector struct {
	SelectedRows   []*primitive.ObjectID `json:"selectedRows"`
	UnselectedRows []*primitive.ObjectID `json:"unselectedRows"`
	CheckedAll     bool                  `json:"checkedAll"`
}

func (m *ControllerRequest) ToRaw() []byte {
	raw, _ := json.Marshal(m)
	return raw
}

/*func GetExportRequestFromServiceRequest(request services.ServiceRequest) *export.ExportRequest {

	response := export.NewExportRequest()
	response.ID = request.ID
	response.RepoID = request.RepoID
	response.User = foundation.User{}
	return response
}*/

func NewFoundationRequestFromContextAndModel(c *gin.Context, model foundation.RepositoryModel) (*foundation.BaseRequest, *HttpError) {

	result := &foundation.BaseRequest{}

	request, herr := GetBasicRequestAndCheckIntegrity(c, model)
	if herr != nil {
		return result, herr
	}

	if model.GetRepoID() == "" {
		model.SetRepoID(request.RepoID)
	}

	request.RepoModel = model

	result, err := services.NewFoundationBaseRequestWithRepository(request)
	if err != nil {
		return result, NewHttpError(http.StatusInternalServerError, err.Error())
	}

	return result, nil
}

func NewResponseWithHttpError(c *gin.Context, httpError *HttpError) {
	if httpError == nil {
		httpError = &HttpError{Code: 0, Message: ""}
	}
	NewResponseWithError(c, httpError.Code, httpError.Message)
}

func NewResponseWithError(c *gin.Context, code int, err string) {
	if code == 0 {
		code = http.StatusInternalServerError
	}
	NewResponse(c, foundation.BaseResponse{Code: code, Message: err})
}

func NewResponseWithErrorResponse(c *gin.Context, response foundation.BaseResponse) {
	if response.Code == 0 {
		response.Code = http.StatusInternalServerError
	}
	NewResponse(c, response)
}

func NewResponseWithModelAndError(c *gin.Context, model interface{}, error string) {
	response := foundation.BaseResponse{Message: error}
	if model == nil {
		response.Code = http.StatusOK
		response.Message = "ok"
		NewResponse(c, response)
	}
	response.AppendToList(model)
	NewResponse(c, response)
}
func NewResponseWithModel(c *gin.Context, model interface{}) {
	response := foundation.BaseResponse{}
	if model == nil {
		response.Code = http.StatusOK
		response.Message = "ok"
		NewResponse(c, response)
	}
	response.AppendToList(model)
	NewResponse(c, response)
}

func NewResponse(c *gin.Context, response foundation.BaseResponse) {
	if response.Code == 0 {
		response.Code = http.StatusOK
	}

	c.JSON(response.Code, response)
}

func NewResponseWithStr(c *gin.Context, response string) {
	NewResponse(c, foundation.BaseResponse{Code: http.StatusOK, Message: response})
}

func FillBasicUserLoginInfo(user *foundation.User) error {

	if user.RepoID == "" {
		return errors.New("Controllers.FillBasicUserInfo: user do not own to any domain")
	}

	if user.SpaceID == "" {
		return errors.New("Controllers.FillBasicUserInfo: user do not own to any space")
	}

	// domain := &models.Domain{}

	// GetSpace

	// domain.ID = utils.GetObjectIdFromStringRaw(user.RepoID)
	// domain.User = user
	// _, err := domain.FindOne()
	// if err != nil {
	// 	return err
	// }
	// user.Connection = domain.Connection

	// if utils.IsEmptyStrArray(user.Products) {
	// 	if utils.IsEmptyStrArray(domain.Products) {
	// 		return errors.New("no_domain_products")
	// 	}
	// 	user.Products = domain.Products
	// }

	return nil
}

func FillFullRequest(request *services.ServiceRequest) error {

	user := &request.User

	if !user.IsValid() {

		if utils.IsEmptyStr(user.Username) && utils.IsEmptyStr(user.ExternalID) {
			return errors.New("Controller.FillFullRequest: " + "User needs Username or external id")
		}

		baseRequest, err := foundation.NewBaseRequestWithModel(user, *user)
		if err != nil {
			return errors.New("Controller.FillFullRequest: " + err.Error())
		}

		user, err = user.GetOne(baseRequest)
		if err != nil {
			return errors.New("Controller.FillFullRequest: " + err.Error())
		}

		user.Password = ""

		request.SpaceID = user.SpaceID
	}

	if user.Username == "" {
		return errors.New("Controller.FillFullRequest: username is required")
	}

	if !utils.HasValidID(user.ID) {
		return errors.New("Controller.FillFullRequest: UserID is required")
	}

	if request.RepoID == "" {
		return errors.New("Controller.FillFullRequest: RepoID is required")
	}

	return nil
}

type HttpError struct {
	Code    int
	Message string
}

func NewHttpError(code int, message string) *HttpError {
	return &HttpError{
		Code:    code,
		Message: message,
	}
}

func GetBasicRequestAndCheckIntegrity(c *gin.Context, model interface{}) (*services.ServiceRequest, *HttpError) {

	request, err := newBasicServiceRequestFromContext(c, model)
	if err != nil {
		if err.Error() == "Controller.FillFullRequest: User not found" {

			return request, NewHttpError(http.StatusUnauthorized, "Unauthenticated")
		}
		return request, NewHttpError(http.StatusBadRequest, err.Error())
	}

	return request, ManageModels(c, request)
}

func GetBasicStaffRequestAndCheckIntegrity(c *gin.Context, model interface{}) (*services.ServiceRequest, *HttpError) {

	request, herr := GetBasicRequestAndCheckIntegrity(c, model)
	if herr != nil {
		return request, herr
	}

	if !request.User.IsStaff() {
		return request, NewHttpError(401, "Not staff user")
	}

	return request, nil
}

func ManageModels(c *gin.Context, request *services.ServiceRequest) *HttpError {

	if request.ParseModel != nil {
		_ = c.ShouldBindJSON(request.ParseModel) // Ignore parsing errors silently
	}

	if request.RepoModel != nil {
		if request.RepoModel.GetRepoID() == "" {
			request.RepoModel.SetRepoID(request.RepoID)
		}
		repo, err := foundation.NewRepositoryFromModel(request.RepoModel, request.Connection)
		if err != nil {
			return NewHttpError(http.StatusInternalServerError, "Controllers.ManageModels Repository error: "+err.Error())
		}
		request.Repo = repo
	}

	return nil
}

func newServiceRequestFromContext(c *gin.Context, model interface{}) (*services.ServiceRequest, error) {

	request := &services.ServiceRequest{}

	webToken := c.GetHeader("Authorization")

	webToken = strings.Replace(webToken, "Bearer ", "", 1)

	request.Token = webToken
	request, err := services.FillRequestFromToken(request)
	if err != nil {
		err = errors.New("Controller.FillFullRequest: " + err.Error())
		return request, err
	}

	err = FillFullRequest(request)
	if err != nil {
		return request, errors.New("Controller.FillFullRequest: " + err.Error())
	}

	err = FillParams(c, request)
	if err != nil {
		return request, errors.New("Controller.FillFullRequest: " + err.Error())
	}

	request.Language, err = request.Language.Normalize()
	if err != nil {
		return request, errors.New("Controller.FillFullRequest: " + err.Error())
	}

	request.ParseModel = model

	return request, nil
}

func newBasicServiceRequestFromContext(c *gin.Context, model interface{}) (*services.ServiceRequest, error) {

	request, err := newServiceRequestFromContext(c, model)
	if err != nil {
		return request, err
	}

	if !request.HasBasicInfo() {
		err = errors.New("Controller.NewBasicServiceRequestFromContext: no basic info")
		return request, err
	}

	return request, nil
}

func FillParams(c *gin.Context, request *services.ServiceRequest) error {

	requestID := c.Param("id")
	id, err := utils.GetObjectIdFromString(requestID)
	if err != nil {
		if err.Error() != "GetObjectIdFromString: id is empty" {
			return err
		}
	}
	request.ID = id

	requestIDs := c.Query("ids")
	if requestIDs != "" {
		ids := utils.StringToArrayString(requestIDs)
		for _, txdID := range ids {
			id, err := utils.GetObjectIdFromString(txdID)
			if err == nil {
				request.AddID(id)
			}
		}
	}

	pageSize, err := utils.StrToInt64(c.Query("pageSize"))
	if err != nil {
		pageSize = DefaultPageSize
	}
	// TODO: do the same with DELETE?
	method := c.Request.Method

	if method == "POST" {
		pageSize = DefaultPageSize
	}

	request.PageSize = pageSize
	currentPage, err := utils.StrToInt64(c.Query("currentPage"))

	if err != nil {
		currentPage = DefaultCurrentPage
	}
	request.CurrentPage = currentPage

	request.Order = &foundation.Orders{}

	field := c.Query("order")
	orderDirection := c.Query("orderDirection")
	if field != "" && orderDirection != "" {
		if orderDirection == "ascendent" {
			request.AddOrderAsc(field)
		}
		if orderDirection == "descendent" {
			request.AddOrderDesc(field)
		}
	}
	if c.Query("searchTerm") != "" {
		searchTerms := strings.Split(c.Query("searchTerm"), ";")
		request.SearchTerms = searchTerms
	}
	txtLabels := c.Query("labels")
	if !utils.IsEmptyStr(txtLabels) {
		labels := utils.StringToArrayString(txtLabels)
		request.Labels = labels
	}

	if request.SpaceID == "" {
		request.SpaceID = c.Query("spaceid")
	}

	location := c.Param("location")
	request.Location = location

	dateRange := utils.StringToArrayString(c.Query("dateRange"))
	request.DateRange.Set(dateRange)

	return nil
}

func NewFoundationBaseRequestFromServiceRequest(request *services.ServiceRequest) (*foundation.BaseRequest, error) {

	return services.NewFoundationBaseRequest(request)

}

func ParseModel(c *gin.Context, model interface{}) error {
	// err := c.BodyParser(model)
	// if err != nil {
	// 	return err
	// }

	println("•••••••••••••••••••••••••••••••••")
	log.Log("Not implemented")
	fmt.Println("Not implemented")
	println("•••••••••••••••••••••••••••••••••")
	return nil
}

func GetResponseError(c *gin.Context, err error) error {
	// switch err.Error() {

	// case "Repository.GetCollection: the model imported_data needs a database name (DomainID)":
	// 	return c.Status(http.StatusInternalServerError).JSON("not_domain_error")

	// case "Models.GetConnection: DomainID is required":
	// 	return c.Status(http.StatusInternalServerError).JSON("not_domain_error")

	// case "ExerciseController.Get: Exercise.First: record not found":
	// 	return c.Status(http.StatusNoContent).JSON("not_exercise_error")

	// case "JournalLineImport.ImportData: there are more than 50 percent of lines out of range":
	// 	return c.Status(http.StatusInternalServerError).JSON("too_many_out_of_range")
	// }

	// return c.Status(http.StatusInternalServerError).JSON(err.Error())
	println("•••••••••••••••••••••••••••••••••")
	log.Log("Not implemented")
	fmt.Println("Not implemented")
	println("•••••••••••••••••••••••••••••••••")

	return nil
}
