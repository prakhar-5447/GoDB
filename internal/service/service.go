package service

import (
	"context"
	"fmt"

	"github.com/prakhar-5447/GoDB/internal/audit"
	"github.com/prakhar-5447/GoDB/internal/auth"
	"github.com/prakhar-5447/GoDB/internal/db"
	"github.com/prakhar-5447/GoDB/internal/pkg/proto"
	"google.golang.org/grpc"
)

type DatabaseServiceServer struct {
	proto.UnimplementedDatabaseServiceServer
}

func RegisterGRPCServices(grpcServer *grpc.Server) {
	proto.RegisterDatabaseServiceServer(grpcServer, &DatabaseServiceServer{})
}

// CreateUser registers a new user and returns a connection string.
func (s *DatabaseServiceServer) CreateUser(ctx context.Context, req *proto.CreateUserRequest) (*proto.CreateUserResponse, error) {
	// Create the user in the auth database.
	err := auth.CreateUser(req.Username, req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Optionally, you might also create a default database for the user at this point.
	// For now, we'll just generate a connection string.
	// Format: grpc://username:password/{databaseName}
	// Here we use a placeholder "<dbname>" that the user can later replace with a real database name.
	connectionString := fmt.Sprintf("grpc://%s:%s/<dbname>", req.Username, req.Password)
	audit.LogEvent(fmt.Sprintf("User %s created successfully. Connection string: %s", req.Username, connectionString))

	return &proto.CreateUserResponse{
		Message:          "User created successfully",
		ConnectionString: connectionString,
	}, nil
}

func (s *DatabaseServiceServer) CreateDatabase(ctx context.Context, req *proto.CreateDatabaseRequest) (*proto.CreateDatabaseResponse, error) {
	// Get database path
	database, err := db.OpenDatabase(req.ConnectionString)

	if err != nil {
		return nil, err
	}
	defer database.Close()

	// Run migrations
	if err := db.RunMigrations(database); err != nil {
		return nil, err
	}

	return &proto.CreateDatabaseResponse{Message: "Database created successfully!"}, nil
}
