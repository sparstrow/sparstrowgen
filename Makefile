# Everything needed to run sparstrowgen locally.
# The stack itself is described in .sparstrowgen/blueprint.yaml — this file only
# says how to start it.

GOOSE_DRIVER   ?= postgres
GOOSE_DBSTRING ?= postgres://sparstrowgen:sparstrowgen@localhost:5433/sparstrowgen?sslmode=disable
export GOOSE_DRIVER
export GOOSE_DBSTRING

.PHONY: help db migrate server daemon web build check test clean

help:
	@echo "make db       - start Postgres in Docker"
	@echo "make migrate  - apply migrations"
	@echo "make server   - run the API server on :8080"
	@echo "make daemon   - run the daemon (drives the agent CLIs)"
	@echo "make web      - run the Next.js app on :3000"
	@echo "make check    - typecheck, lint, vet and build everything"
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

clean:
	rm -rf bin
	docker compose down
