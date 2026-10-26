package sqlite

import "testing"

func latestMigrationVersion(t *testing.T) int64 {
	t.Helper()
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) == 0 {
		t.Fatal("no SQLite migrations are registered")
	}
	return int64(migrations[len(migrations)-1].version)
}
