//go:build no_self_update || ios

package ghupdate

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

type forbiddenUpdateClient struct{ t *testing.T }

func (c forbiddenUpdateClient) Do(*http.Request) (*http.Response, error) {
	c.t.Fatal("disabled updater must not contact the network")
	return nil, errors.New("unexpected network request")
}

func TestUpdateDisabled(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "must-not-be-created")
	updated, err := Update(Config{ArchiveExecutable: "beszel", DataDir: dir, HttpClient: forbiddenUpdateClient{t}})
	if updated {
		t.Fatal("expected no update when self-update is disabled")
	}
	if !errors.Is(err, ErrSelfUpdateDisabled) {
		t.Fatalf("expected ErrSelfUpdateDisabled, got %v", err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("disabled updater touched the data directory: %v", err)
	}
}
