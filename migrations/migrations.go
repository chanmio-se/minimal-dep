// Package migrations embeds SQL migrations in lexical application order.
package migrations

import "embed"

// Files contains all migration SQL files.
//
//go:embed *.sql
var Files embed.FS
