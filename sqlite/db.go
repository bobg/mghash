package sqlite

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"

	"github.com/bobg/mghash"
)

// DB is an implementation of mghash.DB that uses a Sqlite3 file for persistent storage.
type DB struct {
	db   *sql.DB
	keep time.Duration
}

var _ mghash.DB = &DB{}

const schema = `
CREATE TABLE IF NOT EXISTS hashes (
  hash BLOB NOT NULL PRIMARY KEY,
  unix_secs INT NOT NULL
);
`

// Open opens the given file and returns it as a *DB.
// The file is created if it doesn't already exist.
// The database schema is created in the file if needed.
// Callers should call Close when finished operating on the database.
func Open(ctx context.Context, path string, opts ...Option) (*DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, errors.Wrapf(err, "opening sqlite db %s", path)
	}
	_, err = db.ExecContext(ctx, schema)
	if err != nil {
		return nil, errors.Wrap(err, "creating schema")
	}
	result := &DB{db: db}
	for _, opt := range opts {
		opt(result)
	}
	return result, nil
}

// Close releases the resources of s.
func (db *DB) Close() error {
	return db.db.Close()
}

// Option is the type of a config option that can be passed to Open.
type Option func(*DB)

// Keep is an Option that sets the amount of time to keep a database entry.
// By default, DB keeps all entries.
// Using Keep(d) allows DB to evict entries whose last-access time is older than d.
func Keep(d time.Duration) Option {
	return func(db *DB) {
		db.keep = d
	}
}

// Has tells whether db contains the given hash.
// If found, it also updates the last-access time of the hash.
func (db *DB) Has(ctx context.Context, h []byte) (bool, error) {
	const q = `UPDATE hashes SET unix_secs = $1 WHERE hash = $2`
	res, err := db.db.ExecContext(ctx, q, time.Now().Unix(), h)
	if err != nil {
		return false, errors.Wrap(err, "updating database")
	}
	aff, err := res.RowsAffected()
	return aff > 0, errors.Wrap(err, "counting affected rows")
}

// Add adds a hash to db.
// If it is already present, its last-access time is updated.
// If db was opened with the Keep option,
// entries with old last-access times are evicted.
func (db *DB) Add(ctx context.Context, h []byte) error {
	const q = `INSERT INTO hashes (hash, unix_secs) VALUES ($1, $2) ON CONFLICT DO UPDATE SET unix_secs = $2 WHERE hash = $1`
	_, err := db.db.ExecContext(ctx, q, h, time.Now().Unix())
	if err != nil {
		return errors.Wrap(err, "adding hash to database")
	}
	if db.keep > 0 {
		const q2 = `DELETE FROM hashes WHERE unix_secs < $1`
		_, err = db.db.ExecContext(ctx, q, time.Now().Add(-db.keep).Unix())
		if err != nil {
			return errors.Wrap(err, "evicting expired database entries")
		}
	}
	return nil
}
