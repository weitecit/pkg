package foundation

import (
	"encoding/json"
	"runtime"

	"github.com/weitecit/pkg/log"
)

type Trace struct {
	BaseModel `bson:",inline"`
	Message   string `json:"message" bson:"message"`
	Level     string `json:"level" bson:"level"`
	File      string `json:"file" bson:"file"`
	Line      int    `json:"line" bson:"line"`
}

type TraceList struct {
	BaseResponse
}

func (m *Trace) GetCollection() (name string, isGlobal bool) {
	return "traces", true
}

func LogTrace(model *BaseModel, err error) (*Trace, error) {
	m := &Trace{}
	m.BaseModel = BaseModel{}
	m.Message = err.Error()
	m.Level = "trace"
	_, m.File, m.Line, _ = runtime.Caller(2)
	return m.Update()
}

func LogErr(model *BaseModel, err error) (*Trace, error) {
	m := &Trace{}
	m.Message = err.Error()
	m.Level = "error"
	_, m.File, m.Line, _ = runtime.Caller(2)
	return m.Update()
}

func (m *TraceList) ToList() []*Trace {
	if m.List == nil {
		return []*Trace{}
	}
	list := m.List.([]*Trace)
	return list
}

func (m *Trace) ToJSON() string {
	o, err := json.MarshalIndent(&m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func (m *Trace) Find() *TraceList {
	result := &TraceList{}
	// list := []*Trace{}

	// result.GetResponse(m, m.GetFindOptions(NewBaseRequest()), list)
	return result
}

func (m *TraceList) First() (*Trace, error) {
	// result := &Trace{}
	// if m.Error != nil {
	// 	return result, m.Error
	// }
	// if m.TotalRows < 1 {
	// 	err := errors.New("Trace.First: record not found")
	// 	return result, err
	// }
	// response := m.ToList()
	// return response[0], nil
	println("•••••••••••••••••••••••••••••••••")
	println("TraceList.First: Not implemented")
	println("•••••••••••••••••••••••••••••••••")
	return nil, nil
}

func (m *Trace) Update() (*Trace, error) {
	// err := repo.Update(m)
	// return m, err
	return m, nil
}

func (m *Trace) GetFindOptions(request *BaseRequest) FindOptions {
	// findOptions := m.GetBaseFindOptions(request)

	// return findOptions
	return FindOptions{}
}
