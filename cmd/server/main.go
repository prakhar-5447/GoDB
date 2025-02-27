package main

import (
	"log"
	"net"

	"github.com/prakhar-5447/GoDB/internal/audit"
	"github.com/prakhar-5447/GoDB/internal/auth"
	"github.com/prakhar-5447/GoDB/internal/db"
	"github.com/prakhar-5447/GoDB/internal/service"

	"google.golang.org/grpc"
)

func main() {
	audit.InitAuditLogger()

	// Ensure the authentication database is initialized.
	if err := auth.InitAuthDatabase(); err != nil {
		log.Fatalf("Auth DB initialization failed: %v", err)
	}

	// Ensure the data directory exists
	if err := db.EnsureDBDirectory(); err != nil {
		log.Fatalf("failed to create database directory: %v", err)
	}

	// Start gRPC server
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	service.RegisterGRPCServices(grpcServer)

	log.Println("🚀 gRPC server running on port 50051...")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}
}
