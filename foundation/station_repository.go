package foundation

import (
	"encoding/json"
	"errors"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"
)

type WeatherData struct {
	Year             int     `json:"Año"`
	Week             int     `json:"Semana"`
	AvgTemp          float64 `json:"TempMedia"`
	MaxTemp          float64 `json:"TempMax"`
	DateTimeMaxTemp  string  `json:"DiaHorMinTempMax"`
	MinTemp          float64 `json:"TempMin"`
	DateTimeMinTemp  string  `json:"DiaHorMinTempMin"`
	AvgHumidity      float64 `json:"HumedadMedia"`
	MaxHumidity      float64 `json:"HumedadMax"`
	DateTimeMaxHum   string  `json:"DiaHorMinHumMax"`
	MinHumidity      float64 `json:"HumedadMin"`
	DateTimeMinHum   string  `json:"DiaHorMinHumMin"`
	WindSpeed        float64 `json:"VelViento"`
	WindDirection    float64 `json:"DirViento"`
	MaxWindSpeed     float64 `json:"VelVientoMax"`
	DateTimeMaxSpeed string  `json:"DiaHorMinVelMax"`
	WindDirMaxSpeed  float64 `json:"DirVientoVelMax"`
	Radiation        float64 `json:"Radiacion"`
	Precipitation    float64 `json:"Precipitacion"`
	EtPMon           float64 `json:"EtPMon"`
	PePMon           float64 `json:"PePMon"`
	Station          string  `json:"Estacion"`
}

type WeatherDataResponse struct {
	Datos []WeatherData `json:"Datos"`
}

var StationConnectionPool = []StationRepository{}

type StationType utils.Enum

const (
	StationNone   StationType = ""
	StationSIAR   StationType = "station_siar"
	StationWeitec StationType = "station_weitec"
	StationMockup StationType = "station_mockup"
)

func (m StationType) GetStationType(s string) StationType {
	switch s {
	case "station_weitec":
		return StationWeitec
	case "station_siar":
		return StationSIAR
	case "station_mockup":
		return StationMockup
	default:
		return StationNone
	}
}

type StationRequest struct {
	StationName string
	DateRange   *DateRange
	StationType StationType
}

func (m *StationRequest) ToJSON() string {
	o, err := json.MarshalIndent(&m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

type StationRepoResponse struct {
	Error   error
	Content WeatherData
}

func NewStationRepository(request StationRequest) (StationRepository, error) {

	if request.StationName == "" {
		return nil, errors.New("NewStationRepository: StationName is required")
	}

	if request.DateRange == nil {
		return nil, errors.New("NewStationRepository: DateRange is required")
	}

	if request.StationType == StationNone {
		return nil, errors.New("NewStationRepository: StationType is required")
	}

	switch request.StationType {
	case StationType(StationSIAR):
		station, err := NewStationSIARRepository(request)
		return &station, err
	case StationType(StationWeitec):
		station, err := NewStationWeitecRepository(request)
		return &station, err
	case StationType(StationMockup):
		station, err := NewMockStationSIARRepository(request)
		return &station, err
	default:
		return nil, errors.New("NewStationRepository: StationType is not supported: " + string(request.StationType))
	}
}

type StationRepository interface {
	GetData() StationRepoResponse
}
