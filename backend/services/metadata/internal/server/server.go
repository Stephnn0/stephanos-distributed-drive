// Package server implements the MetadataService gRPC API.
//
// Each endpoint lives in its own file and does three things: check the
// request, call the store, and convert the result to a proto message.
// There is no SQL here; that's all in the store package.
package server

import (
	metadatav1 "github.com/Stephnn0/stephanos-distributed-drive/backend/gen/drive/metadata/v1"
	"github.com/Stephnn0/stephanos-distributed-drive/backend/services/metadata/internal/store"
)

// Server implements metadatav1.MetadataServiceServer.
type Server struct {
	// Embedding this keeps the server compiling when new RPCs are added
	// to the proto: any RPC we haven't written yet returns Unimplemented.
	metadatav1.UnimplementedMetadataServiceServer

	store *store.Store
}

// New returns a Server that keeps its data in st.
func New(st *store.Store) *Server {
	return &Server{store: st}
}
