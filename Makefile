# Everything needed to run sparstrowgen locally.
# The stack itself is described in .sparstrowgen/blueprint.yaml — this file only
# says how to start it.

GOOSE_DRIVER   ?= postgres
GOOSE_DBSTRING ?= postgres://sparstrowgen:sparstrowgen@localhost:5433/sparstrowgen?sslmode=disable
export GOOSE_DRIVER
export GOOSE_DBSTRING

.PHONY: help db migrate server daemon web build check test test-linux clean

help:
	@echo "make db       - start Postgres in Docker"
	@echo "make migrate  - apply migrations"
	@echo "make server   - run the API server on :8080"
	@echo "make daemon   - run the daemon (drives the agent CLIs)"
	@echo "make web      - run the Next.js app on :3000"
	@echo "make check    - typecheck, lint, vet and build everything"
	@echo "make test-linux - the Go suite on Linux, under the race detector"
	@echo ""
	@echo "First run:  make db && make migrate"
	@echo "Then, in three terminals: make server / make daemon / make web"

db:
	docker compose up -d
	@echo "waiting for postgres..."
	@until docker exec sparstrowgen-db pg_isready -U sparstrowgen -d sparstrowgen >/dev/null 2>&1; do sleep 1; done
	@echo "postgres ready on :5433"

migrate:
	cd server && goose -dir migrations up

# The server never reaches out to the owner's machine; the daemon dials in.
server:
	cd server && go run ./cmd/server

# Runs where the agent CLIs are installed. Needs claude/codex/agy on PATH.
daemon:
	cd server && go run ./cmd/daemon

web:
	pnpm dev

build:
	cd server && go build -o ../bin/sparstrowgen-server ./cmd/server
	cd server && go build -o ../bin/sparstrowgen-daemon ./cmd/daemon
	pnpm build

# One command for the whole pipeline, the way Multica does it (D-009).
check:
	pnpm typecheck
	pnpm lint
	cd server && go vet ./...
	cd server && go build ./...
	pnpm build

test:
	cd server && go test ./...

# The Go suite on Linux, with the race detector.
#
# Two things this machine cannot do itself, and both were open gaps until it
# existed (G-15, G-16). The daemon only ever runs on Windows, so the process
# GROUP half of stopping a turn — SIGTERM then SIGKILL to -pgid — had never
# executed anywhere; and -race needs cgo, which needs a C compiler, and there is
# no gcc on PATH here. A container has both.
#
# host.docker.internal reaches the Postgres that `make db` publishes on 5433.
# Without it the store and api tests SKIP rather than fail, which would quietly
# turn this into a much weaker check than it looks.
#
# --init is load-bearing, not decoration. Without it `go test` is PID 1, and PID
# 1 in a container does not reap orphaned children — so a process the stop
# correctly killed lingers as a zombie, and a zombie still answers kill(pid, 0)
# as though it were alive. The tree tests then report a clean stop as
# unconfirmed. A real Linux host has an init doing this, so the container needs
# one to behave like the thing being tested for. See docs/KnownGaps.md G-17.
test-linux:
	docker run --rm --init \
		-v "$(CURDIR)/server:/src" -w /src \
		-e GOFLAGS=-buildvcs=false -e CGO_ENABLED=1 \
		-e TEST_DATABASE_URL="postgres://sparstrowgen:sparstrowgen@host.docker.internal:5433/sparstrowgen?sslmode=disable" \
		golang:1.27 go test ./... -race

clean:
	rm -rf bin
	docker compose down
