package service 
import (
	"context"
	"fmt"
	"strings"

	"github.com/prakhar-5447/GoDB/internal/db"
	"github.com/prakhar-5447/GoDB/internal/pkg/proto"
)
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
	_, err = database.Exec("DELETE FROM indexes WHEREindex_name = ?", req.IndexName)
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