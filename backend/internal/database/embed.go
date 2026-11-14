package database

import "io/fs"

// Migrations returns the embedded SQL migrations as their own filesystem root.
func Migrations() fs.FS {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		panic(err) // the directory is embedded at compile time, so this cannot happen
	}
	return sub
}
