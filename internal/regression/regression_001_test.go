package regression

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jb843051627/cave-echo/internal/cache"
	"github.com/jb843051627/cave-echo/internal/model"
)

var baseTime = time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)

func TestBug01_ApplyConcurrentWritersStayExclusive(t *testing.T) {
	c := cache.New()
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				c.Apply(model.Reading{
					SiteID:     fmt.Sprintf("site-%02d", g),
					ChamberID:  fmt.Sprintf("ch-%02d-%d", g, i%7),
					SensorID:   fmt.Sprintf("sen-%02d-%03d", g, i),
					SensorType: model.SensorTemperature,
					ObservedAt: baseTime.Add(time.Duration(i) * time.Second),
					Value:      float64(i),
					Quality:    model.QualityGood,
				})
			}
		}(g)
	}
	wg.Wait()
}
