package feishu

import (
	"path/filepath"
	"testing"
)

func requireReviewStageFiveMigration(t *testing.T, version int, name string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stage-five-migration.db")
	for pass := 0; pass < 2; pass++ {
		store, err := OpenSQLiteReviewGatewayStore(path)
		if err != nil {
			t.Fatalf("open pass %d: %v", pass+1, err)
		}
		var count int
		err = store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=? AND name=?`, version, name).Scan(&count)
		_ = store.Close()
		if err != nil || count != 1 {
			t.Fatalf("migration %d/%s count=%d pass=%d err=%v", version, name, count, pass+1, err)
		}
	}
}

func TestServiceInstanceMigrationIsIdempotent(t *testing.T) {
	requireReviewStageFiveMigration(t, 16, "review_service_instances_v1")
}
