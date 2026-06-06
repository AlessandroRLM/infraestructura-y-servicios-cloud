// Package migrations embeds the SQL migration files for use with golang-migrate.
package migrations

import "embed"

// FS is the embedded filesystem containing all migration files.
//
//go:embed *.sql
var FS embed.FS
