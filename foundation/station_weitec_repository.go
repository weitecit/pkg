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

type StationWeitecRepository struct {
	APIKey      string
	BaseURL     string
	StationName string
	DateRange   *DateRange
}

type WeatherMetric struct {
	TempHigh      float64 `json:"tempHigh"`
	TempLow       float64 `json:"tempLow"`
	TempAvg       float64 `json:"tempAvg"`
	WindspeedHigh float64 `json:"windspeedHigh"`
	WindspeedAvg  float64 `json:"windspeedAvg"`
	HumidityHigh  int     `json:"humidityHigh"`
	HumidityLow   int     `json:"humidityLow"`
	HumidityAvg   int     `json:"humidityAvg"`
}

type WeatherObservation struct {
	StationID    string        `json:"stationID"`
	ObsTimeLocal string        `json:"obsTimeLocal"`
	WinddirAvg   float64       `json:"winddirAvg"`
	Metric       WeatherMetric `json:"metric"`
}

type WeatherResponse struct {
	Observations []WeatherObservation `json:"observations"`
}

func (m *StationWeitecRepository) ToJSON() string {
	o, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func NewStationWeitecRepository(request StationRequest) (StationWeitecRepository, error) {
	repo := StationWeitecRepository{
		APIKey:      utils.GetEnv("STATION_WEITEC_APIKEY"),
		BaseURL:     utils.GetEnv("STATION_WEITEC_URL"),
		StationName: request.StationName,
		DateRange:   request.DateRange,
	}

	return repo, nil
}

func (m *StationWeitecRepository) GetData() StationRepoResponse {

	u, err := url.Parse(m.BaseURL)
	if err != nil {
		return StationRepoResponse{Error: fmt.Errorf("error parsing URL: %v", err)}
	}
	println("•••••••••••••••••••••••••••••••••")
	println("m.DateRange.EndDate", m.DateRange.EndDate.Format("20060102"))
	println("•••••••••••••••••••••••••••••••••")
	q := u.Query()
	q.Set("stationId", m.StationName)
	q.Set("format", "json")
	q.Set("units", "m")
	q.Set("date", m.DateRange.EndDate.Format("20060102"))
	q.Set("apiKey", m.APIKey)
	u.RawQuery = q.Encode()

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return StationRepoResponse{Error: fmt.Errorf("error reading response body: %v", err)}
	}

	var weatherResp WeatherResponse
	err = json.Unmarshal(body, &weatherResp)
	if err != nil {
		return StationRepoResponse{Error: fmt.Errorf("error unmarshalling response body: %v", err)}
	}

	if len(weatherResp.Observations) == 0 {
		return StationRepoResponse{Error: fmt.Errorf("no data available")}
	}

	obs := weatherResp.Observations[0]

	// Convert to our WeatherData format
	weatherData := WeatherData{
		Year:          m.DateRange.EndDate.Year(),
		Week:          utils.WeekNumber(m.DateRange.EndDate),
		AvgTemp:       obs.Metric.TempAvg,
		MaxTemp:       obs.Metric.TempHigh,
		MinTemp:       obs.Metric.TempLow,
		AvgHumidity:   float64(obs.Metric.HumidityAvg),
		MaxHumidity:   float64(obs.Metric.HumidityHigh),
		MinHumidity:   float64(obs.Metric.HumidityLow),
		WindSpeed:     obs.Metric.WindspeedAvg,
		WindDirection: obs.WinddirAvg,
		MaxWindSpeed:  obs.Metric.WindspeedHigh,
		Station:       obs.StationID,
	}

	response := WeatherDataResponse{
		Datos: []WeatherData{weatherData},
	}

	return StationRepoResponse{Content: response.Datos[0]}
}
