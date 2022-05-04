package sources

import (
	"math/rand"
	"temperature/pkg/weather"
	"time"
)

type FakeSource struct{}

func (f FakeSource) GetWeatherByDate(date *time.Time, coordinate *weather.Coordinates) (float32, error) {
	return rand.Float32(), nil
}
