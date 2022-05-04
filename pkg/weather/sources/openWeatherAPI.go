package sources

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"temperature/pkg/weather"
	"time"
)

const (
	urlTemplate  = "https://api.openweathermap.org/data/2.5/onecall/timemachine?lat=%f&lon=%f&dt=%d&appid=%s&units=metric"
	maxDaysSince = 3600000000000 * 24 * 4
)

type Response struct {
	Current Info   `json:"current"`
	Hourly  []Info `json:"hourly"`
}

type Info struct {
	Temperature float32 `json:"temp"`
}

type OpenWeatherSource struct {
	token *string
	http  *http.Client
}

func NewOpenWeatherAPI(token *string) weather.Source {
	api := OpenWeatherSource{
		token: token,
		http: &http.Client{
			Timeout: 10000 * time.Millisecond,
		},
	}
	return &api
}

func (w *OpenWeatherSource) GetWeatherByDate(date *time.Time, coordinate *weather.Coordinates) (float32, error) {
	if !validDate(date) {
		return 0, fmt.Errorf("Превышена максимальная глубина поиска")
	}

	request, err := w.prepareRequest(date, coordinate)
	if err != nil {
		return 0, err
	}

	res, err := w.http.Do(request)
	if err != nil {
		return 0, err
	}
	if res.StatusCode != 200 {
		message := extractBody(res)
		return 0, errors.New(*message)
	}

	defer res.Body.Close()
	data, err := extractDataFromResponse(res)
	return extractAverageTemperature(data), nil
}

func (w *OpenWeatherSource) prepareRequest(date *time.Time, coordinate *weather.Coordinates) (*http.Request, error) {
	url := fmt.Sprintf(urlTemplate, coordinate.Lat, coordinate.Lon, date.Unix(), *w.token)
	return http.NewRequest("GET", url, nil)
}

func validDate(date *time.Time) bool {
	since := time.Since(*date)
	return since.Hours() < maxDaysSince
}

func extractDataFromResponse(response *http.Response) (*Response, error) {
	data := Response{}
	body := extractBody(response)

	if err := json.Unmarshal([]byte(*body), &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func extractBody(response *http.Response) *string {
	builder := &strings.Builder{}
	if _, err := io.Copy(builder, response.Body); err != nil {
		panic(err)
	}
	str := builder.String()
	return &str
}

func extractAverageTemperature(payload *Response) float32 {
	var sumTemp float32 = 0.0
	for _, info := range payload.Hourly {
		sumTemp = sumTemp + info.Temperature
	}
	return sumTemp / float32(len(payload.Hourly))
}
