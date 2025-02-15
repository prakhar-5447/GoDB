package server

import (
	"context"
	"fmt"
	"log"
	"strings"

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
	log.Printf("User %s created successfully. Connection string: %s", req.Username, connectionString)

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

func (s *DatabaseServiceServer) CreateTable(ctx context.Context, req *proto.CreateTableRequest) (*proto.CreateTableResponse, error) {
	database, err := db.OpenDatabase(req.ConnectionString)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	var columnDefs []string
	for colName, colType := range req.Columns {
		columnDefs = append(columnDefs, fmt.Sprintf("%s %s", colName, colType))
	}
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", req.TableName, stringJoin(columnDefs, ", "))

	_, err = database.Exec(query)
	if err != nil {
		return nil, err
	}
	return &proto.CreateTableResponse{Message: "Table created successfully!"}, nil
}

func (s *DatabaseServiceServer) InsertRecord(ctx context.Context, req *proto.InsertRecordRequest) (*proto.InsertRecordResponse, error) {
	database, err := db.OpenDatabase(req.ConnectionString)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	var columns []string
	var values []string
	var args []interface{}

	for col, val := range req.Record {
		columns = append(columns, col)
		values = append(values, "?")
		args = append(args, val)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", req.TableName, stringJoin(columns, ", "), stringJoin(values, ", "))
	_, err = database.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	return &proto.InsertRecordResponse{Message: "Record inserted successfully!"}, nil
}

// QueryData implementation
func (s *DatabaseServiceServer) QueryData(ctx context.Context, req *proto.QueryDataRequest) (*proto.QueryDataResponse, error) {
	database, err := db.OpenDatabase(req.ConnectionString)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	// Construct the SQL query
	query := fmt.Sprintf("SELECT %s FROM %s", req.Columns, req.TableName)
	if req.Condition != "" {
		query += " WHERE " + req.Condition
	}

	rows, err := database.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var response proto.QueryDataResponse

	// Iterate over the rows and fetch data
	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}

		// Convert values to string map
		rowData := &proto.QueryRow{
			Data: make(map[string]string),
		}
		for i, colName := range columns {
			rowData.Data[colName] = fmt.Sprintf("%v", values[i])
		}

		// Append to response
		response.Rows = append(response.Rows, rowData)
	}

	return &response, nil
}

// UpdateTable updates the structure of an existing table.
func (s *DatabaseServiceServer) UpdateTable(ctx context.Context, req *proto.UpdateTableRequest) (*proto.UpdateTableResponse, error) {
	database, err := db.OpenDatabase(req.ConnectionString)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	// Construct ALTER TABLE query
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", req.TableName, req.ColumnName, req.ColumnType)

	_, err = database.Exec(query)
	if err != nil {
		return nil, err
	}

	return &proto.UpdateTableResponse{Message: "Table updated successfully"}, nil
}

// UpdateRecord updates an existing record in the database.
func (s *DatabaseServiceServer) UpdateRecord(ctx context.Context, req *proto.UpdateRecordRequest) (*proto.UpdateRecordResponse, error) {
	database, err := db.OpenDatabase(req.ConnectionString)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	// Construct the UPDATE query dynamically
	setClauses := []string{}
	args := []interface{}{}

	for col, val := range req.Updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
		args = append(args, val)
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", req.TableName, strings.Join(setClauses, ", "), req.Condition)

	stmt, err := database.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)
	if err != nil {
		return nil, err
	}

	return &proto.UpdateRecordResponse{Message: "Record updated successfully"}, nil
}

// Add Index
func (s *DatabaseServiceServer) AddIndex(ctx context.Context, req *proto.AddIndexRequest) (*proto.AddIndexResponse, error) {
	database, err := db.OpenDatabase(req.ConnectionString)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	// Ensure each database has an `indexes` table
	_, err = database.Exec(`CREATE TABLE IF NOT EXISTS indexes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		table_name TEXT NOT NULL,
		index_name TEXT NOT NULL UNIQUE,
		columns TEXT NOT NULL,
		UNIQUE(user_id, index_name)
	)`)
	if err != nil {
		return nil, err
	}

	// Generate index name if not provided
	indexName := req.IndexName
	if indexName == "" {
		indexName = fmt.Sprintf("%s_%s_idx", req.TableName, strings.Join(req.Columns, "_"))
	}

	// Create index in the table
	query := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)", indexName, req.TableName, strings.Join(req.Columns, ", "))
	_, err = database.Exec(query)
	if err != nil {
		return nil, err
	}

	// Store index metadata
	_, err = database.Exec("INSERT INTO indexes (table_name, index_name, columns) VALUES (?, ?, ?)",
		req.TableName, indexName, strings.Join(req.Columns, ", "))
	if err != nil {
		return nil, err
	}

	return &proto.AddIndexResponse{Message: "Index created successfully!"}, nil
}

// Delete Index
func (s *DatabaseServiceServer) DeleteIndex(ctx context.Context, req *proto.DeleteIndexRequest) (*proto.DeleteIndexResponse, error) {
	database, err := db.OpenDatabase(req.ConnectionString)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	// Check if the index exists for the user
	var count int
	err = database.QueryRow("SELECT COUNT(*) FROM indexes WHERE index_name = ?", req.IndexName).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, fmt.Errorf("index '%s' not found", req.IndexName)
	}

	// Drop the index from the database
	_, err = database.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", req.IndexName))
	if err != nil {
		return nil, err
	}

	// Remove index metadata
	_, err = database.Exec("DELETE FROM indexes WHEREindex_name = ?",req.IndexName)
	if err != nil {
		return nil, err
	}

	return &proto.DeleteIndexResponse{Message: "Index deleted successfully!"}, nil
}

// List Indexes
func (s *DatabaseServiceServer) ListIndexes(ctx context.Context, req *proto.ListIndexesRequest) (*proto.ListIndexesResponse, error) {
	database, err := db.OpenDatabase(req.ConnectionString)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	rows, err := database.Query("SELECT index_name, table_name, columns FROM indexes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []*proto.Index
	for rows.Next() {
		var index proto.Index
		err := rows.Scan(&index.IndexName, &index.TableName, &index.Columns)
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, &index)
	}

	return &proto.ListIndexesResponse{Indexes: indexes}, nil
}

func stringJoin(elements []string, separator string) string {
	result := ""
	for i, elem := range elements {
		if i > 0 {
			result += separator
		}
		result += elem
	}
	return result
}
