package foundation

import (
	"strings"

	"github.com/weitecit/pkg/utils"
)

type Label utils.Enum

const (
	LabelNone          Label = ""
	LabelClient        Label = "client"
	LabelEnterprise    Label = "enterprise"
	LabelContactPerson Label = "contact_person"
	LabelField         Label = "field"
	LabelPlot          Label = "plot"
	LabelDomain        Label = "domain"
	LabelUser          Label = "user"
	LabelSystem        Label = "system"
	LabelMember        Label = "member"
	LabelHidden        Label = "hidden"
	LabelHeritage      Label = "heritage"
	LabelGuest         Label = "guest"
	LabelStaff         Label = "staff"
	LabelAdmin         Label = "admin"
	LabelProcesable    Label = "procesable"
	LabelTemplate      Label = "template"
	LabelSpaceRole     Label = "space_role"
	LabelGlobal        Label = "global"
	LabelQueued        Label = "queued"

	LabelEpicTask  Label = "epic_task"
	LabelCompleted Label = "completed"
	LabelInactive  Label = "inactive"

	LabelLink Label = "link"

	LabelDraft Label = "draft"
	LabelError Label = "error"

	LabelSupport     Label = "support"
	LabelNoData      Label = "no_data"
	LabelNeedRefresh Label = "need_refresh"
)

type Labels []Label

func (m Labels) ToString() []string {

	var result []string

	for _, label := range m {
		result = append(result, label.ToString())
	}

	return result

}

func (m Label) IsError() bool {
	return strings.Contains(string(m), "error_")
}

func (m Label) IsWarning() bool {
	return strings.Contains(string(m), "warning_")
}

func (m Label) ToString() string {
	return string(m)
}

func NewLabelFromString(text string) Label {

	switch text {
	case "staff":
		return LabelStaff
	case "admin":
		return LabelAdmin
	case "task_epic":
		return LabelEpicTask
	case "completed":
		return LabelCompleted
	default:
		return LabelNone
	}
}

func (m *Labels) Add(labels ...Label) {

	if m == nil {
		m = &Labels{}
	}

	for _, label := range labels {
		if label == LabelNone {
			continue
		}
		if !m.Has(label) {
			*m = append(*m, label)
		}
	}
}

func (m *Labels) Has(label Label) bool {

	if m == nil {
		m = &Labels{}
	}

	for _, item := range *m {
		if item == label {
			return true
		}
	}
	return false
}

func (m *Labels) Remove(labels ...Label) bool {

	if m == nil {
		m = &Labels{}
	}

	labeled := false

	for _, label := range labels {
		for i, item := range *m {
			if item == label {
				*m = append((*m)[:i], (*m)[i+1:]...)
				labeled = true
				break
			}
		}
	}

	return labeled
}

func (m *Labels) ToArray() []Label {
	return *m
}

func (m Labels) Total() int {
	return len(m)
}
