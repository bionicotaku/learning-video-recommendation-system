GO ?= go
GO_PACKAGES := ./...
GO_FILES := $(shell find . -type f -name '*.go' -not -path './vendor/*' | sort)
LEARNINGENGINE_MIGRATIONS_DIR := internal/learningengine/infrastructure/migration
RECOMMENDATION_MIGRATIONS_DIR := internal/recommendation/scheduler/infrastructure/migration
MIGRATE := $(GO) run -tags 'postgres,file' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3

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

.PHONY: learningengine-sqlc-generate
learningengine-sqlc-generate: sqlc-generate

.PHONY: recommendation-sqlc-generate
recommendation-sqlc-generate: sqlc-generate

.PHONY: accept
accept: sqlc-generate lint

.PHONY: test
test:
	@$(GO) test ./...

.PHONY: check
check: accept test

.PHONY: learningengine-migrate-up
learningengine-migrate-up:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@db_url="$(DATABASE_URL)"; \
	sep='?'; \
	case "$$db_url" in *\?*) sep='&';; esac; \
	$(MIGRATE) -path $(LEARNINGENGINE_MIGRATIONS_DIR) -database "$${db_url}$${sep}x-migrations-table=learningengine_schema_migrations" up

.PHONY: learningengine-migrate-down
learningengine-migrate-down:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@db_url="$(DATABASE_URL)"; \
	sep='?'; \
	case "$$db_url" in *\?*) sep='&';; esac; \
	$(MIGRATE) -path $(LEARNINGENGINE_MIGRATIONS_DIR) -database "$${db_url}$${sep}x-migrations-table=learningengine_schema_migrations" down -all

.PHONY: learningengine-migrate-version
learningengine-migrate-version:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@db_url="$(DATABASE_URL)"; \
	sep='?'; \
	case "$$db_url" in *\?*) sep='&';; esac; \
	$(MIGRATE) -path $(LEARNINGENGINE_MIGRATIONS_DIR) -database "$${db_url}$${sep}x-migrations-table=learningengine_schema_migrations" version

.PHONY: learningengine-migrate-force
learningengine-migrate-force:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@test -n "$(VERSION)" || (echo "VERSION is required" && exit 1)
	@db_url="$(DATABASE_URL)"; \
	sep='?'; \
	case "$$db_url" in *\?*) sep='&';; esac; \
	$(MIGRATE) -path $(LEARNINGENGINE_MIGRATIONS_DIR) -database "$${db_url}$${sep}x-migrations-table=learningengine_schema_migrations" force "$(VERSION)"

.PHONY: recommendation-migrate-up
recommendation-migrate-up:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@db_url="$(DATABASE_URL)"; \
	sep='?'; \
	case "$$db_url" in *\?*) sep='&';; esac; \
	$(MIGRATE) -path $(RECOMMENDATION_MIGRATIONS_DIR) -database "$${db_url}$${sep}x-migrations-table=recommendation_schema_migrations" up

.PHONY: recommendation-migrate-down
recommendation-migrate-down:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@db_url="$(DATABASE_URL)"; \
	sep='?'; \
	case "$$db_url" in *\?*) sep='&';; esac; \
	$(MIGRATE) -path $(RECOMMENDATION_MIGRATIONS_DIR) -database "$${db_url}$${sep}x-migrations-table=recommendation_schema_migrations" down -all

.PHONY: recommendation-migrate-version
recommendation-migrate-version:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@db_url="$(DATABASE_URL)"; \
	sep='?'; \
	case "$$db_url" in *\?*) sep='&';; esac; \
	$(MIGRATE) -path $(RECOMMENDATION_MIGRATIONS_DIR) -database "$${db_url}$${sep}x-migrations-table=recommendation_schema_migrations" version

.PHONY: recommendation-migrate-force
recommendation-migrate-force:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@test -n "$(VERSION)" || (echo "VERSION is required" && exit 1)
	@db_url="$(DATABASE_URL)"; \
	sep='?'; \
	case "$$db_url" in *\?*) sep='&';; esac; \
	$(MIGRATE) -path $(RECOMMENDATION_MIGRATIONS_DIR) -database "$${db_url}$${sep}x-migrations-table=recommendation_schema_migrations" force "$(VERSION)"
