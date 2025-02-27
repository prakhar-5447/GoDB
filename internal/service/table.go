package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/prakhar-5447/GoDB/internal/audit"
	"github.com/prakhar-5447/GoDB/internal/db"
	"github.com/prakhar-5447/GoDB/internal/pkg/proto"
)

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

func (s *DatabaseServiceServer) InsertMultipleRecords(ctx context.Context, req *proto.InsertMultipleRecordsRequest) (*proto.InsertMultipleRecordsResponse, error) {
	database, err := db.OpenDatabase(req.ConnectionString)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	// We'll loop over each record in the request and insert them.
	for _, rec := range req.Records {
		var columns []string
		var placeholders []string
		var args []interface{}

		for col, val := range rec.Data {
			columns = append(columns, col)
			placeholders = append(placeholders, "?")
			args = append(args, val)
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", req.TableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
		_, err = database.Exec(query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to insert record: %w", err)
		}
	}

	return &proto.InsertMultipleRecordsResponse{Message: "Records inserted successfully!"}, nil
}

func (s *DatabaseServiceServer) QueryData(ctx context.Context, req *proto.QueryDataRequest) (*proto.QueryDataResponse, error) {
	// Open the database using the connection string.
	database, err := db.OpenDatabase(req.ConnectionString)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	// Build the query using the provided condition directly.
	query := fmt.Sprintf("SELECT %s FROM %s", req.Columns, req.TableName)
	if req.Condition != "" {
		query += " WHERE " + req.Condition
	}

	audit.LogEvent(fmt.Sprintf("Executing query: %s", query))

	rows, err := database.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get column names.
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var response proto.QueryDataResponse
	var lastID string
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rowData := &proto.QueryRow{
			Data: make(map[string]string),
		}
		for i, col := range cols {
			val := fmt.Sprintf("%v", values[i])
			rowData.Data[col] = val
			// If the column is "id", capture its value as the last ID.
			if strings.ToLower(col) == "id" {
				lastID = val
			}
		}
		response.Rows = append(response.Rows, rowData)
	}

	// Set next_cursor if any rows were returned.
	if len(response.Rows) > 0 {
		response.NextCursor = lastID
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
