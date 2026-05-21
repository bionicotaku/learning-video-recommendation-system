.PHONY: fmt lint test quick-check check sqlc-generate semantic-test-integration learningengine-test-integration normalizer-test-integration integration-test catalog-test-integration recommendation-test-integration e2e-test \
	analytics-migrate-up analytics-migrate-down analytics-migrate-version analytics-migrate-status \
	semantic-migrate-up semantic-migrate-down semantic-migrate-version semantic-migrate-status \
	catalog-migrate-up catalog-migrate-down catalog-migrate-version catalog-migrate-status \
	learningengine-migrate-up learningengine-migrate-down learningengine-migrate-version learningengine-migrate-status \
	recommendation-migrate-up recommendation-migrate-down recommendation-migrate-version recommendation-migrate-status \
	recommendation-refresh

fmt:
	gofmt -w cmd internal

lint:
	go vet ./...

test:
	go test ./...

quick-check:
	gofmt -w cmd internal
	go vet ./...
	go test ./...

sqlc-generate:
	sqlc generate -f internal/analytics/infrastructure/persistence/sqlc.yaml
	sqlc generate -f internal/semantic/infrastructure/persistence/sqlc.yaml
	sqlc generate -f internal/catalog/infrastructure/persistence/sqlc.yaml
	sqlc generate -f internal/learningengine/reducer/infrastructure/persistence/sqlc.yaml
	sqlc generate -f internal/learningengine/normalizer/infrastructure/persistence/sqlc.yaml
	sqlc generate -f internal/recommendation/infrastructure/persistence/sqlc.yaml

semantic-test-integration:
	go test -tags=integration ./internal/semantic/test/integration/...

learningengine-test-integration:
	go test -tags=integration ./internal/learningengine/reducer/test/integration/...

normalizer-test-integration:
	go test -tags=integration ./internal/learningengine/normalizer/test/integration/...

recommendation-test-integration:
	go test -tags=integration ./internal/recommendation/test/integration/...

catalog-test-integration:
	go test -tags=integration ./internal/catalog/test/integration/...

integration-test:
	go test -tags=integration ./internal/semantic/test/integration/... ./internal/catalog/test/integration/... ./internal/learningengine/reducer/test/integration/... ./internal/learningengine/normalizer/test/integration/... ./internal/recommendation/test/integration/...

e2e-test:
	go test -tags=e2e ./internal/test/e2e/...

check:
	$(MAKE) quick-check
	$(MAKE) integration-test

analytics-migrate-up:
	go run ./cmd/dbtool migrate up --module=analytics

analytics-migrate-down:
	go run ./cmd/dbtool migrate down --module=analytics

analytics-migrate-version:
	go run ./cmd/dbtool migrate version --module=analytics

analytics-migrate-status:
	go run ./cmd/dbtool migrate status --module=analytics

semantic-migrate-up:
	go run ./cmd/dbtool migrate up --module=semantic

semantic-migrate-down:
	go run ./cmd/dbtool migrate down --module=semantic

semantic-migrate-version:
	go run ./cmd/dbtool migrate version --module=semantic

semantic-migrate-status:
	go run ./cmd/dbtool migrate status --module=semantic

catalog-migrate-up:
	go run ./cmd/dbtool migrate up --module=catalog

catalog-migrate-down:
	go run ./cmd/dbtool migrate down --module=catalog

catalog-migrate-version:
	go run ./cmd/dbtool migrate version --module=catalog

catalog-migrate-status:
	go run ./cmd/dbtool migrate status --module=catalog

learningengine-migrate-up:
	go run ./cmd/dbtool migrate up --module=learningengine

learningengine-migrate-down:
	go run ./cmd/dbtool migrate down --module=learningengine

learningengine-migrate-version:
	go run ./cmd/dbtool migrate version --module=learningengine

learningengine-migrate-status:
	go run ./cmd/dbtool migrate status --module=learningengine

recommendation-migrate-up:
	go run ./cmd/dbtool migrate up --module=recommendation

recommendation-migrate-down:
	go run ./cmd/dbtool migrate down --module=recommendation

recommendation-migrate-version:
	go run ./cmd/dbtool migrate version --module=recommendation

recommendation-migrate-status:
	go run ./cmd/dbtool migrate status --module=recommendation

recommendation-refresh:
	go run ./cmd/dbtool refresh recommendation
