package storage_test

import (
	"testing"

	"github.com/buggregator/go-buggregator/internal/storage"
)

func TestMigrator_AddSQL_and_Run(t *testing.T) {
	db := setupTestDB(t) // reuse from sqlite_test.go
	m := storage.NewMigrator(db)

	m.AddSQL("test", "001_create_test.sql", "CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)")

	if err := m.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify table was created
	_, err := db.Exec("INSERT INTO test_table (id, name) VALUES (1, 'hello')")
	if err != nil {
		t.Fatalf("table not created: %v", err)
	}

	// Verify migration is tracked
	var count int
	db.QueryRow("SELECT COUNT(*) FROM migrations WHERE name = ?", "test/001_create_test.sql").Scan(&count)
	if count != 1 {
		t.Errorf("migration not tracked, count = %d", count)
	}
}

func TestMigrator_SkipsAlreadyApplied(t *testing.T) {
	db := setupTestDB(t)
	m := storage.NewMigrator(db)

	m.AddSQL("test", "001_create.sql", "CREATE TABLE skip_test (id INTEGER)")
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}

	// Run again with same migration — should not fail
	m2 := storage.NewMigrator(db)
	m2.AddSQL("test", "001_create.sql", "CREATE TABLE skip_test (id INTEGER)")
	if err := m2.Run(); err != nil {
		t.Fatalf("second Run should skip: %v", err)
	}
}

func TestMigrator_SortsByName(t *testing.T) {
	db := setupTestDB(t)
	m := storage.NewMigrator(db)

	// Add in reverse order
	m.AddSQL("test", "002_add_col.sql", "ALTER TABLE ordered_test ADD COLUMN name TEXT")
	m.AddSQL("test", "001_create.sql", "CREATE TABLE ordered_test (id INTEGER PRIMARY KEY)")

	if err := m.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify both applied (table exists with name column)
	_, err := db.Exec("INSERT INTO ordered_test (id, name) VALUES (1, 'test')")
	if err != nil {
		t.Fatalf("migration order incorrect: %v", err)
	}
}

func TestMigrator_MultipleStatements(t *testing.T) {
	db := setupTestDB(t)
	m := storage.NewMigrator(db)

	m.AddSQL("test", "001_multi.sql", `
		CREATE TABLE multi_a (id INTEGER);
		CREATE TABLE multi_b (id INTEGER);
	`)

	if err := m.Run(); err != nil {
		t.Fatal(err)
	}

	// Verify both tables exist
	_, err := db.Exec("INSERT INTO multi_a (id) VALUES (1)")
	if err != nil {
		t.Fatal("multi_a not created")
	}
	_, err = db.Exec("INSERT INTO multi_b (id) VALUES (1)")
	if err != nil {
		t.Fatal("multi_b not created")
	}
}
