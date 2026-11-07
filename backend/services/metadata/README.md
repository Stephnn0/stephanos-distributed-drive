# metadata service

Source of truth for what exists in the drive: folders (and later files), who owns them, and where they live.
API: [`proto/drive/metadata/v1/metadata.proto`](../../../proto/drive/metadata/v1/metadata.proto).

## Where to look

```
metadata/
├── cmd/server/main.go        ENTRY POINT: config → connect DB → start gRPC → graceful stop
├── migrations/               SQL that creates the tables (run with `make migrate-up`)
└── internal/
    ├── server/               API LAYER: gRPC endpoints, no SQL
    │   ├── server.go         Server struct + New()
    │   ├── create_folder.go  one file per endpoint:
    │   ├── get_folder.go       1. check the request
    │   ├── list_folders.go     2. call the store
    │   ├── update_folder.go    3. convert the result to proto
    │   ├── delete_folder.go
    │   ├── convert.go        store.Folder → proto, "folders/{id}" names, validation
    │   ├── errors.go         store errors → gRPC codes
    │   └── auth.go           who is calling (x-user-id header, until real auth)
    └── store/                DATABASE LAYER: all SQL, no gRPC
        ├── store.go          Store struct + Connect() (connection pool)
        ├── folder.go         Folder struct + the errors the store returns
        └── folder_queries.go one method per query
```

A request flows: `main.go` → `server/<endpoint>.go` → `store/folder_queries.go` → Postgres.

Generated code from the proto lives in `backend/gen/`. Never edit it; run `make proto` instead.

## Run it

```bash
make migrate-up                                   # from the repo root, once
cd backend && go run ./services/metadata/cmd/server
```

Environment: `DATABASE_URL` (default: local `drive_metadata` database).
