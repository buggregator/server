package storage

import "embed"

//go:embed migrations/*.sql
var CoreMigrations embed.FS
