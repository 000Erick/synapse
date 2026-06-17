//go:build sqlite_fts5 || fts5

package engram

// Importing with sqlite_fts5 build tag enables FTS5 in mattn/go-sqlite3.
// This is needed for tests that seed a temp engram.db with FTS5 tables.
import _ "github.com/mattn/go-sqlite3"
