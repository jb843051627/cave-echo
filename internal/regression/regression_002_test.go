package regression

import (
	"testing"

	"github.com/jb843051627/cave-echo/internal/metrics"
)

func TestBug02_SnapshotIsolatedFromInternalMap(t *testing.T) {
	reg := metrics.New()
	reg.Add("readings.accepted", 1)
	snap := reg.Snapshot()
	snap["readings.accepted"] = 999
	if got := reg.Get("readings.accepted"); got != 1 {
		t.Fatalf("snapshot aliases internal map: get=%d want=1", got)
	}
}
