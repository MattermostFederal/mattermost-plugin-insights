package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// Store wraps the plugin's database access. Read-heavy aggregation queries
// hit the replica; the master is exposed for completeness but Insights does
// not write. PostgreSQL is assumed — Mattermost ended MySQL support before
// this plugin was created, so all queries can use Postgres-specific syntax
// (`string_agg`, `SPLIT_PART`, `Posts.Props ->>`, ...) freely.
type Store struct {
	master  *sql.DB
	replica *sql.DB
	Builder sq.StatementBuilderType
}

// New returns a Store backed by the given pluginapi client.
func New(client *pluginapi.Client) (*Store, error) {
	master, err := client.Store.GetMasterDB()
	if err != nil {
		return nil, err
	}
	replica, err := client.Store.GetReplicaDB()
	if err != nil {
		return nil, err
	}
	return &Store{
		master:  master,
		replica: replica,
		Builder: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}, nil
}

// Replica returns the read-replica connection. All Insights queries should use
// this; the master is reserved for any future write paths.
func (s *Store) Replica() *sql.DB { return s.replica }

// NewFromDB constructs a Store directly from an open *sql.DB. Used by the
// test harness, which connects to a testcontainers-managed Postgres rather
// than going through the plugin RPC driver.
func NewFromDB(db *sql.DB) *Store {
	return &Store{
		master:  db,
		replica: db,
		Builder: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}
