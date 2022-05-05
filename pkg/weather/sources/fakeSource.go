package sources

import (
	"context"
	"fmt"
	"math/rand"
	"temperature/pkg/weather"
	"time"
)

type FakeSource struct{}

func (f FakeSource) GetWeatherByDate(ctx context.Context, date *time.Time, coordinate *weather.Coordinates) (float32, error) {
	v := rand.Float32()
	if v > 0.8 {
		return 0, fmt.Errorf("New Error")
	}
	return v, nil
}
