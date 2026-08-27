package regression

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/jb843051627/cave-echo/internal/store"
)

func TestBug07_AcknowledgeMissingAlertReportsNotFound(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.AcknowledgeAlert("alr-does-not-exist", time.Now().UTC()); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("ack missing alert err=%v, want ErrNotFound", err)
	}
}
