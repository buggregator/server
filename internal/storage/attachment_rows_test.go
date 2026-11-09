package storage_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/buggregator/server/v2/internal/storage"
)

func TestDeleteAttachmentRows(t *testing.T) {
	db := setupTestDB(t)
	if _, err := db.Exec(`CREATE TABLE att_rows_test (uuid TEXT PRIMARY KEY, event_uuid TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DROP TABLE att_rows_test`) })

	// More events than one chunk, two rows each, plus one event that stays.
	var del []string
	for i := 0; i < 1200; i++ {
		ev := "ev" + strconv.Itoa(i)
		del = append(del, ev)
		db.Exec(`INSERT INTO att_rows_test VALUES (?, ?), (?, ?)`, ev+"-a", ev, ev+"-b", ev)
	}
	db.Exec(`INSERT INTO att_rows_test VALUES ('keep-a', 'keep')`)

	if err := storage.DeleteAttachmentRows(context.Background(), db, "att_rows_test", del); err != nil {
		t.Fatal(err)
	}
	var left []string
	rows, err := db.Query(`SELECT uuid FROM att_rows_test`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var u string
		rows.Scan(&u)
		left = append(left, u)
	}
	if len(left) != 1 || left[0] != "keep-a" {
		t.Errorf("rows left = %v, want [keep-a]", left)
	}

	if err := storage.DeleteAttachmentRows(context.Background(), db, "att_rows_test", nil); err != nil {
		t.Errorf("empty list: %v", err)
	}
}
