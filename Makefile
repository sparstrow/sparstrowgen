# Everything needed to run sparstrowgen locally.
# The stack itself is described in .sparstrowgen/blueprint.yaml — this file only
# says how to start it.

# This worktree's own ports, database volume, daemon home and token, assigned
# once by `devstack` and recorded outside the checkout so two agents working at
# the same time cannot collide (WORKFLOW.md, Local isolation). The main checkout
# always gets slot 0, which is 5433/8080/3000 — today's stack, unchanged.
#
# Regenerated with `make stack`; `make stack-release` gives the slot back when a
# worktree is deleted. `export` with no arguments passes all of it to recipes.
STACK_ENV := .stack.env

$(STACK_ENV):
	@cd server && go run ./cmd/devstack env -dir .. > ../$(STACK_ENV)

-include $(STACK_ENV)
export

.PHONY: help db migrate server daemon web build check test test-linux clean stack stack-release

help:
	@echo "make db       - start Postgres in Docker"
	@echo "make migrate  - apply migrations"
	@echo "make server   - run the API server on $(ADDR)"
	@echo "make daemon   - run the daemon (drives the agent CLIs)"
	@echo "make web      - run the Next.js app on :$(PORT)"
	@echo "make stack    - show this worktree's own ports and reserve them"
	@echo "make check    - typecheck, lint, vet and build everything"
	@echo "make test-linux - the Go suite on Linux, under the race detector"
	@echo ""
	@echo "First run:  make db && make migrate"
	@echo "Then, in three terminals: make server / make daemon / make web"
	@echo ""
	@echo "Running go test directly? Export this worktree's stack first, or it"
	@echo "will use another checkout's Postgres:  set -a; . ./.stack.env; set +a"

# -f names the file: docker-compose.yaml in this repo is what COOLIFY deploys,
# and a bare `docker compose up` here would pick one of the two by a precedence
# rule nobody remembers.
db: $(STACK_ENV)
	docker compose -f compose.dev.yaml up -d
	@echo "waiting for postgres..."
	@until docker compose -f compose.dev.yaml exec -T db pg_isready -U sparstrowgen -d sparstrowgen >/dev/null 2>&1; do sleep 1; done
	@echo "postgres ready on :$(SPARSTROWGEN_DB_PORT)"

# What this worktree was given. Run it to see where its app and database are.
stack:
	@rm -f $(STACK_ENV)
	@$(MAKE) --no-print-directory $(STACK_ENV)
	@cat $(STACK_ENV)

# Hand the slot back — for a worktree that is being deleted, so the next one can
# have its ports.
stack-release:
	@cd server && go run ./cmd/devstack release -dir ..
	@rm -f $(STACK_ENV)

migrate: $(STACK_ENV)
	cd server && goose -dir migrations up

# Development configuration. NOT secrets: the token below is in a public
# repository and only ever unlocks a server on localhost. Production sets these
# in Coolify (docs/runbooks/deploy.md).
#
# There is no password here. Accounts live in Postgres: register the owner
# address below at http://localhost:3000/register, exactly as in production.
# With MAIL_TRANSPORT=log the confirmation link is printed in the server's
# output instead of being emailed.
#
# These live here rather than as defaults in the program on purpose. The server
# refuses to start without them, so there is exactly one code path and no
# "authentication off" or "mail off" mode that could reach production by
# accident.
# DAEMON_TOKEN, WEB_ORIGIN, the database URL and the daemon's home come from
# $(STACK_ENV) above, so they differ per worktree. Only what is the same
# everywhere is written here.
DEV_OWNER_EMAIL   ?= owner@localhost.test

export OWNER_EMAIL         = $(DEV_OWNER_EMAIL)
# Development only; the server refuses it when SESSION_SECURE is on.
export MAIL_TRANSPORT      = log

# The server never reaches out to the owner's machine; the daemon dials in.
server: $(STACK_ENV)
	cd server && go run ./cmd/server

# Runs where the agent CLIs are installed. Needs claude/codex/agy on PATH.
daemon: $(STACK_ENV)
	cd server && go run ./cmd/daemon

web: $(STACK_ENV)
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

test: $(STACK_ENV)
	cd server && go test ./...

# The Go suite on Linux, with the race detector.
#
# Two things this machine cannot do itself, and both were open gaps until it
# existed (G-15, G-16). The daemon only ever runs on Windows, so the process
# GROUP half of stopping a turn — SIGTERM then SIGKILL to -pgid — had never
# executed anywhere; and -race needs cgo, which needs a C compiler, and there is
# no gcc on PATH here. A container has both.
#
# host.docker.internal reaches the Postgres that `make db` publishes — on this
# worktree's own port, not a fixed 5433, which would be another checkout's
# database. Without it the store and api tests SKIP rather than fail, which
# would quietly turn this into a much weaker check than it looks.
#
# --init is load-bearing, not decoration. Without it `go test` is PID 1, and PID
# 1 in a container does not reap orphaned children — so a process the stop
# correctly killed lingers as a zombie, and a zombie still answers kill(pid, 0)
# as though it were alive. The tree tests then report a clean stop as
# unconfirmed. A real Linux host has an init doing this, so the container needs
# one to behave like the thing being tested for. See docs/KnownGaps.md G-17.
test-linux: $(STACK_ENV)
	docker run --rm --init \
		-v "$(CURDIR)/server:/src" -w /src \
		-e GOFLAGS=-buildvcs=false -e CGO_ENABLED=1 \
		-e TEST_DATABASE_URL="postgres://sparstrowgen:sparstrowgen@host.docker.internal:$(SPARSTROWGEN_DB_PORT)/sparstrowgen_test?sslmode=disable" \
		golang:1.27 go test ./... -race

clean:
	rm -rf bin
	docker compose -f compose.dev.yaml down
