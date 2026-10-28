package internal

import (
	"path/filepath"
	"testing"

	"github.com/gosoline-project/sqlc"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/log"
)

func testLocalState(t *testing.T, path string) sqlc.Client {
	var err error
	var client sqlc.Client

	t.Helper()

	config := readStarRocksTestConfig(t, "")
	var settings *sqlc.Settings

	if settings, err = sqlc.ReadSettings(config, "local_state"); err != nil {
		t.Fatal(err)
	}

	settings.Uri.Database = path
	settings.Migrations.Path = "../build/migrations"

	if client, err = newLocalState(appctx.WithContainer(t.Context()), config, log.NewLogger(), settings); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Error(err)
		}
	})

	return client
}

func TestLocalStateGooseMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.sqlite")
	handler := &HandlerMaterializedViews{state: testLocalState(t, path)}

	if err := handler.saveRefreshPolicy(t.Context(), "analytics", "1", "EVERY(INTERVAL 5 MINUTE)"); err != nil {
		t.Fatal(err)
	}

	if err := handler.state.Close(); err != nil {
		t.Fatal(err)
	}

	handler.state = testLocalState(t, path)
	var versions int

	if err := handler.state.Get(t.Context(), &versions, "SELECT COUNT(*) FROM goose_db_version WHERE version_id = 1 AND is_applied = 1"); err != nil || versions != 1 {
		t.Fatalf("expected migration 1 applied once: %d, %v", versions, err)
	}

	if policy, err := handler.loadRefreshPolicy(t.Context(), "analytics", "1"); err != nil || policy != "EVERY(INTERVAL 5 MINUTE)" {
		t.Fatalf("migration rerun lost saved policy: %q, %v", policy, err)
	}
}
