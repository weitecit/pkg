package foundation

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"
)

/*
	{
	           "Año": 2024,
	           "Semana": 39,
	           "TempMedia": 22.396,
	           "TempMax": 33.710,
	           "DiaHorMinTempMax": "2024-09-26T14:00:00",
	           "TempMin": 14.520,
	           "DiaHorMinTempMin": "2024-09-29T03:30:00",
	           "HumedadMedia": 57.651,
	           "HumedadMax": 95.400,
	           "DiaHorMinHumMax": "2024-09-24T04:40:00",
	           "HumedadMin": 27.410,
	           "DiaHorMinHumMin": "2024-09-25T13:40:00",
	           "VelViento": 1.099,
	           "DirViento": 314.483,
	           "VelVientoMax": 6.664,
	           "DiaHorMinVelMax": null,
	           "DirVientoVelMax": null,
	           "Radiacion": 16.313,
	           "Precipitacion": 0.000,
	           "EtPMon": 24.128014999999998,
	           "PePMon": 0.0,
	           "Estacion": "A21"
	       }
*/

type StationSiarRepository struct {
	APIKey      string
	BaseURL     string
	StationName string
	DateRange   *DateRange
}

func (m *StationSiarRepository) ToJSON() string {
	o, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func NewStationSIARRepository(request StationRequest) (StationSiarRepository, error) {
	repo := StationSiarRepository{
		APIKey:      utils.GetEnv("STATION_SIA_APIKEY"),
		BaseURL:     utils.GetEnv("STATION_SIA_URL"),
		StationName: request.StationName,
		DateRange:   request.DateRange,
	}

	return repo, nil
}

func (m *StationSiarRepository) GetData() StationRepoResponse {

	u, err := url.Parse(m.BaseURL)
	if err != nil {
		return StationRepoResponse{Error: fmt.Errorf("error parsing URL: %v", err)}
	}

	if m.DateRange == nil {
		return StationRepoResponse{Error: fmt.Errorf("DateRange is required")}
	}

	start_date := m.DateRange.StartDate
	end_date := m.DateRange.EndDate

	q := u.Query()
	q.Set("id", m.StationName)
	q.Set("FechaInicial", start_date.Format("2006-01-02"))
	q.Set("FechaFinal", end_date.Format("2006-01-02"))
	q.Set("ClaveAPI", m.APIKey)
	u.RawQuery = q.Encode()

	// Create a new request
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return StationRepoResponse{Error: fmt.Errorf("error creating request: %v", err)}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return StationRepoResponse{Error: fmt.Errorf("error sending request: %v", err)}
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return StationRepoResponse{Error: fmt.Errorf("http error status: %v", resp.Status)}
	}
	// Read and parse the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return StationRepoResponse{Error: fmt.Errorf("error reading response body: %v", err)}
	}

	var data WeatherDataResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return StationRepoResponse{Error: fmt.Errorf("error unmarshalling response body: %v", err)}
	}

	// if data == (WeatherData{}) {
	// 	return StationRepoResponse{Error: fmt.Errorf("data is empty: %v", err)}
	// }

	if len(data.Datos) == 0 {
		return StationRepoResponse{Error: fmt.Errorf("StationSiarRepository.GetData: no data available")}
	}

	return StationRepoResponse{Content: data.Datos[0]}

}
