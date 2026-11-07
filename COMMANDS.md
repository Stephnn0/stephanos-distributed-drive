# Commands

Quick reference for working on this project. Run commands from the **repo root** unless a `cd` is shown.

---

## 1. One-time setup (new machine)

```bash
# Tools
brew install go bufbuild/buf/buf golang-migrate grpcurl postgresql@18
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Make Go-installed tools (protoc-gen-go...) runnable. Add to ~/.zshrc:
export PATH="$PATH:$(go env GOPATH)/bin"

# Start Postgres (and on every login)
brew services start postgresql@18

# Create the database, then the tables
psql postgres -c "CREATE DATABASE drive_metadata;"
make migrate-up
```

---

## 2. Go dependencies

All services share one Go module: `backend/go.mod`.

```bash
cd backend

go mod download                         # install everything in go.mod (e.g. after cloning)
go get github.com/some/lib@latest       # add a new library (or upgrade one)
go get github.com/some/lib@v1.2.3       # add/pin a specific version
go mod tidy                             # remove unused libs, add missing ones (run after editing imports)
go list -m all                          # see every dependency and its version
```

---

## 3. Run the metadata service

```bash
cd backend
go run ./services/metadata/cmd/server                  # starts on port 50051
go run ./services/metadata/cmd/server -port 6000       # different port
DATABASE_URL="postgres://localhost:5432/other_db?sslmode=disable" \
  go run ./services/metadata/cmd/server                # different database
```

Stop with **Ctrl+C** (it finishes in-flight requests first).

---

## 4. Test the API by hand (grpcurl)

grpcurl is `curl` for gRPC. The server must be running (section 3).
Every call needs the `x-user-id` header (stand-in for real login).

```bash
# What services and methods exist?
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext localhost:50051 describe drive.metadata.v1.MetadataService
grpcurl -plaintext localhost:50051 describe drive.metadata.v1.Folder

# Full form of one call
grpcurl -plaintext -H "x-user-id: stephano" \
  -d '{"folder":{"display_name":"My Drive"}}' \
  localhost:50051 drive.metadata.v1.MetadataService/CreateFolder
```

Shortcut: paste this function into your terminal, then use `G <Method> '<json>'`:

```bash
G() { grpcurl -plaintext -H "x-user-id: stephano" -d "$2" localhost:50051 drive.metadata.v1.MetadataService/$1; }
```

```bash
# Create your root folder (once per user)
G CreateFolder '{"folder":{"display_name":"My Drive"}}'

# Create a folder inside root ("folders/root" = your root folder)
G CreateFolder '{"parent":"folders/root","folder":{"display_name":"Taxes"}}'

# Get one folder
G GetFolder '{"name":"folders/<id>"}'

# List folders inside root, 10 per page
G ListFolders '{"parent":"folders/root","page_size":10}'
# Next page: pass the nextPageToken from the previous response
G ListFolders '{"parent":"folders/root","page_size":10,"page_token":"<token>"}'

# Rename
G UpdateFolder '{"folder":{"name":"folders/<id>","display_name":"Taxes 2026"},"update_mask":{"paths":["display_name"]}}'

# Move into another folder
G UpdateFolder '{"folder":{"name":"folders/<id>","parent":"folders/<other-id>"},"update_mask":{"paths":["parent"]}}'

# Delete (moves to trash)
G DeleteFolder '{"name":"folders/<id>"}'

# Health check (what Kubernetes calls)
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

> In grpcurl, write `update_mask` as `{"paths":[...]}`. The short string form (`"update_mask":"display_name"`) is rejected by grpcurl, though real clients accept it.

---

## 5. Database & migrations

Migration files live in `backend/services/metadata/migrations/`. Each change is a pair:
`NNNNNN_name.up.sql` (apply) and `NNNNNN_name.down.sql` (undo). **Never edit a migration that has already run.** Add a new one instead.

```bash
make migrate-up            # apply all pending migrations
make migrate-down          # undo the most recent migration

# Create a new migration pair (next number is picked automatically)
migrate create -ext sql -dir backend/services/metadata/migrations -seq add_files_table

# Which version is the database at?
migrate -path backend/services/metadata/migrations \
  -database "postgres://localhost:5432/drive_metadata?sslmode=disable" version

# A migration failed halfway and the DB is "dirty": fix the SQL by hand, then
# mark the version it's really at (e.g. 1)
migrate -path backend/services/metadata/migrations \
  -database "postgres://localhost:5432/drive_metadata?sslmode=disable" force 1
```

Look at the data:

```bash
psql drive_metadata                                   # interactive shell (\q to quit)
psql drive_metadata -c '\dt'                          # list tables
psql drive_metadata -c '\d folders'                   # table columns + indexes
psql drive_metadata -c 'SELECT * FROM folders'        # all rows
psql drive_metadata -c "DELETE FROM folders WHERE owner_id = 'stephano'"   # wipe your test data
```

---

## 6. Protocol Buffers (the API contract)

After editing anything in `proto/`:

```bash
make proto       # lint, then regenerate Go code into backend/gen/, then go mod tidy
make lint        # only check style
make breaking    # fail if the change would break existing clients (compares to main)
```

Never edit `backend/gen/` by hand. It's overwritten on every `make proto`.

---

## 7. Build, check, test

```bash
cd backend

go build ./...       # compile everything (fast way to check it builds)
go vet ./...         # catch common bugs
gofmt -l .           # list files that aren't formatted (no output = all good)
gofmt -w .           # format them

go test ./...                                    # run all tests
go test ./services/metadata/...                  # only the metadata service
go test -run TestCreateFolder ./services/metadata/internal/store   # one test by name
go test -v ./...                                 # verbose: show every test
go test -race ./...                              # also detect data races
```

> No tests exist yet. The store tests are the next step.

---

## Where things are

| What | Path |
|---|---|
| API definition | `proto/drive/metadata/v1/metadata.proto` |
| Metadata service entry point | `backend/services/metadata/cmd/server/main.go` |
| Metadata service file map | `backend/services/metadata/README.md` |
| Database migrations | `backend/services/metadata/migrations/` |
| Make targets | `Makefile` |
