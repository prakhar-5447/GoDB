package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/prakhar-5447/GoDB/proto/databasepb"
	"google.golang.org/grpc"
)

type DatabaseServiceServer struct {
	databasepb.UnimplementedDatabaseServiceServer
}

func (s *DatabaseServiceServer) CreateDatabase(ctx context.Context, req *databasepb.CreateDatabaseRequest) (*databasepb.CreateDatabaseResponse, error) {
	dbPath := fmt.Sprintf("%s.db", req.DatabaseName)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return &databasepb.CreateDatabaseResponse{Message: "Database created successfully!"}, nil
}

func (s *DatabaseServiceServer) CreateTable(ctx context.Context, req *databasepb.CreateTableRequest) (*databasepb.CreateTableResponse, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s.db", req.DatabaseName))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var columnDefs []string
	for colName, colType := range req.Columns {
		columnDefs = append(columnDefs, fmt.Sprintf("%s %s", colName, colType))
	}
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", req.TableName, stringJoin(columnDefs, ", "))

	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}
	return &databasepb.CreateTableResponse{Message: "Table created successfully!"}, nil
}

func (s *DatabaseServiceServer) InsertRecord(ctx context.Context, req *databasepb.InsertRecordRequest) (*databasepb.InsertRecordResponse, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s.db", req.DatabaseName))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var columns []string
	var values []string
	var args []interface{}

	for col, val := range req.Record {
		columns = append(columns, col)
		values = append(values, "?")
		args = append(args, val)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", req.TableName, stringJoin(columns, ", "), stringJoin(values, ", "))
	_, err = db.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	return &databasepb.InsertRecordResponse{Message: "Record inserted successfully!"}, nil
}

// QueryData implementation
func (s *DatabaseServiceServer) QueryData(ctx context.Context, req *databasepb.QueryDataRequest) (*databasepb.QueryDataResponse, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s.db", req.DatabaseName))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Construct the SQL query
	query := fmt.Sprintf("SELECT %s FROM %s", req.Columns, req.TableName)
	if req.Condition != "" {
		query += " WHERE " + req.Condition
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var response databasepb.QueryDataResponse

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
		rowData := &databasepb.QueryRow{
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
func (s *DatabaseServiceServer) UpdateTable(ctx context.Context, req *databasepb.UpdateTableRequest) (*databasepb.UpdateTableResponse, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s.db", req.DatabaseName))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Construct ALTER TABLE query
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", req.TableName, req.ColumnName, req.ColumnType)

	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	return &databasepb.UpdateTableResponse{Message: "Table updated successfully"}, nil
}

// UpdateRecord updates an existing record in the database.
func (s *DatabaseServiceServer) UpdateRecord(ctx context.Context, req *databasepb.UpdateRecordRequest) (*databasepb.UpdateRecordResponse, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s.db", req.DatabaseName))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Construct the UPDATE query dynamically
	setClauses := []string{}
	args := []interface{}{}

	for col, val := range req.Updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
		args = append(args, val)
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", req.TableName, strings.Join(setClauses, ", "), req.Condition)

	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)
	if err != nil {
		return nil, err
	}

	return &databasepb.UpdateRecordResponse{Message: "Record updated successfully"}, nil
}

// Add Index
func (s *DatabaseServiceServer) AddIndex(ctx context.Context, req *databasepb.AddIndexRequest) (*databasepb.AddIndexResponse, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s.db", req.DatabaseName))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Ensure each database has an `indexes` table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS indexes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT NOT NULL,
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
	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	// Store index metadata
	_, err = db.Exec("INSERT INTO indexes (user_id, table_name, index_name, columns) VALUES (?, ?, ?, ?)",
		req.UserId, req.TableName, indexName, strings.Join(req.Columns, ", "))
	if err != nil {
		return nil, err
	}

	return &databasepb.AddIndexResponse{Message: "Index created successfully!"}, nil
}

// Delete Index
func (s *DatabaseServiceServer) DeleteIndex(ctx context.Context, req *databasepb.DeleteIndexRequest) (*databasepb.DeleteIndexResponse, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s.db", req.DatabaseName))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Check if the index exists for the user
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM indexes WHERE user_id = ? AND index_name = ?", req.UserId, req.IndexName).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, fmt.Errorf("index '%s' not found", req.IndexName)
	}

	// Drop the index from the database
	_, err = db.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", req.IndexName))
	if err != nil {
		return nil, err
	}

	// Remove index metadata
	_, err = db.Exec("DELETE FROM indexes WHERE user_id = ? AND index_name = ?", req.UserId, req.IndexName)
	if err != nil {
		return nil, err
	}

	return &databasepb.DeleteIndexResponse{Message: "Index deleted successfully!"}, nil
}

// List Indexes
func (s *DatabaseServiceServer) ListIndexes(ctx context.Context, req *databasepb.ListIndexesRequest) (*databasepb.ListIndexesResponse, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s.db", req.DatabaseName))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT index_name, table_name, columns FROM indexes WHERE user_id = ?", req.UserId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []*databasepb.Index
	for rows.Next() {
		var index databasepb.Index
		err := rows.Scan(&index.IndexName, &index.TableName, &index.Columns)
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, &index)
	}

	return &databasepb.ListIndexesResponse{Indexes: indexes}, nil
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

func main() {
	// Start gRPC Server
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	databasepb.RegisterDatabaseServiceServer(grpcServer, &DatabaseServiceServer{})

	log.Println("gRPC server is running on port 50051...")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
