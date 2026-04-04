GO ?= go
SCHEDULER_MIGRATIONS_DIR := internal/recommendation/scheduler/infrastructure/migration
SCHEDULER_MIGRATE := $(GO) run -tags 'postgres,file' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3

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

.PHONY: scheduler-sqlc-generate
scheduler-sqlc-generate:
	@sqlc generate
