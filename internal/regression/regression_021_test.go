package regression

import (
	"testing"

	"github.com/jb843051627/cave-echo/internal/model"
)

func TestBug21_InRangeTreatsBoundsAsInside(t *testing.T) {
	sensor := model.Sensor{MinValue: 10, MaxValue: 90}
	for _, value := range []float64{10, 50, 90} {
		if !sensor.InRange(value) {
			t.Fatalf("value %v on closed range [10, 90] boundary reported outside", value)
		}
	}
	for _, value := range []float64{9.9, 90.1} {
		if sensor.InRange(value) {
			t.Fatalf("value %v outside range [10, 90] reported inside", value)
		}
	}
}
