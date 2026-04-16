.PHONY: fmt lint test check sqlc-generate recommendation-test-integration e2e-test \
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

sqlc-generate:
	sqlc generate -f internal/learningengine/infrastructure/persistence/sqlc.yaml
	sqlc generate -f internal/recommendation/infrastructure/persistence/sqlc.yaml

recommendation-test-integration:
	go test -tags=integration ./internal/recommendation/test/integration/...

e2e-test:
	go test -tags=e2e ./internal/test/e2e/...

check:
	gofmt -w cmd internal
	go vet ./...
	go test ./...

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
