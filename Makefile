GO ?= go
GO_PACKAGES := ./...
GO_FILES := $(shell find . -type f -name '*.go' -not -path './vendor/*' | sort)
SCHEDULER_MIGRATIONS_DIR := internal/recommendation/scheduler/infrastructure/migration
SCHEDULER_MIGRATE := $(GO) run -tags 'postgres,file' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3

.PHONY: fmt
fmt:
	@test -n "$(GO_FILES)" || (echo "no Go files found" && exit 1)
	@gofmt -w $(GO_FILES)

.PHONY: fmt-check
fmt-check:
	@test -n "$(GO_FILES)" || (echo "no Go files found" && exit 1)
	@unformatted="$$(gofmt -l $(GO_FILES))"; \
	if [ -n "$$unformatted" ]; then \
		echo "Go files are not gofmt-formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

.PHONY: vet
vet:
	@$(GO) vet $(GO_PACKAGES)

.PHONY: staticcheck
staticcheck:
	@staticcheck $(GO_PACKAGES)

.PHONY: lint
lint: fmt-check vet staticcheck

.PHONY: sqlc-generate
sqlc-generate:
	@sqlc generate

.PHONY: accept
accept: sqlc-generate lint

.PHONY: test
test:
	@$(GO) test ./...

.PHONY: check
check: accept test

.PHONY: scheduler-migrate-up
scheduler-migrate-up:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@$(SCHEDULER_MIGRATE) -path $(SCHEDULER_MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

.PHONY: scheduler-migrate-down
scheduler-migrate-down:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@$(SCHEDULER_MIGRATE) -path $(SCHEDULER_MIGRATIONS_DIR) -database "$(DATABASE_URL)" down -all

.PHONY: scheduler-migrate-version
scheduler-migrate-version:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@$(SCHEDULER_MIGRATE) -path $(SCHEDULER_MIGRATIONS_DIR) -database "$(DATABASE_URL)" version

.PHONY: scheduler-migrate-force
scheduler-migrate-force:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@test -n "$(VERSION)" || (echo "VERSION is required" && exit 1)
	@$(SCHEDULER_MIGRATE) -path $(SCHEDULER_MIGRATIONS_DIR) -database "$(DATABASE_URL)" force "$(VERSION)"
