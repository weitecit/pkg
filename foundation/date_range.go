package foundation

import (
	"errors"
	"math"
	"time"

	"github.com/weitecit/pkg/utils"
)

type DateRange struct {
	StartDate *time.Time `json:"start_date,omitempty" bson:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty" bson:"end_date,omitempty"`
}

func NewDateRange(startDate *time.Time, endDate *time.Time) (*DateRange, error) {
	if endDate == nil && startDate == nil {
		return &DateRange{}, nil
	}

	if startDate == nil {
		startDate = utils.Now()
	}

	if endDate == nil {
		endDate = startDate
	}

	if !utils.DateIsValid(startDate) {
		return &DateRange{}, errors.New("NewDateRange: invalid start date")
	}

	if !utils.DateIsValid(endDate) {
		return &DateRange{}, errors.New("NewDateRange: invalid end date")
	}

	if endDate != nil && startDate != nil && endDate.Before(*startDate) {
		return &DateRange{}, errors.New("NewDateRange: endDate cannot be before startDate")
	}

	return &DateRange{
		StartDate: startDate,
		EndDate:   endDate,
	}, nil
}

func NewDateRangeFromStartDate(startDate *time.Time) (*DateRange, error) {
	return NewDateRange(startDate, startDate)
}

func NewDateRangeWeek(startDate *time.Time) (*DateRange, error) {
	week := startDate.AddDate(0, 0, 6)
	return NewDateRange(startDate, &week)
}

func NewDateRangeFromStr(startDate string, endDate string) (DateRange, error) {

	if startDate == "" || endDate == "" {
		return DateRange{}, errors.New("NewDateRangeFromStr error: startDate or endDate is empty")
	}

	startDateParsed := utils.StrMillisecondsToDate(startDate)
	endDateParsed := utils.StrMillisecondsToDate(endDate)

	result, err := NewDateRange(startDateParsed, endDateParsed)

	if err != nil {
		return DateRange{}, err
	}

	return *result, nil

}

func NewDataRangeFromRange(r *DateRange, add time.Duration) *DateRange {

	if r.StartDate == nil || r.EndDate == nil {
		return nil
	}

	startDate := r.StartDate.Add(add)
	endDate := r.EndDate.Add(add)

	return &DateRange{
		StartDate: &startDate,
		EndDate:   &endDate,
	}
}

func NewDateRangeFullWeek(startDate *time.Time) *DateRange {

	// Get the current weekday (0 = Sunday, 1 = Monday, ..., 6 = Saturday)
	weekday := startDate.Weekday()

	// Calculate the number of days to subtract to get to Monday
	daysToMonday := int(weekday) - 1
	if daysToMonday == -1 {
		daysToMonday = 6 // If today is Sunday, we need to go back 6 days
	}

	// Calculate Monday (start of the week)
	monday := startDate.AddDate(0, 0, -daysToMonday)

	// Set time to midnight and remove timezone
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)

	// Calculate Sunday (end of the week)
	sunday := monday.AddDate(0, 0, 6)

	return &DateRange{
		StartDate: &monday,
		EndDate:   &sunday,
	}
}

func NewDateRangeNextWeek(startDate *time.Time) (*DateRange, error) {
	week := startDate.AddDate(0, 0, 6)
	return NewDateRangeWeek(&week)
}

func NewDateRangeFromWeekNumber(weekNumber int) (*DateRange, error) {

	if weekNumber < 1 || weekNumber > 53 {
		return nil, errors.New("el número de semana debe estar entre 1 y 53")
	}

	// Año actual
	now := time.Now()
	year := now.Year()

	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)

	// Retroceder hasta el lunes de esa semana
	for jan4.Weekday() != time.Monday {
		jan4 = jan4.AddDate(0, 0, -1)
	}

	startDate := jan4.AddDate(0, 0, (weekNumber-1)*7)
	endDate := startDate.AddDate(0, 0, 6)

	return &DateRange{
		StartDate: &startDate,
		EndDate:   &endDate,
	}, nil
}

func NewDataRangeThisWeek() (*DateRange, error) {
	now := time.Now()

	return NewDateRangeWeek(&now)
}

func NewDateRangeFullDay(startDate *time.Time) DateRange {
	result := &DateRange{}
	return result.SetOneFullDay(startDate)
}

func (m DateRange) GetWeek() int {
	if m.StartDate == nil {
		return 0
	}
	return utils.WeekNumber(m.StartDate)
}

func (m *DateRange) GetDateRangeInWeek() *DateRange {
	// get monday
	monday := m.StartDate.AddDate(0, 0, -int(m.StartDate.Weekday()))

	// get monday plus 6 days
	sunday := monday.AddDate(0, 0, 6)

	return &DateRange{
		StartDate: &monday,
		EndDate:   &sunday,
	}
}

func (m DateRange) Add(time time.Duration) *DateRange {
	m = *NewDataRangeFromRange(&m, time)
	return &m
}

func (m DateRange) SetOneFullDay(day *time.Time) DateRange {
	if day == nil {
		return m
	}

	startDate := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	m.StartDate = &startDate
	endDate := m.StartDate.Add(24 * time.Hour)
	m.EndDate = &endDate

	return m
}

func (m DateRange) ExtractSuccessiveMonthDates() []time.Time {
	startDate := *m.StartDate
	endDate := *m.EndDate

	dates := []time.Time{}

	for endDate.After(startDate) {
		dates = append(dates, startDate)
		startDate = startDate.AddDate(0, 1, 0)
	}

	return dates
}

func (m DateRange) SetEnd() DateRange {
	endDate := time.Now()
	m.EndDate = &endDate

	return m
}

func (m DateRange) Difference() time.Duration {
	if m.StartDate == nil || m.EndDate == nil {
		return 0
	}
	return m.EndDate.Sub(*m.StartDate)
}

func (m *DateRange) RecalcDateLimits(dateToInclude *time.Time) {
	if !utils.DateIsValid(dateToInclude) {
		return
	}
	if m.StartDate == nil {
		m.StartDate = dateToInclude
		m.EndDate = dateToInclude
		return
	}
	if dateToInclude.Before(*m.StartDate) {
		m.StartDate = dateToInclude
	}
	if dateToInclude.After(*m.EndDate) {
		m.EndDate = dateToInclude
	}
}

func InTimeSpan(start, end, checkTime time.Time) bool {
	if checkTime.Before(start) {
		return false
	}
	if checkTime.After(end) {
		return false
	}
	return true
}

func (m *DateRange) TimeIn(checkTime time.Time) bool {
	if checkTime.Before(*m.StartDate) {
		return false
	}
	if checkTime.After(*m.EndDate) {
		return false
	}
	return true
}

func (m *DateRange) RangeIn(checkDateRange DateRange) bool {
	if checkDateRange.StartDate.Before(*m.StartDate) {
		return false
	}
	if checkDateRange.EndDate.After(*m.EndDate) {
		return false
	}
	return true
}

func (m *DateRange) Overlaps(dateRange DateRange) bool {

	if m == nil {
		return false
	}

	if m.StartDate == nil {
		return false
	}

	if m.EndDate == nil {
		return false
	}

	if m.EndDate.Before(*dateRange.StartDate) || m.StartDate.After(*dateRange.EndDate) {
		return false
	}
	return true
}

func (m DateRange) IsValid() bool {
	return m.StartDate != nil && m.EndDate != nil
}

func (m DateRange) IsEqual(dateRange DateRange) bool {
	if m.StartDate.String() != dateRange.StartDate.String() {
		return false
	}
	if m.EndDate.String() != dateRange.EndDate.String() {
		return false
	}
	return true
}

func (m DateRange) IsEqualOmitTime(dateRange DateRange) bool {

	if m.StartDate.Year() != dateRange.StartDate.Year() ||
		m.StartDate.Month() != dateRange.StartDate.Month() ||
		m.StartDate.Day() != dateRange.StartDate.Day() {
		return false
	}
	if m.EndDate.Year() != dateRange.EndDate.Year() ||
		m.EndDate.Month() != dateRange.EndDate.Month() ||
		m.EndDate.Day() != dateRange.EndDate.Day() {
		return false
	}
	return true
}

func (m DateRange) GetPreviousYearDateRange() DateRange {

	startDate := m.StartDate.AddDate(-1, 0, 0)
	endDate := m.EndDate.AddDate(-1, 0, 0)

	result := DateRange{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	return result
}

func (m DateRange) GetFutureYearDateRange() DateRange {

	startDate := m.StartDate.AddDate(1, 0, 0)
	endDate := m.EndDate.AddDate(1, 0, 0)

	result := DateRange{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	return result
}

func (m DateRange) GetPreviousDateRange() DateRange {

	difference := m.EndDate.Sub(*m.StartDate)

	differenceMonths := int(math.Round(difference.Hours() / 24 / 30))

	result := m.AddMonths(-differenceMonths)

	return result
}

func (m DateRange) GetFutureDateRange() DateRange {

	difference := m.EndDate.Sub(*m.StartDate)

	differenceMonths := int(math.Round(difference.Hours() / 24 / 30))

	result := m.AddMonths(differenceMonths)

	return result
}

func (m DateRange) AdjustHours() DateRange {

	startDate := time.Date(m.StartDate.Year(), m.StartDate.Month(), m.StartDate.Day(), 0, 0, 0, 0, time.UTC)
	endDate := time.Date(m.EndDate.Year(), m.EndDate.Month(), m.EndDate.Day(), 23, 59, 59, 0, time.UTC)

	m.StartDate = &startDate
	m.EndDate = &endDate

	return m
}

func (m DateRange) AddMonths(months int) DateRange {
	if months > 0 {
		startDate := m.EndDate.AddDate(0, 0, 1)
		m.StartDate = &startDate
		m.EndDate = addMonthsToDate(m.EndDate, months)
		return m
	}
	if months < 0 {
		endDate := m.StartDate.AddDate(0, 0, -1)
		m.EndDate = &endDate
		m.StartDate = addMonthsToDate(m.StartDate, months)
		return m
	}
	return m
}

func addMonthsToDate(t *time.Time, months int) *time.Time {
	year := t.Year()
	month := t.Month()
	day := t.Day()

	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	isLastDayOfMonth := firstOfMonth.AddDate(0, 1, -1).Day() == day

	if months > 0 {
		for i := months; i > 0; i-- {
			month++
			if month > 12 {
				month = 1
				year++
			}
		}
	} else {
		for i := months; i < 0; i++ {
			month--
			if month < 1 {
				month = 12
				year--
			}
		}
	}

	if isLastDayOfMonth {
		firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		day = firstOfMonth.AddDate(0, 1, -1).Day()
	}

	date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

	t = &date
	return t
}

func (m DateRange) GetStartYear() int {
	if m.StartDate == nil {
		return 0
	}
	return m.StartDate.Year()
}

func (m DateRange) GetEndYear() int {
	if m.EndDate == nil {
		return 0
	}
	return m.EndDate.Year()
}

func (m *DateRange) Set(date []string) {
	if len(date) < 2 {
		return
	}

	m.StartDate = utils.StrMillisecondsToDate(date[0])
	m.EndDate = utils.StrMillisecondsToDate(date[1])
}

func (m *DateRange) IsNextDateRange(dateRange DateRange) bool {
	if m.EndDate == nil || dateRange.StartDate == nil {
		return false
	}

	return m.EndDate.AddDate(0, 0, 1).Equal(*dateRange.StartDate)
}

func (m *DateRange) IsPreviousDateRange(dateRange DateRange) bool {
	if m.StartDate == nil || dateRange.EndDate == nil {
		return false
	}

	return m.StartDate.AddDate(0, 0, -1).Equal(*dateRange.EndDate)
}
