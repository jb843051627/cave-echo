package regression

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jb843051627/cave-echo/internal/ingest"
)

type boomErr struct{}

func (boomErr) Error() string { return "boom" }

func TestBug04_RetryHonorsCanceledContextImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	errc := make(chan error, 1)
	go func() {
		errc <- ingest.Retry(ctx, 400, func() error { return boomErr{} })
	}()
	select {
	case err := <-errc:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("retry canceled err=%v, want context.Canceled", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("retry ignored canceled context (kept backing off)")
	}
}
