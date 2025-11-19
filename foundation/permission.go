package foundation

import (
	"errors"

	"github.com/weitecit/pkg/utils"
)

type RolePermission struct {
	PermissionID   string         `json:"permission_id" bson:"permission_id"`
	PermissionType PermissionType `json:"permission_type" bson:"permission_type"`
	Role           SpaceRole      `json:"role" bson:"role"`
}

func (m RolePermission) HasPermission(permissionID string) bool {
	if m.PermissionID == "" {
		return false
	}

	if m.PermissionID != permissionID {
		return false
	}

	if m.Role == SpaceRoleNone {
		return false
	}

	if m.PermissionType == PermissionTypeNone {
		return false
	}
	return true
}

type RolePermissions []RolePermission

func (m RolePermissions) AddPermission(permission RolePermission) RolePermissions {
	// add permission if not exists
	for _, p := range m {
		if p.PermissionID == permission.PermissionID {
			return m
		}
	}

	m = append(m, permission)

	return m
}

func (m RolePermissions) GetPermission(permissionID string) (RolePermission, bool) {

	for _, permission := range m {
		if permission.PermissionID == permissionID {
			return permission, true
		}
	}

	return RolePermission{}, false
}

type SpaceRole utils.Enum

const (
	SpaceRoleNone   SpaceRole = ""
	SpaceRoleNoRole SpaceRole = "no_role"
	SpaceRoleMember SpaceRole = "member"
	SpaceRoleAdmin  SpaceRole = "admin"
	SpaceRoleOwner  SpaceRole = "owner"
	SpaceRoleGuest  SpaceRole = "guest"
)

func (m SpaceRole) ToString() string {
	return string(m)
}

type SpaceMember struct {
	UserID    string    `json:"user_id" bson:"user_id"`
	SpaceRole SpaceRole `json:"space_role" bson:"space_role"`
}

type SpaceMembers []SpaceMember

func (m SpaceMembers) AddMember(user User) (SpaceMembers, bool) {

	if user.ID == nil {
		return m, false
	}
	return m.AddRoleToMembers(user.GetIDStr(), SpaceRoleMember)
}

func (m SpaceMembers) AddRoleToMembers(userID string, role SpaceRole) (SpaceMembers, bool) {

	if userID == "" {
		return m, false
	}

	member := SpaceMember{
		UserID:    userID,
		SpaceRole: role,
	}

	for _, spaceMember := range m {
		if spaceMember.UserID == member.UserID {
			return m, false
		}
	}

	m = append(m, member)
	return m, true

}

func (m SpaceMembers) GetMember(userID string) (SpaceMember, bool) {

	for _, spaceMember := range m {
		if spaceMember.UserID == userID {
			return spaceMember, true
		}
	}
	return SpaceMember{}, false
}

func (m SpaceMembers) RemoveMember(userID string, user User) (SpaceMembers, error) {

	if userID == "" {
		return m, errors.New("SpaceMembers.RemoveMember: userID is empty")
	}

	for i, spaceMember := range m {
		if spaceMember.UserID == userID {
			m = append(m[:i], m[i+1:]...)
			return m, nil
		}
	}

	return m, errors.New("SpaceMembers.RemoveMember: User not found")
}

type PermissionType int

const (
	// Incremental permissions, each level includes the previous, best with iota for comparison
	PermissionTypeNone PermissionType = iota
	PermissionTypeNoAccess
	// read only
	PermissionTypeView
	// comment and sign
	PermissionTypeComment
	// feedback
	PermissionTypeFeedback
	// edit but not empty bin, not recover from bin and not change permissions
	PermissionTypeEdit
	// full access except system blocks
	PermissionTypeFull
)

type PermissionClass utils.Enum

const (
	PermissionClassFirm PermissionClass = "permission_class_firm"
)

// A system block is a block that is not created by a user... PermissionTypeTotal is assigned to system_user
type PermissionCloud struct {
	Permissions []Permission `json:"permissions" bson:"permissions, omitempty"`
}

func NewPermissionCloud(permissions ...Permission) *PermissionCloud {

	return &PermissionCloud{
		Permissions: permissions,
	}
}

func (m PermissionCloud) IsEmpty() bool {
	return len(m.Permissions) == 0
}

// Tags are detailed tokens for know permission information
type Permission struct {
	ID             string         `json:"id" bson:"id"`
	PermissionType PermissionType `json:"permission_type" bson:"permission_type"`
	Value          string         `json:"value" bson:"value"`
	Label          Label          `json:"label" bson:"label"`
}

func (m Permission) HasDomainRestriction() bool {
	labels := []Label{
		LabelField,
		LabelPlot,
	}

	for _, label := range labels {
		if m.Label == label {
			return true
		}
	}

	return false
}

func (m *PermissionCloud) HasDomainRestriction() bool {

	for _, permission := range m.Permissions {
		if permission.HasDomainRestriction() {
			return true
		}
	}

	return false
}

func (m *PermissionCloud) GetPermission(user *User) (Permission, error) {

	if m.Permissions == nil {
		m.Permissions = []Permission{}
		return Permission{}, errors.New("PermissionCloud.GetPermission: Permissions are empty")
	}

	role := user.Role
	roleStr := role.ToString()
	userID := user.GetIDStr()

	for _, permission := range m.Permissions {
		if role == SpaceRoleNoRole {
			if permission.ID == userID {

				return permission, nil
			}
			continue
		}
		if roleStr == permission.Value {
			return permission, nil
		}
	}

	if user.IsStaff() {
		// return owner permission
		for _, permission := range m.Permissions {
			if permission.PermissionType == PermissionTypeFull {
				return permission, nil
			}
		}
	}

	return Permission{}, errors.New("PermissionCloud.GetPermission: Permission not found")
}

func (m *PermissionCloud) AddUser(user User, permissionType PermissionType) error {
	for _, permission := range m.Permissions {
		if permission.ID == user.GetIDStr() {
			return errors.New("PermissionCloud.AddUser: User already exists")
		}
	}

	permission := Permission{
		ID:             user.GetIDStr(),
		PermissionType: permissionType,
		Value:          user.Username,
		Label:          LabelUser,
	}

	m.Permissions = append(m.Permissions, permission)
	return nil
}

func (m *PermissionCloud) AddPermission(permission Permission, user User) error {

	return errors.New("PermissionCloud.AddPermission: Not implemented")

}

func (m *PermissionCloud) HasPermission(token []string, permissionType PermissionType) bool {

	return false
}

func (m *PermissionCloud) HasPermission_v2(user *User, permissionType PermissionType) bool {

	// if !

	// if m.IsEmpty() || token.IsEmpty() {
	// 	return false
	// }

	// if permissionType == PermissionTypeNone {
	// 	return false
	// }

	// for _, permission := range m.Permissions {
	// 	if !utils.ContainsStr(token, permission.ID) {
	// 		continue
	// 	}
	// 	if permission.PermissionType >= permissionType {
	// 		return true
	// 	}
	// }

	return false
}

func (m *PermissionCloud) RemovePermission(id string, user User) error {

	// if !m.HasPermission(user.Token, PermissionTypeFull) {
	// 	return errors.New("PermissionCloud.RemovePermission: User does not have permission to remove permission")
	// }

	// if id == user.GetIDStr() {
	// 	return errors.New("PermissionCloud.RemovePermission: User cannot remove his own permission")
	// }

	// if m.IsEmpty() || !utils.HasValidIDStr(id) {
	// 	return errors.New("PermissionCloud.RemovePermission: PermissionCloud or token is empty")
	// }

	// for i, permission := range m.Permissions {
	// 	if permission.ID != id {
	// 		continue
	// 	}

	// 	m.Permissions = append(m.Permissions[:i], m.Permissions[i+1:]...)
	// 	return nil

	// }

	// return errors.New("PermissionCloud.RemovePermission: PermissionCloud does not have this permission")

	return errors.New("PermissionCloud.RemovePermission: Not implemented")
}
