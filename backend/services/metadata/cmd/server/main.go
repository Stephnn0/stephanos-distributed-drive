// Command server is the ENTRY POINT of the metadata service.
//
// It only wires things together: read config, connect to the database,
// start the gRPC server, and stop cleanly. The real logic lives in
// internal/server (API endpoints) and internal/store (database).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	metadatav1 "github.com/Stephnn0/stephanos-distributed-drive/backend/gen/drive/metadata/v1"
	"github.com/Stephnn0/stephanos-distributed-drive/backend/services/metadata/internal/server"
	"github.com/Stephnn0/stephanos-distributed-drive/backend/services/metadata/internal/store"
)

func main() {
	port := flag.Int("port", 50051, "gRPC port")
	flag.Parse()

	// ctx is cancelled on Ctrl+C or Kubernetes SIGTERM. Set up first, so an
	// early signal is never missed.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Connect to Postgres.
	st, err := store.Connect(ctx, databaseURL())
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	// 2. Open the port.
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	// 3. Create the gRPC server and register endpoints (like http.HandleFunc).
	s := grpc.NewServer()
	metadatav1.RegisterMetadataServiceServer(s, server.New(st))
	healthpb.RegisterHealthServer(s, health.NewServer()) // Kubernetes health checks
	reflection.Register(s)                               // lets grpcurl discover the API

	// 4. When a stop signal arrives, finish in-flight requests, then stop.
	go func() {
		<-ctx.Done()
		log.Println("shutting down")
		s.GracefulStop()
	}()

	// 5. Serve until stopped.
	log.Printf("metadata service listening on %s", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// databaseURL reads DATABASE_URL, falling back to the local dev database.
// In Kubernetes it comes from a Secret.
func databaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return "postgres://localhost:5432/drive_metadata?sslmode=disable"
}
