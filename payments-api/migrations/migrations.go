package migrations

import "embed"

// FS holds the embedded SQL migration files for payments-api.
//
//go:embed *.sql
var FS embed.FS
