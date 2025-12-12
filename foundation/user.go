package foundation

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ActionLog utils.Enum

type UserLog struct {
	User string    `json:"user" bson:"user"`
	Time time.Time `json:"time" bson:"time"`
}

func GetUserByID(id string) (*User, error) {
	user := &User{}
	user.ID = utils.GetObjectIdFromStringRaw(id)
	request, err := NewBaseRequestWithModel(user, *user)
	if err != nil {
		return nil, err
	}
	user, err = user.GetOne(request)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func GetValueToObjectId(container map[string]interface{}, key string) *primitive.ObjectID {
	value, ok := container[key].(string)
	if !ok {
		return nil
	}
	value = strings.TrimSpace(value)
	objID, err := primitive.ObjectIDFromHex(value)
	if err != nil {
		return nil
	}

	return &objID
}

type (
	User struct {
		// El external ID es el ID de auth0
		BaseModel     `bson:",inline"`
		Nick          string          `json:"nick" bson:"nick"`
		Username      string          `json:"username" bson:"username"`
		ContactID     string          `json:"contact_id" bson:"contact_id"`
		Email         string          `json:"email" bson:"email"`
		Password      string          `json:"password" bson:"password"`
		Licenses      []string        `json:"licenses" bson:"licenses"`
		Connection    string          `json:"connection" bson:"connection"`
		Session       DateRange       `json:"session" bson:"session"`
		ExternalToken string          `json:"external_token" bson:"external_token"`
		Avatar        string          `json:"avatar" bson:"avatar"`
		Roles         RolePermissions `json:"roles" bson:"-"`
		RolePermission
		SpaceID        string `json:"space_id" bson:"-"`
		ChangePassword bool   `json:"change_password" bson:"-"`
	}
)

type Key = []string

func NewUser(userName string) *User {
	m := &User{}
	m.Username = userName
	return m
}

func (m *User) GetCollection() (name string, isGlobal bool) {
	return "users", true
}

func (m *User) ToJSON() string {
	o, err := json.MarshalIndent(&m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func (m *User) ToRaw() []byte {
	raw, _ := json.Marshal(m)
	return raw
}

func (m *User) GetRepoType() RepoType {
	return RepoTypeMongoDB
}

func (m *User) ToArray() []string {
	return []string{m.GetIDStr()}
}

func (m *User) FindOrCreate(request *BaseRequest) BaseResponse {
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
		return BaseResponse{Error: errors.New("User.FindOrCreate: more than one record found")}
	}

	return m.Update(request)
}

func (m *User) Find(request *BaseRequest) BaseResponse {

	findOptions := m.GetFindOptions(request)

	request.SetFindOptions(findOptions)

	request.Model = m
	request.List = []*User{}

	return m.BaseFind(request)
}

func (m *User) GetFindOptions(request *BaseRequest) *FindOptions {
	findOptions := m.GetBaseFindOptions(request)

	if m.Username != "" {
		findOptions.AddEquals("username", m.Username)
	}

	return findOptions
}

func (m *User) FindOne(request *BaseRequest) BaseResponse {

	if m.ID != nil {
		response := m.BaseFindOne(*request)
		if response.Error != nil {
			return response
		}
	}

	response := m.BaseFindOne(*request)
	if response.Error != nil {
		return response
	}

	if response.TotalRows == 0 {
		return NewBaseResponseFromError(errors.New("User.FindOne: no record found"))
	}
	if response.TotalRows > 1 {
		return NewBaseResponseFromError(errors.New("User.FindOne: more than one record found"))
	}

	err := response.GetFirst(m)
	if err != nil {
		return NewBaseResponseFromError(err)
	}

	return response
}

func (m *User) GetOne(request *BaseRequest) (*User, error) {

	if m.ID != nil {
		response := m.FindOne(request)
		if response.Error != nil {
			return m, response.Error
		}
		err := response.GetFirst(m)
		if err != nil {
			return m, err
		}
		return m, nil
	}

	response := m.Find(request)
	if response.Error != nil {
		return m, response.Error
	}

	if response.TotalRows == 0 {
		return m, errors.New("User.GetOne: no results")
	}

	if response.TotalRows > 1 {
		return m, errors.New("User.GetOne: more than one result")
	}

	err := response.GetFirst(m)
	if err != nil {
		return m, err
	}

	return m, nil
}

func (m *User) Update(request *BaseRequest) BaseResponse {

	err := m.Validate()
	if err != nil {
		return NewBaseResponseFromError(err)
	}

	if m.IsNew() || m.ChangePassword {
		err := m.EncryptPassword()
		if err != nil {
			return NewBaseResponseFromError(err)
		}
		m.Language = "es-ES"
	}

	userNameChanged := false

	if !m.IsNew() {
		user := &User{}
		user.ID = m.ID
		userRequest, err := request.Clone(user)
		if err != nil {
			return NewBaseResponseFromError(err)
		}

		user, err = user.GetOne(userRequest)
		if err != nil {
			return NewBaseResponseFromError(err)
		}

		if user.Username != m.Username {
			userNameChanged = true
		}

	}

	if m.IsNew() || userNameChanged {
		user := &User{}
		user.Username = m.Username
		userRequest, err := request.Clone(user)
		if err != nil {
			return NewBaseResponseFromError(err)
		}
		_, err = user.GetOne(userRequest)
		if err == nil {
			return NewBaseResponseFromError(errors.New("User.Update: user already exists"))
		}

		if err.Error() != "User.GetOne: no results" {
			return NewBaseResponseFromError(err)
		}
	}

	return m.UpdateRaw(request)
}

func (m *User) EncryptPassword() error {

	if m.Password == "" {
		return errors.New("User.EncryptPassword: Password required")
	}

	hash, err := utils.GenerateHash(m.Password)
	if err != nil {
		return err
	}
	m.Password = string(hash)
	return nil
}

func (m *User) UpdateRaw(request *BaseRequest) BaseResponse {

	err := m.Validate()
	if err != nil {
		return NewBaseResponseFromError(err)
	}

	return m.BaseUpdate(*request)
}

func (m *User) Validate() error {

	if m.Username == "" {
		return errors.New("User.Validate: UserName required")
	}

	if m.Password == "" {
		return errors.New("User.Validate: Password required")
	}

	if strings.Contains(m.Username, " ") == true || strings.Contains(m.Password, " ") == true {
		return errors.New("User.Validate: UserName nor Password cannot contain spaces")
	}

	// Tendría que validar que no exista un usuario con ese nombre ya en el repositorio
	return nil
}

func (m *User) Delete(request *BaseRequest) BaseResponse {

	request.SetFindOptions(m.GetFindOptions(request))
	request.Model = m

	if m.DeletedBy != nil {
		return m.BaseDelete(*request)
	}

	return m.BaseDeleteSoft(*request)

}

func (m *User) GetUserLog() *UserLog {

	if m.ID == nil {
		return &UserLog{
			Time: time.Now(),
		}
	}

	return &UserLog{
		User: m.GetIDStr(),
		Time: time.Now(),
	}
}

func (m *User) GetSystemUser() (*User, error) {
	adminUser := &User{}
	adminUser.Username = utils.GetEnv("SYSTEM_USER")
	token := utils.GetEnv("SYSTEM_TOKEN")

	id, err := utils.GetObjectIdFromString(token)
	if err != nil {
		return adminUser, errors.New("User.GetSystemUser: System Token invalid: " + err.Error())
	}
	adminUser.ID = id
	adminUser.Label(LabelSystem)

	adminUser.Language = "en-ES"

	return adminUser, nil
}

func (m *User) GetNotificationUsersIDs() []string {
	strIDs := utils.GetEnv("NOTIFICATION_USERS")
	return utils.SplitLevelArray(strIDs)
}

func (m *User) CheckPassword(request *BaseRequest) (*User, error) {

	user := &User{}
	user.Username = m.Username
	user.RepoID = m.RepoID
	request.Model = user

	user, err := user.GetOne(request)
	if err != nil {
		return m, err
	}

	match := utils.CompareStringAndHash(m.Password, user.Password)
	if !match {
		// Not clues
		return m, errors.New("User.CheckPassword: wrong user or password")
	}

	m = user

	return user, nil

}

func (m *User) OpenSession(request *BaseRequest) error {
	user := &User{}
	user.ID = m.ID
	user.RepoID = m.RepoID
	user, err := user.GetOne(request)
	if err != nil {
		return err
	}
	user.Session = DateRange{
		StartDate: utils.Now(),
	}
	response := user.UpdateRaw(request)

	return response.Error
}

func (m *User) CloseSession(request *BaseRequest) BaseResponse {
	user := &User{}
	user.ID = m.ID
	user.RepoID = m.RepoID
	user, err := user.GetOne(request)
	if err != nil {
		return NewBaseResponseFromError(err)
	}
	if user.Session.StartDate == nil {
		user.Session.StartDate = utils.Now()
	}
	user.Session.EndDate = utils.Now()
	return user.UpdateRaw(request)
}

func (m *User) IsStaff() bool {
	return m.HasLabel(LabelStaff)
}

func (m *User) IsAdmin() bool {
	return m.HasLabel(LabelAdmin)
}

func (m *User) IsSystem() bool {
	return m.HasLabel(LabelSystem)
}

func (m *User) IsValid() bool {
	// check if user has name, if not check id if id get one

	if m.Username != "" {
		return true
	}

	if !utils.HasValidID(m.ID) {
		return false
	}

	if m.RepoID == "" {
		return false
	}

	user := &User{}
	user.ID = m.ID
	user.RepoID = m.RepoID
	request, err := NewBaseRequestWithModel(user, *user)
	if err != nil {
		return false
	}

	user, err = user.GetOne(request)
	if err != nil {
		return false
	}

	if user.Username == "" {
		return false
	}

	m = user
	return true
}

func (m *User) GetFromMap(token map[string]interface{}) error {

	m.ID = utils.GetValueToObjectId(token, "UserID")
	m.Username = utils.GetValueToStr(token, "Username")
	m.RepoID = utils.GetValueToStr(token, "DomainID")
	m.ContactID = utils.GetValueToStr(token, "ContactID")
	m.SpaceID = utils.GetValueToStr(token, "SpaceID")
	m.Nick = utils.GetValueToStr(token, "Nick")

	languageStr := utils.GetValueToStr(token, "Language")
	userLanguageStr := utils.GetValueToStr(token, "UserLanguage")
	if userLanguageStr == "" {
		userLanguageStr = languageStr
	}
	if languageStr == "" {
		languageStr = userLanguageStr
	}
	userLanguage, _ := NewLanguage(userLanguageStr)
	m.Language = userLanguage

	userRoles, ok := token["Roles"]

	blockPermissions := RolePermissions{}

	if ok && userRoles != nil {
		// transform the userRoles to a []BlockPermission
		for _, userRole := range userRoles.([]interface{}) {
			permission := userRole.(map[string]interface{})
			blockPermission := RolePermission{
				PermissionID:   permission["PermissionID"].(string),
				PermissionType: PermissionType(permission["PermissionType"].(float64)),
				Role:           SpaceRole(permission["Role"].(string)),
			}
			blockPermissions = append(blockPermissions, blockPermission)
		}

	}
	m.Roles = blockPermissions

	labels := utils.GetValueToArrayStr(token, "UserLabels")
	m.LabelFromStrings(labels...)

	if !m.IsValid() {
		return errors.New("user is not valid")
	}

	return nil
}
