# Stephanos Distributed Drive

A distributed cloud storage system inspired by Google Drive, built with **Go**, **gRPC**, **Protocol Buffers**, **Kubernetes**, and **Terraform**. Features scalable file storage, metadata management, synchronization, and file sharing.

## Repository Structure

This project is a **monorepo**: every part of the system lives in one repository, with a single shared API contract.

```
stephanos-distributed-drive/
├── proto/      # .proto files, the API contract shared by backend + client
├── backend/    # Go microservices
├── client/     # Frontend application
└── infra/      # Terraform + Kubernetes
```

### `proto/`
The **source of truth** for every API in the system. All gRPC services and messages are defined here as Protocol Buffer (`.proto`) files. Code for the backend (Go) and the client is **generated** from these definitions, so both sides always agree on the contract.

It lives at the root (not inside `backend/`) because more than one part of the system consumes it.

### `backend/`
The Go microservices that make up the core of the system. Each service owns one domain, exposes a gRPC API defined in `proto/`, and scales independently on Kubernetes.

See [MVP Services](#mvp-services) below.

```
backend/
├── go.mod                  # one Go module for all services
└── services/
    ├── metadata/
    │   ├── cmd/server/     # entry point (main.go)
    │   └── internal/       # private to this service
    ├── storage/
    │   ├── cmd/server/
    │   └── internal/
    └── sync/
        ├── cmd/server/
        └── internal/
```

Each service has its own entry point under `cmd/server/` and builds to its own binary and container image. Code under `internal/` cannot be imported by other services, so the Go compiler enforces service boundaries.

Run a service locally:
```bash
cd backend
go run ./services/metadata/cmd/server
```

### `client/`
The frontend application users interact with. It talks to the backend through the API gateway.

### `infra/`
Infrastructure as code, split into two layers that change at different speeds:

- **Terraform** creates the **platform**: GKE cluster, networking, Cloud SQL (Postgres), storage buckets, API Gateway, Artifact Registry, and IAM. Changes rarely.
- **Kubernetes** manifests describe the **apps** running on that platform: Deployments, Services, autoscaling, and config for each service. Changes on every deploy.

```
infra/
├── terraform/
│   ├── modules/          # reusable building blocks (gke/, cloudsql/, buckets/, ...)
│   └── envs/
│       ├── dev/          # dev environment: small cluster
│       └── prod/         # prod environment: larger, more replicas
└── k8s/
    ├── base/             # one folder per service, shared by all environments
    │   ├── metadata/     #   deployment.yaml, service.yaml, hpa.yaml
    │   ├── storage/
    │   └── sync/
    └── overlays/         # per-environment changes (Kustomize)
        ├── dev/          #   e.g. 1 replica, debug logging
        └── prod/         #   e.g. 3+ replicas, more CPU
```

Kubernetes config uses **Kustomize** (built into `kubectl`): `base/` holds the shared manifests and each overlay changes only what differs for its environment.

## Deployment

Pushing to GitHub does not change anything in the cloud by itself. A CI/CD pipeline applies changes, and nobody applies them by hand from a laptop.

### Infrastructure changes (Terraform)
1. Open a pull request that changes `infra/terraform/`
2. CI (GitHub Actions / Cloud Build) runs `terraform plan` and posts the result on the PR
3. The plan is reviewed like code
4. On merge to `main`, CI runs `terraform apply`
5. Terraform state is stored in a GCS bucket, so CI and developers share one view of what exists

### App changes (Go services)
1. Merge code to `main`
2. CI runs tests, builds a container image, and pushes it to Artifact Registry
3. The new image is deployed to GKE, either by CI running `kubectl apply`, or with **GitOps** (Argo CD / Config Sync), where a tool in the cluster keeps it in sync with `infra/k8s/` in Git
4. Kubernetes performs a rolling update: new pods pass health checks before old pods shut down gracefully

### Roadmap
1. Run everything locally on Kubernetes with `kind`, using `infra/k8s/`
2. Write Terraform for a dev environment on Google Cloud
3. Automate both with GitHub Actions

## Generating Code from Protos

The `.proto` files in `proto/` are compiled into Go code under `backend/gen/` with [buf](https://buf.build). Generated code is never edited by hand.

One-time setup (macOS):
```bash
brew install go bufbuild/buf/buf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH="$PATH:$(go env GOPATH)/bin"   # add to ~/.zshrc
```

Then, from the repo root:
```bash
make proto      # lint + generate + go mod tidy
make lint       # lint only
make breaking   # check for breaking API changes against main
```

## MVP Services

The MVP is three Go microservices, each owning one domain and communicating over gRPC.

### 1. metadata
The source of truth for **what exists**: files, folders, ownership, and permissions. Backed by **PostgreSQL**.
- Create, list, get, and delete files and folders
- Share files (read/write access) and check permissions on every call
- Publishes change events (created, renamed, moved, deleted) to **Kafka**

### 2. storage
Handles **the bytes**, and knows nothing about folders or users. Backed by **object storage** (GCS / MinIO).
- `Upload`: client-streaming gRPC, so large files are sent in chunks
- `Download`: server-streaming gRPC

### 3. sync
Keeps every device **up to date**.
- Consumes change events from Kafka and maintains a per-user change log
- `WatchChanges(cursor)`: server-streaming gRPC that sends clients only what changed since their last cursor
- Resolves conflicts when two devices edit the same file (MVP: last write wins, conflict copy kept)

### Upload flow
1. Client calls **metadata** to create the file and receives a file ID
2. Client streams the bytes to **storage** under that ID
3. **metadata** marks the file as ready and publishes a `FileCreated` event
4. **sync** picks up the event and pushes the change to the user's other devices

### Out of scope for the MVP
- **Auth**: handled by Firebase Auth / Identity Platform; services verify tokens in a shared interceptor
- **API gateway**: Google Cloud API Gateway / GKE Gateway, configured in `infra/`
- **Later services**: sharing, search, preview/thumbnails, notifications, quotas

## Tech Stack

| Layer          | Technology                          |
|----------------|-------------------------------------|
| Services       | Go                                  |
| API            | gRPC + Protocol Buffers             |
| Gateway        | Google Cloud API Gateway / GKE Gateway |
| Metadata DB    | PostgreSQL                          |
| File storage   | Object storage (GCS / MinIO)        |
| Cache          | Redis                               |
| Events         | Kafka                               |
| Orchestration  | Kubernetes (GKE)                    |
| Provisioning   | Terraform                           |
