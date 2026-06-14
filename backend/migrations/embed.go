// Package migrations embeds the forward-only SQL migration files so they can
// be applied at runtime by the Postgres store.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
