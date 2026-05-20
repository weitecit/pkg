package foundation

// MockRepository implements Repository with substitutable function fields and
// built-in call tracking.
//
// Each method records its arguments in a corresponding Calls slice BEFORE
// delegating to the Func field. This means call tracking works even when
// you override the Func — you always get a complete invocation log.
//
// Usage:
//
//	mock := NewMockRepository()
//	mock.FindFunc = func(req RepoRequest) RepoResponse {
//	    return RepoResponse{List: []MyModel{{ID: "1"}}}
//	}
//
//	// after exercising the code under test:
//	if len(mock.FindCalls) != 1 { t.Fatal("expected 1 call to Find") }
//	if mock.FindCalls[0].Request.ID != "abc" { ... }
//
// All Func fields default to returning zero values, so you only need to
// set up the methods your test actually calls.
type MockRepository struct {
	AggregateCalls    []MockRepoCall
	FindCalls         []MockRepoCall
	CountCalls        []MockRepoCall
	FindOneCalls      []MockRepoCall
	UpdateCalls       []MockRepoCall
	UpdateManyCalls   []MockRepoCallWithValues
	UpdateFieldCalls  []MockRepoCallWithFieldValue
	SwitchItemInArrayCalls []MockRepoCallWithFieldString
	AddItemInArrayCalls    []MockRepoCallWithFieldString
	RemoveItemInArrayCalls []MockRepoCallWithFieldString
	MoveCalls         []MockRepoCall
	DeleteCalls       []MockRepoCall
	DeleteSoftCalls   []MockRepoCall
	RemoveFieldCalls  []MockRepoCallWithField
	GetFilterCalls    []MockRepoCallGetFilter
	GetOrderCalls     []MockRepoCallGetOrder
	GetTypeCalls      []MockRepoCallGetType
	GetRepoIDCalls    []MockRepoCallGetRepoID
	GetDataBaseCalls  []MockRepoCallGetDataBase
	GetConnectionCalls []MockRepoCallGetConnection
	SetRepoIDCalls    []MockRepoCallSetRepoID
	RepoBackupCalls   []MockRepoCallWithBackupID
	RepoRestoreCalls  []MockRepoCallWithBackupID
	DeleteDatabaseCalls []MockRepoCallDeleteDatabase

	AggregateFunc    func(request RepoRequest) RepoResponse
	FindFunc         func(request RepoRequest) RepoResponse
	CountFunc        func(request RepoRequest) RepoResponse
	FindOneFunc      func(request RepoRequest) RepoResponse
	UpdateFunc       func(request RepoRequest) RepoResponse
	UpdateManyFunc   func(request RepoRequest, values map[string]interface{}) RepoResponse
	UpdateFieldFunc  func(request RepoRequest, field string, value interface{}) RepoResponse
	SwitchItemInArrayFunc func(request RepoRequest, field string, value string) RepoResponse
	AddItemInArrayFunc    func(request RepoRequest, field string, value string) RepoResponse
	RemoveItemInArrayFunc func(request RepoRequest, field string, value string) RepoResponse
	MoveFunc         func(request RepoRequest) RepoResponse
	DeleteFunc       func(request RepoRequest) RepoResponse
	DeleteSoftFunc   func(request RepoRequest) RepoResponse
	RemoveFieldFunc  func(request RepoRequest, field string) RepoResponse
	GetFilterFunc    func(filterOptions FindOptions) (map[string]interface{}, error)
	GetOrderFunc     func(filterOptions FindOptions) map[string]interface{}
	GetTypeFunc      func() RepoType
	GetRepoIDFunc    func() string
	GetDataBaseFunc  func() string
	GetConnectionFunc func() string
	SetRepoIDFunc    func(value string) error
	RepoBackupFunc   func(request RepoRequest, backupID string) RepoResponse
	RepoRestoreFunc  func(request RepoRequest, backupID string) RepoResponse
	DeleteDatabaseFunc func(connection string, database string) error
}

// compile-time check: *MockRepository implements Repository
var _ Repository = (*MockRepository)(nil)

// NewMockRepository returns a MockRepository with all Func fields set to
// default implementations that return zero values. Override specific fields
// in your test before exercising the code under test.
func NewMockRepository() *MockRepository {
	return &MockRepository{
		AggregateFunc:    defaultRepoFunc,
		FindFunc:         defaultRepoFunc,
		CountFunc:        defaultRepoFunc,
		FindOneFunc:      defaultRepoFunc,
		UpdateFunc:       defaultRepoFunc,
		UpdateManyFunc:   defaultRepoFuncWithValues,
		UpdateFieldFunc:  defaultRepoFuncWithFieldValue,
		SwitchItemInArrayFunc: defaultRepoFuncWithFieldString,
		AddItemInArrayFunc:    defaultRepoFuncWithFieldString,
		RemoveItemInArrayFunc: defaultRepoFuncWithFieldString,
		MoveFunc:         defaultRepoFunc,
		DeleteFunc:       defaultRepoFunc,
		DeleteSoftFunc:   defaultRepoFunc,
		RemoveFieldFunc:  defaultRepoFuncWithField,
		GetFilterFunc:    defaultGetFilterFunc,
		GetOrderFunc:     defaultGetOrderFunc,
		GetTypeFunc:      defaultGetTypeFunc,
		GetRepoIDFunc:    defaultGetRepoIDFunc,
		GetDataBaseFunc:  defaultGetDataBaseFunc,
		GetConnectionFunc: defaultGetConnectionFunc,
		SetRepoIDFunc:    defaultSetRepoIDFunc,
		RepoBackupFunc:   defaultRepoFuncWithBackupID,
		RepoRestoreFunc:  defaultRepoFuncWithBackupID,
		DeleteDatabaseFunc: defaultDeleteDatabaseFunc,
	}
}

// ---------------------------------------------------------------------------
// Repository interface implementation
//
// Each method records the call (args) BEFORE delegating to the Func field.
// This guarantees tracking even with custom Func overrides.
// ---------------------------------------------------------------------------

func (m *MockRepository) Aggregate(request RepoRequest) RepoResponse {
	m.AggregateCalls = append(m.AggregateCalls, MockRepoCall{Request: request})
	return m.AggregateFunc(request)
}

func (m *MockRepository) Find(request RepoRequest) RepoResponse {
	m.FindCalls = append(m.FindCalls, MockRepoCall{Request: request})
	return m.FindFunc(request)
}

func (m *MockRepository) Count(request RepoRequest) RepoResponse {
	m.CountCalls = append(m.CountCalls, MockRepoCall{Request: request})
	return m.CountFunc(request)
}

func (m *MockRepository) FindOne(request RepoRequest) RepoResponse {
	m.FindOneCalls = append(m.FindOneCalls, MockRepoCall{Request: request})
	return m.FindOneFunc(request)
}

func (m *MockRepository) Update(request RepoRequest) RepoResponse {
	m.UpdateCalls = append(m.UpdateCalls, MockRepoCall{Request: request})
	return m.UpdateFunc(request)
}

func (m *MockRepository) UpdateMany(request RepoRequest, values map[string]interface{}) RepoResponse {
	m.UpdateManyCalls = append(m.UpdateManyCalls, MockRepoCallWithValues{Request: request, Values: values})
	return m.UpdateManyFunc(request, values)
}

func (m *MockRepository) UpdateField(request RepoRequest, field string, value interface{}) RepoResponse {
	m.UpdateFieldCalls = append(m.UpdateFieldCalls, MockRepoCallWithFieldValue{Request: request, Field: field, Value: value})
	return m.UpdateFieldFunc(request, field, value)
}

func (m *MockRepository) SwitchItemInArray(request RepoRequest, field string, value string) RepoResponse {
	m.SwitchItemInArrayCalls = append(m.SwitchItemInArrayCalls, MockRepoCallWithFieldString{Request: request, Field: field, Value: value})
	return m.SwitchItemInArrayFunc(request, field, value)
}

func (m *MockRepository) AddItemInArray(request RepoRequest, field string, value string) RepoResponse {
	m.AddItemInArrayCalls = append(m.AddItemInArrayCalls, MockRepoCallWithFieldString{Request: request, Field: field, Value: value})
	return m.AddItemInArrayFunc(request, field, value)
}

func (m *MockRepository) RemoveItemInArray(request RepoRequest, field string, value string) RepoResponse {
	m.RemoveItemInArrayCalls = append(m.RemoveItemInArrayCalls, MockRepoCallWithFieldString{Request: request, Field: field, Value: value})
	return m.RemoveItemInArrayFunc(request, field, value)
}

func (m *MockRepository) Move(request RepoRequest) RepoResponse {
	m.MoveCalls = append(m.MoveCalls, MockRepoCall{Request: request})
	return m.MoveFunc(request)
}

func (m *MockRepository) Delete(request RepoRequest) RepoResponse {
	m.DeleteCalls = append(m.DeleteCalls, MockRepoCall{Request: request})
	return m.DeleteFunc(request)
}

func (m *MockRepository) DeleteSoft(request RepoRequest) RepoResponse {
	m.DeleteSoftCalls = append(m.DeleteSoftCalls, MockRepoCall{Request: request})
	return m.DeleteSoftFunc(request)
}

func (m *MockRepository) RemoveField(request RepoRequest, field string) RepoResponse {
	m.RemoveFieldCalls = append(m.RemoveFieldCalls, MockRepoCallWithField{Request: request, Field: field})
	return m.RemoveFieldFunc(request, field)
}

func (m *MockRepository) GetFilter(filterOptions FindOptions) (map[string]interface{}, error) {
	m.GetFilterCalls = append(m.GetFilterCalls, MockRepoCallGetFilter{FilterOptions: filterOptions})
	return m.GetFilterFunc(filterOptions)
}

func (m *MockRepository) GetOrder(filterOptions FindOptions) map[string]interface{} {
	m.GetOrderCalls = append(m.GetOrderCalls, MockRepoCallGetOrder{FilterOptions: filterOptions})
	return m.GetOrderFunc(filterOptions)
}

func (m *MockRepository) GetType() RepoType {
	m.GetTypeCalls = append(m.GetTypeCalls, MockRepoCallGetType{})
	return m.GetTypeFunc()
}

func (m *MockRepository) GetRepoID() string {
	m.GetRepoIDCalls = append(m.GetRepoIDCalls, MockRepoCallGetRepoID{})
	return m.GetRepoIDFunc()
}

func (m *MockRepository) GetDataBase() string {
	m.GetDataBaseCalls = append(m.GetDataBaseCalls, MockRepoCallGetDataBase{})
	return m.GetDataBaseFunc()
}

func (m *MockRepository) GetConnection() string {
	m.GetConnectionCalls = append(m.GetConnectionCalls, MockRepoCallGetConnection{})
	return m.GetConnectionFunc()
}

func (m *MockRepository) SetRepoID(value string) error {
	m.SetRepoIDCalls = append(m.SetRepoIDCalls, MockRepoCallSetRepoID{Value: value})
	return m.SetRepoIDFunc(value)
}

func (m *MockRepository) RepoBackup(request RepoRequest, backupID string) RepoResponse {
	m.RepoBackupCalls = append(m.RepoBackupCalls, MockRepoCallWithBackupID{Request: request, BackupID: backupID})
	return m.RepoBackupFunc(request, backupID)
}

func (m *MockRepository) RepoRestore(request RepoRequest, backupID string) RepoResponse {
	m.RepoRestoreCalls = append(m.RepoRestoreCalls, MockRepoCallWithBackupID{Request: request, BackupID: backupID})
	return m.RepoRestoreFunc(request, backupID)
}

func (m *MockRepository) DeleteDatabase(connection string, database string) error {
	m.DeleteDatabaseCalls = append(m.DeleteDatabaseCalls, MockRepoCallDeleteDatabase{Connection: connection, Database: database})
	return m.DeleteDatabaseFunc(connection, database)
}

// ---------------------------------------------------------------------------
// Call record types
//
// Each type captures the arguments of one invocation for a specific method
// (or group of methods sharing the same signature).
// ---------------------------------------------------------------------------

// MockRepoCall records a call to any method taking (RepoRequest) RepoResponse.
type MockRepoCall struct {
	Request RepoRequest
}

// MockRepoCallWithValues records a call to UpdateMany.
type MockRepoCallWithValues struct {
	Request RepoRequest
	Values  map[string]interface{}
}

// MockRepoCallWithFieldValue records a call to UpdateField.
type MockRepoCallWithFieldValue struct {
	Request RepoRequest
	Field   string
	Value   interface{}
}

// MockRepoCallWithFieldString records a call to SwitchItemInArray,
// AddItemInArray, or RemoveItemInArray.
type MockRepoCallWithFieldString struct {
	Request RepoRequest
	Field   string
	Value   string
}

// MockRepoCallWithField records a call to RemoveField.
type MockRepoCallWithField struct {
	Request RepoRequest
	Field   string
}

// MockRepoCallWithBackupID records a call to RepoBackup or RepoRestore.
type MockRepoCallWithBackupID struct {
	Request  RepoRequest
	BackupID string
}

// MockRepoCallGetFilter records a call to GetFilter.
type MockRepoCallGetFilter struct {
	FilterOptions FindOptions
}

// MockRepoCallGetOrder records a call to GetOrder.
type MockRepoCallGetOrder struct {
	FilterOptions FindOptions
}

// MockRepoCallGetType records a call to GetType (no args).
type MockRepoCallGetType struct{}

// MockRepoCallGetRepoID records a call to GetRepoID (no args).
type MockRepoCallGetRepoID struct{}

// MockRepoCallGetDataBase records a call to GetDataBase (no args).
type MockRepoCallGetDataBase struct{}

// MockRepoCallGetConnection records a call to GetConnection (no args).
type MockRepoCallGetConnection struct{}

// MockRepoCallSetRepoID records a call to SetRepoID.
type MockRepoCallSetRepoID struct {
	Value string
}

// MockRepoCallDeleteDatabase records a call to DeleteDatabase.
type MockRepoCallDeleteDatabase struct {
	Connection string
	Database   string
}

// ---------------------------------------------------------------------------
// Default implementations — all return zero / safe values
// ---------------------------------------------------------------------------

func defaultRepoFunc(RepoRequest) RepoResponse {
	return RepoResponse{}
}

func defaultRepoFuncWithValues(_ RepoRequest, _ map[string]interface{}) RepoResponse {
	return RepoResponse{}
}

func defaultRepoFuncWithFieldValue(_ RepoRequest, _ string, _ interface{}) RepoResponse {
	return RepoResponse{}
}

func defaultRepoFuncWithFieldString(_ RepoRequest, _ string, _ string) RepoResponse {
	return RepoResponse{}
}

func defaultRepoFuncWithField(_ RepoRequest, _ string) RepoResponse {
	return RepoResponse{}
}

func defaultRepoFuncWithBackupID(_ RepoRequest, _ string) RepoResponse {
	return RepoResponse{}
}

func defaultGetFilterFunc(_ FindOptions) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func defaultGetOrderFunc(_ FindOptions) map[string]interface{} {
	return map[string]interface{}{}
}

func defaultGetTypeFunc() RepoType {
	return RepoTypeUnknown
}

func defaultGetRepoIDFunc() string {
	return ""
}

func defaultGetDataBaseFunc() string {
	return ""
}

func defaultGetConnectionFunc() string {
	return ""
}

func defaultSetRepoIDFunc(_ string) error {
	return nil
}

func defaultDeleteDatabaseFunc(_ string, _ string) error {
	return nil
}
