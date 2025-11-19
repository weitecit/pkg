package foundation

import (
	"encoding/json"

	"github.com/weitecit/pkg/log"
)

type MockStationSiarRepository struct {
	APIKey      string
	BaseURL     string
	StationName string
	DateRange   *DateRange
}

func (m *MockStationSiarRepository) ToJSON() string {
	o, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func NewMockStationSIARRepository(request StationRequest) (MockStationSiarRepository, error) {
	repo := MockStationSiarRepository{
		APIKey:      "_1dFj0issvt8bd_Kz2TNQ9mLKdbwG55xzjJpHsoA_zHhYvOYi_",
		BaseURL:     "https://servicio.mapama.gob.es/apisiar/API/v1/Datos/Semanales/Estacion",
		StationName: request.StationName,
		DateRange:   request.DateRange,
	}

	return repo, nil
}

func (m *MockStationSiarRepository) GetData() StationRepoResponse {

	var data = WeatherDataResponse{
		Datos: []WeatherData{
			{
				Year:             2024,
				Week:             39,
				AvgTemp:          22.396,
				MaxTemp:          33.710,
				DateTimeMaxTemp:  "2024-09-26T14:00:00",
				MinTemp:          14.520,
				DateTimeMinTemp:  "2024-09-29T03:30:00",
				AvgHumidity:      57.651,
				MaxHumidity:      95.400,
				DateTimeMaxHum:   "2024-09-24T04:40:00",
				MinHumidity:      27.410,
				DateTimeMinHum:   "2024-09-25T13:40:00",
				WindSpeed:        1.099,
				WindDirection:    314.483,
				MaxWindSpeed:     6.664,
				DateTimeMaxSpeed: "",
				WindDirMaxSpeed:  0,
				Radiation:        16.313,
				Precipitation:    0.000,
				EtPMon:           24.128,
				PePMon:           0.0,
				Station:          "A21",
			},
		},
	}

	return StationRepoResponse{Content: data.Datos[0]}

}
