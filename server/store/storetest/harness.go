// Package storetest provides a session-scoped PostgreSQL container for
// integration tests against the Insights store layer. The first call to NewDB
// in a `go test` run boots a postgres:14-alpine container, applies the minimal
// schema in schema.sql, and caches the resulting *sql.DB. Subsequent calls
// reuse it; only the data is reset between calls.
package storetest

import (
	"context"
	"database/sql"
	_ "embed"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

//go:embed schema.sql
var schemaSQL string

// resetTables is truncated between tests so each test starts with an empty
// fixture set. Extend as new insights add tables.
var resetTables = []string{
	"reactions",
	"posts",
	"threadmemberships",
	"threads",
	"channelmembers",
	"publicchannels",
	"channels",
	"teammembers",
	"bots",
	"users",
	"focalboard_blocks_history",
	"focalboard_boards_history",
	"focalboard_board_members",
	"focalboard_boards",
	"ir_incident",
	"ir_playbookmember",
	"ir_playbook",
}

var (
	mu       sync.Mutex
	cachedDB *sql.DB
)

// NewDB returns a *sql.DB connected to the session's PostgreSQL test
// container, with the test tables truncated. If Docker is not reachable, the
// test is skipped with a clear message.
func NewDB(t *testing.T) *sql.DB {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()

	if cachedDB == nil {
		dsn, err := startContainer()
		if err != nil {
			t.Skipf("storetest: cannot start postgres container (%v)", err)
			return nil
		}
		db, openErr := sql.Open("postgres", dsn)
		if openErr != nil {
			t.Fatalf("storetest: open db: %v", openErr)
		}
		if _, err := db.Exec(schemaSQL); err != nil {
			t.Fatalf("storetest: apply schema: %v", err)
		}
		cachedDB = db
	}

	if _, err := cachedDB.Exec("TRUNCATE " + strings.Join(resetTables, ", ") + " RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("storetest: truncate: %v", err)
	}
	return cachedDB
}

func startContainer() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	container, err := postgres.Run(ctx,
		"postgres:14-alpine",
		postgres.WithDatabase("insights_test"),
		postgres.WithUsername("insights"),
		postgres.WithPassword("insights"),
		testcontainers.WithWaitStrategy(
			tcwait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return "", err
	}
	return container.ConnectionString(ctx, "sslmode=disable")
}
