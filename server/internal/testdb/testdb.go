// Package testdb hands tests their own database.
//
// It exists because of a bug that cost real time (docs/KnownGaps.md G-20). The
// suite used to run against the DEVELOPMENT database, and the auth tests have
// to be able to get back to "nobody has an account here" — so running
// `go test ./...` deleted whatever account the owner had created, and left
// conversations behind that later white-screened the sidebar (docs/Bugs.md
// B-13). Tests and the thing you are developing must not share a database.
//
// So the default points at sparstrowgen_test, which this package creates and
// migrates on first use. Nothing to set up and nothing to remember, which is
// the only version of this that survives contact with a hurry.
package testdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver, for goose
	"github.com/pressly/goose/v3"
)

// DefaultDSN is a DIFFERENT database on the same Postgres the development
// stack uses, so `make db` is still the only thing anyone has to start.
//
// The name here is a BASE. Each package gets its own database derived from it
// — see Pool.
const DefaultDSN = "postgres://sparstrowgen:sparstrowgen@localhost:5433/sparstrowgen_test?sslmode=disable"

// prepared guards the create-and-migrate, which every package's tests would
// otherwise race each other to do.
var prepared sync.Once

// preparedErr is the outcome of that one attempt, so the second caller is told
// the same thing as the first rather than silently proceeding.
var preparedErr error

// Pool returns a connection pool to a migrated test database, creating and
// migrating it if this is the first call in the process.
//
// Skips rather than fails when Postgres is not running at all: not every
// checkout has the stack up, and a skip says "this was not run" where a failure
// would say "this is broken".
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = DefaultDSN
	}
	// One database PER PACKAGE, not one for the suite. `go test ./...` runs
	// packages concurrently in separate processes, and several of these tests
	// have to empty the users table to get back to "nobody has claimed this
	// app" — so a shared test database means the api package wipes the account
	// the store package is in the middle of using. That is the same bug as
	// G-20 one level down, and the same fix: do not share.
	dsn, err := perPackage(dsn)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}

	prepared.Do(func() { preparedErr = prepare(dsn) })
	if preparedErr != nil {
		t.Skipf("no test database: %v", preparedErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("test database unreachable: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// prepare creates the database if it is missing, then brings it to the latest
// migration.
func prepare(dsn string) error {
	name, admin, err := splitDSN(dsn)
	if err != nil {
		return err
	}

	// CREATE DATABASE cannot run inside a transaction and cannot target the
	// database you are connected to, hence the connection to `postgres`.
	adminDB, err := sql.Open("pgx", admin)
	if err != nil {
		return err
	}
	defer func() { _ = adminDB.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := adminDB.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres is not running — start it with `make db`: %w", err)
	}

	var exists bool
	if err := adminDB.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		// Quoted with the identifier rules rather than interpolated: the name
		// comes from an environment variable, and this connection is a
		// superuser on somebody's development machine.
		//
		// `go test ./...` runs each package in its own PROCESS, so the sync.Once
		// above does not serialise them — two packages can both find the
		// database missing and both try to create it. The loser gets 42P04,
		// which means the database now exists, which is what it wanted.
		_, err := adminDB.ExecContext(ctx, `CREATE DATABASE `+quoteIdent(name))
		if err != nil && !isDuplicateDatabase(err) {
			return err
		}
	}

	dir, err := migrationsDir()
	if err != nil {
		return err
	}

	testDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer func() { _ = testDB.Close() }()

	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	// Same reason: concurrent test processes would otherwise race to apply the
	// same migrations, and goose is not safe against that on its own. An
	// advisory lock is held for the duration of this connection's migration and
	// released on unlock, so the second process waits and then finds nothing to
	// do. The constant is arbitrary and only has to be the same everywhere.
	const migrationLock = 0x5B0A_7570 // "sparrow", loosely
	if _, err := testDB.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrationLock); err != nil {
		return err
	}
	defer func() {
		_, _ = testDB.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", migrationLock)
	}()
	// The same migration files the real database runs. Applying the schema any
	// other way — a dump, a hand-written CREATE TABLE — is how a suite ends up
	// passing against a schema production does not have.
	return goose.Up(testDB, dir)
}

// perPackage appends the name of the package under test to the database name,
// so every package's tests own their data outright. `go test` runs each package
// with the working directory set to that package, which is where the name comes
// from.
func perPackage(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	suffix := sanitise(filepath.Base(wd))
	if suffix == "" {
		return dsn, nil
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "_" + suffix
	return u.String(), nil
}

// sanitise keeps a package name usable as part of an identifier.
func sanitise(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

// splitDSN returns the database name and a DSN for the `postgres` maintenance
// database on the same server.
func splitDSN(dsn string) (name string, admin string, err error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", "", err
	}
	name = strings.TrimPrefix(u.Path, "/")
	if name == "" {
		return "", "", fmt.Errorf("no database name in %q", dsn)
	}
	u.Path = "/postgres"
	return name, u.String(), nil
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// isDuplicateDatabase reports Postgres error 42P04, which means another process
// created the database between our check and our CREATE.
func isDuplicateDatabase(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P04"
}

// migrationsDir finds server/migrations from wherever `go test` happens to have
// put the working directory, which is the package under test.
func migrationsDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find server/migrations above the working directory")
		}
		dir = parent
	}
}
