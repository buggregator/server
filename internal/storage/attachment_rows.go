package storage

import (
	"context"
	"database/sql"
	"strings"
)

// DeleteAttachmentRows removes the rows of an attachments table (columns
// event_uuid, …) that belong to the given events. Modules call it from
// OnEventsDeleted; the files themselves are removed by AttachmentStore.
func DeleteAttachmentRows(ctx context.Context, db *sql.DB, table string, eventUUIDs []string) error {
	const chunk = 500
	for len(eventUUIDs) > 0 {
		n := min(chunk, len(eventUUIDs))
		part := eventUUIDs[:n]
		eventUUIDs = eventUUIDs[n:]

		args := make([]any, len(part))
		for i, u := range part {
			args[i] = u
		}
		q := `DELETE FROM ` + table + ` WHERE event_uuid IN (?` + strings.Repeat(",?", len(part)-1) + `)`
		if _, err := db.ExecContext(ctx, q, args...); err != nil {
			return err
		}
	}
	return nil
}
