package database

import "io/fs"

// mustSub returns the migrations directory as its own filesystem root.
func mustSub() fs.FS {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		panic(err) // the directory is embedded at compile time, so this cannot happen
	}
	return sub
}
