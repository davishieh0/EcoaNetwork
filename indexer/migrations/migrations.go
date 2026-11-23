// Package migrations holds the indexer's database migrations, embedded in the
// binary so the image carries them.
package migrations


import (
	"embed"
)


// --- GLOBALS ---

// FS is every migration, named <version>_<what>.sql as goose reads them.
//
//go:embed *.sql
var FS embed.FS
