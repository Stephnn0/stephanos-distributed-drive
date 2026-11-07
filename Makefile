.PHONY: proto lint breaking migrate-up migrate-down

## proto: lint the API contract, then generate Go code into backend/gen/
proto: lint
	buf generate
	cd backend && go mod tidy

## lint: check protos follow the style rules in buf.yaml
lint:
	buf lint

## breaking: fail if protos break compatibility with main
breaking:
	buf breaking --against '.git#branch=main'

# Local Postgres for the metadata service. Override with: make migrate-up DB_URL=...
DB_URL ?= postgres://localhost:5432/drive_metadata?sslmode=disable
MIGRATIONS := backend/services/metadata/migrations

## migrate-up: apply all pending database migrations
migrate-up:
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" up

## migrate-down: undo the most recent migration
migrate-down:
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" down 1
