package assets

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// BulkOperation represents a bulk operation request
type BulkOperation struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	Type           string                 `json:"type" db:"type"`     // "import", "export", "update", "delete"
	Status         string                 `json:"status" db:"status"` // "pending", "processing", "completed", "failed"
	RequestedBy    uuid.UUID              `json:"requested_by" db:"requested_by"`
	Parameters     map[string]interface{} `json:"parameters" db:"parameters"`
	Progress       int                    `json:"progress" db:"progress"`
	TotalItems     int                    `json:"total_items" db:"total_items"`
	ProcessedItems int                    `json:"processed_items" db:"processed_items"`
	SuccessCount   int                    `json:"success_count" db:"success_count"`
	ErrorCount     int                    `json:"error_count" db:"error_count"`
	Errors         []BulkError            `json:"errors" db:"errors"`
	Results        map[string]interface{} `json:"results" db:"results"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	StartedAt      *time.Time             `json:"started_at" db:"started_at"`
	CompletedAt    *time.Time             `json:"completed_at" db:"completed_at"`
}

// BulkError represents an error during bulk operations
type BulkError struct {
	Row     int                    `json:"row"`
	Field   string                 `json:"field,omitempty"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// BulkUpdateRequest represents a bulk update request
type BulkUpdateRequest struct {
	AssetIDs []uuid.UUID            `json:"asset_ids" binding:"required"`
	Updates  map[string]interface{} `json:"updates" binding:"required"`
	Notes    *string                `json:"notes"`
}

// BulkAssignRequest represents a bulk assignment request
type BulkAssignRequest struct {
	AssetIDs       []uuid.UUID `json:"asset_ids" binding:"required"`
	AssignToUserID *uuid.UUID  `json:"assign_to_user_id"`
	Notes          *string     `json:"notes"`
}

// BulkDeleteRequest represents a bulk delete request
type BulkDeleteRequest struct {
	AssetIDs []uuid.UUID `json:"asset_ids" binding:"required"`
	Force    bool        `json:"force"`
	Notes    *string     `json:"notes"`
}

// ImportPreview represents a preview of import data
type ImportPreview struct {
	TotalRows        int                      `json:"total_rows"`
	ValidRows        int                      `json:"valid_rows"`
	ErrorRows        int                      `json:"error_rows"`
	Columns          []string                 `json:"columns"`
	SampleData       []map[string]interface{} `json:"sample_data"`
	ValidationErrors []BulkError              `json:"validation_errors"`
	Suggestions      []string                 `json:"suggestions"`
}

// ExportRequest represents an export request
type ExportRequest struct {
	Format   string                 `json:"format" binding:"required"` // "csv", "xlsx", "json"
	AssetIDs []uuid.UUID            `json:"asset_ids,omitempty"`
	Filters  AssetSearchFilters     `json:"filters,omitempty"`
	Columns  []string               `json:"columns,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// CSV Import Functions

// PreviewImport analyzes a CSV file and provides a preview
func (s *Service) PreviewImport(ctx context.Context, file multipart.File, filename string) (*ImportPreview, error) {
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	// Read header row
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	preview := &ImportPreview{
		Columns:          headers,
		SampleData:       make([]map[string]interface{}, 0),
		ValidationErrors: make([]BulkError, 0),
		Suggestions:      make([]string, 0),
	}

	// Validate headers
	requiredColumns := []string{"asset_tag", "name"}
	missingColumns := []string{}
	for _, required := range requiredColumns {
		found := false
		for _, header := range headers {
			if strings.EqualFold(header, required) {
				found = true
				break
			}
		}
		if !found {
			missingColumns = append(missingColumns, required)
		}
	}

	if len(missingColumns) > 0 {
		preview.ValidationErrors = append(preview.ValidationErrors, BulkError{
			Row:     0,
			Message: fmt.Sprintf("Missing required columns: %s", strings.Join(missingColumns, ", ")),
		})
	}

	// Read and validate sample rows
	rowNum := 1
	sampleCount := 0
	maxSamples := 5

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			preview.ValidationErrors = append(preview.ValidationErrors, BulkError{
				Row:     rowNum,
				Message: fmt.Sprintf("Failed to parse row: %v", err),
			})
			rowNum++
			continue
		}

		preview.TotalRows++

		// Convert record to map
		rowData := make(map[string]interface{})
		for i, value := range record {
			if i < len(headers) {
				rowData[headers[i]] = strings.TrimSpace(value)
			}
		}

		// Validate row data
		rowErrors := s.validateImportRow(rowData, rowNum)
		if len(rowErrors) > 0 {
			preview.ValidationErrors = append(preview.ValidationErrors, rowErrors...)
			preview.ErrorRows++
		} else {
			preview.ValidRows++
		}

		// Add to sample data
		if sampleCount < maxSamples {
			preview.SampleData = append(preview.SampleData, rowData)
			sampleCount++
		}

		rowNum++
	}

	// Generate suggestions
	preview.Suggestions = s.generateImportSuggestions(preview)

	return preview, nil
}

// ImportAssets imports assets from CSV data
func (s *Service) ImportAssets(ctx context.Context, file multipart.File, requestedBy uuid.UUID, options map[string]interface{}) (*BulkOperation, error) {
	operation := &BulkOperation{
		ID:          uuid.New(),
		Type:        "import",
		Status:      "processing",
		RequestedBy: requestedBy,
		Parameters:  options,
		CreatedAt:   time.Now(),
		StartedAt:   &[]time.Time{time.Now()}[0],
		Errors:      make([]BulkError, 0),
		Results:     make(map[string]interface{}),
	}

	// Save initial operation record
	err := s.saveBulkOperation(ctx, operation)
	if err != nil {
		return nil, fmt.Errorf("failed to save bulk operation: %w", err)
	}

	// Process import asynchronously
	go s.processImport(context.Background(), operation.ID, file)

	return operation, nil
}

func (s *Service) processImport(ctx context.Context, operationID uuid.UUID, file multipart.File) {
	// Seek back to beginning of file
	file.Seek(0, 0)

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	// Read headers
	headers, err := reader.Read()
	if err != nil {
		s.markOperationFailed(ctx, operationID, fmt.Sprintf("Failed to read headers: %v", err))
		return
	}

	operation, err := s.getBulkOperation(ctx, operationID)
	if err != nil {
		return
	}

	rowNum := 1
	successCount := 0
	errorCount := 0
	errors := make([]BulkError, 0)

	// Start transaction for batch processing
	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.markOperationFailed(ctx, operationID, fmt.Sprintf("Failed to begin transaction: %v", err))
		return
	}
	defer tx.Rollback(ctx)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, BulkError{
				Row:     rowNum,
				Message: fmt.Sprintf("Failed to parse row: %v", err),
			})
			errorCount++
			rowNum++
			continue
		}

		// Convert to map
		rowData := make(map[string]interface{})
		for i, value := range record {
			if i < len(headers) {
				rowData[headers[i]] = strings.TrimSpace(value)
			}
		}

		// Validate and create asset
		asset, rowErrors := s.createAssetFromImportRow(ctx, tx, rowData, operation.RequestedBy)
		if len(rowErrors) > 0 {
			errors = append(errors, rowErrors...)
			errorCount++
		} else if asset != nil {
			successCount++
		}

		// Update progress periodically
		if rowNum%10 == 0 {
			operation.ProcessedItems = rowNum
			operation.SuccessCount = successCount
			operation.ErrorCount = errorCount
			operation.Progress = int((float64(rowNum) / float64(operation.TotalItems)) * 100)
			s.updateBulkOperation(ctx, operation)
		}

		rowNum++
	}

	// Commit transaction if we have more successes than errors
	if successCount > errorCount || errorCount == 0 {
		err = tx.Commit(ctx)
		if err != nil {
			s.markOperationFailed(ctx, operationID, fmt.Sprintf("Failed to commit transaction: %v", err))
			return
		}
		operation.Status = "completed"
	} else {
		tx.Rollback(ctx)
		operation.Status = "failed"
	}

	// Final update
	now := time.Now()
	operation.CompletedAt = &now
	operation.ProcessedItems = rowNum - 1
	operation.SuccessCount = successCount
	operation.ErrorCount = errorCount
	operation.Errors = errors
	operation.Progress = 100
	operation.Results = map[string]interface{}{
		"imported_assets": successCount,
		"failed_rows":     errorCount,
		"total_rows":      rowNum - 1,
	}

	s.updateBulkOperation(ctx, operation)
}

// Bulk Update Functions

// BulkUpdateAssets updates multiple assets
func (s *Service) BulkUpdateAssets(ctx context.Context, req BulkUpdateRequest, updatedBy uuid.UUID) (*BulkOperation, error) {
	operation := &BulkOperation{
		ID:          uuid.New(),
		Type:        "update",
		Status:      "processing",
		RequestedBy: updatedBy,
		Parameters:  map[string]interface{}{"updates": req.Updates, "notes": req.Notes},
		TotalItems:  len(req.AssetIDs),
		CreatedAt:   time.Now(),
		StartedAt:   &[]time.Time{time.Now()}[0],
		Errors:      make([]BulkError, 0),
		Results:     make(map[string]interface{}),
	}

	// Save operation
	err := s.saveBulkOperation(ctx, operation)
	if err != nil {
		return nil, fmt.Errorf("failed to save bulk operation: %w", err)
	}

	// Process updates
	successCount := 0
	errorCount := 0
	errors := make([]BulkError, 0)

	for i, assetID := range req.AssetIDs {
		// Get current asset
		currentAsset, err := s.GetAsset(ctx, assetID)
		if err != nil {
			errors = append(errors, BulkError{
				Row:     i + 1,
				Message: fmt.Sprintf("Asset not found: %v", err),
				Data:    map[string]interface{}{"asset_id": assetID},
			})
			errorCount++
			continue
		}

		// Build update request
		updateReq := s.buildUpdateRequestFromMap(req.Updates)

		// Update asset
		_, err = s.UpdateAsset(ctx, assetID, updateReq, updatedBy)
		if err != nil {
			errors = append(errors, BulkError{
				Row:     i + 1,
				Message: fmt.Sprintf("Failed to update asset: %v", err),
				Data:    map[string]interface{}{"asset_id": assetID, "asset_tag": currentAsset.AssetTag},
			})
			errorCount++
		} else {
			successCount++
		}

		// Update progress
		operation.ProcessedItems = i + 1
		operation.Progress = int((float64(i+1) / float64(len(req.AssetIDs))) * 100)
	}

	// Complete operation
	now := time.Now()
	operation.CompletedAt = &now
	operation.Status = "completed"
	operation.SuccessCount = successCount
	operation.ErrorCount = errorCount
	operation.Errors = errors
	operation.Results = map[string]interface{}{
		"updated_assets": successCount,
		"failed_updates": errorCount,
		"total_assets":   len(req.AssetIDs),
	}

	err = s.updateBulkOperation(ctx, operation)
	if err != nil {
		return nil, fmt.Errorf("failed to update bulk operation: %w", err)
	}

	return operation, nil
}

// BulkAssignAssets assigns multiple assets
func (s *Service) BulkAssignAssets(ctx context.Context, req BulkAssignRequest, assignedBy uuid.UUID) (*BulkOperation, error) {
	operation := &BulkOperation{
		ID:          uuid.New(),
		Type:        "assign",
		Status:      "processing",
		RequestedBy: assignedBy,
		Parameters:  map[string]interface{}{"assign_to_user_id": req.AssignToUserID, "notes": req.Notes},
		TotalItems:  len(req.AssetIDs),
		CreatedAt:   time.Now(),
		StartedAt:   &[]time.Time{time.Now()}[0],
		Errors:      make([]BulkError, 0),
		Results:     make(map[string]interface{}),
	}

	err := s.saveBulkOperation(ctx, operation)
	if err != nil {
		return nil, fmt.Errorf("failed to save bulk operation: %w", err)
	}

	successCount := 0
	errorCount := 0
	errors := make([]BulkError, 0)

	for i, assetID := range req.AssetIDs {
		assignReq := AssignAssetRequest{
			AssignedToUserID: req.AssignToUserID,
			Notes:            req.Notes,
		}

		err := s.AssignAsset(ctx, assetID, assignReq, assignedBy)
		if err != nil {
			errors = append(errors, BulkError{
				Row:     i + 1,
				Message: fmt.Sprintf("Failed to assign asset: %v", err),
				Data:    map[string]interface{}{"asset_id": assetID},
			})
			errorCount++
		} else {
			successCount++
		}

		operation.ProcessedItems = i + 1
		operation.Progress = int((float64(i+1) / float64(len(req.AssetIDs))) * 100)
	}

	now := time.Now()
	operation.CompletedAt = &now
	operation.Status = "completed"
	operation.SuccessCount = successCount
	operation.ErrorCount = errorCount
	operation.Errors = errors
	operation.Results = map[string]interface{}{
		"assigned_assets":    successCount,
		"failed_assignments": errorCount,
		"total_assets":       len(req.AssetIDs),
	}

	err = s.updateBulkOperation(ctx, operation)
	return operation, err
}

// Export Functions

// ExportAssets exports assets to various formats
func (s *Service) ExportAssets(ctx context.Context, req ExportRequest, requestedBy uuid.UUID) (*BulkOperation, error) {
	operation := &BulkOperation{
		ID:          uuid.New(),
		Type:        "export",
		Status:      "processing",
		RequestedBy: requestedBy,
		Parameters:  map[string]interface{}{"format": req.Format, "columns": req.Columns, "options": req.Options},
		CreatedAt:   time.Now(),
		StartedAt:   &[]time.Time{time.Now()}[0],
		Results:     make(map[string]interface{}),
	}

	err := s.saveBulkOperation(ctx, operation)
	if err != nil {
		return nil, fmt.Errorf("failed to save bulk operation: %w", err)
	}

	// Process export asynchronously
	go s.processExport(context.Background(), operation.ID, req)

	return operation, nil
}

func (s *Service) processExport(ctx context.Context, operationID uuid.UUID, req ExportRequest) {
	operation, err := s.getBulkOperation(ctx, operationID)
	if err != nil {
		return
	}

	// Get assets to export
	var assets []Asset
	if len(req.AssetIDs) > 0 {
		// Export specific assets
		for _, assetID := range req.AssetIDs {
			asset, err := s.GetAsset(ctx, assetID)
			if err == nil {
				assets = append(assets, *asset)
			}
		}
	} else {
		// Export based on filters
		result, err := s.ListAssets(ctx, req.Filters)
		if err != nil {
			s.markOperationFailed(ctx, operationID, fmt.Sprintf("Failed to get assets: %v", err))
			return
		}
		assets = result.Assets
	}

	operation.TotalItems = len(assets)

	var exportData []byte
	var filename string
	var contentType string

	switch strings.ToLower(req.Format) {
	case "csv":
		exportData, err = s.exportToCSV(assets, req.Columns)
		filename = fmt.Sprintf("assets_export_%s.csv", time.Now().Format("20060102_150405"))
		contentType = "text/csv"
	case "json":
		exportData, err = s.exportToJSON(assets, req.Columns)
		filename = fmt.Sprintf("assets_export_%s.json", time.Now().Format("20060102_150405"))
		contentType = "application/json"
	default:
		err = fmt.Errorf("unsupported export format: %s", req.Format)
	}

	if err != nil {
		s.markOperationFailed(ctx, operationID, fmt.Sprintf("Export failed: %v", err))
		return
	}

	// In a real implementation, you would save the file to object storage
	// For now, we'll store basic results
	now := time.Now()
	operation.CompletedAt = &now
	operation.Status = "completed"
	operation.ProcessedItems = len(assets)
	operation.SuccessCount = len(assets)
	operation.Progress = 100
	operation.Results = map[string]interface{}{
		"exported_count": len(assets),
		"filename":       filename,
		"content_type":   contentType,
		"file_size":      len(exportData),
	}

	s.updateBulkOperation(ctx, operation)
}

// Helper Functions

func (s *Service) validateImportRow(rowData map[string]interface{}, rowNum int) []BulkError {
	var errors []BulkError

	// Validate required fields
	if assetTag, ok := rowData["asset_tag"].(string); !ok || assetTag == "" {
		errors = append(errors, BulkError{
			Row:     rowNum,
			Field:   "asset_tag",
			Message: "Asset tag is required",
		})
	}

	if name, ok := rowData["name"].(string); !ok || name == "" {
		errors = append(errors, BulkError{
			Row:     rowNum,
			Field:   "name",
			Message: "Name is required",
		})
	}

	// Validate status if provided
	if status, ok := rowData["status"].(string); ok && status != "" {
		validStatuses := []string{"active", "inactive", "maintenance", "retired", "disposed"}
		valid := false
		for _, validStatus := range validStatuses {
			if strings.EqualFold(status, validStatus) {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, BulkError{
				Row:     rowNum,
				Field:   "status",
				Message: fmt.Sprintf("Invalid status. Must be one of: %s", strings.Join(validStatuses, ", ")),
			})
		}
	}

	return errors
}

func (s *Service) createAssetFromImportRow(ctx context.Context, _ pgx.Tx, rowData map[string]interface{}, createdBy uuid.UUID) (*Asset, []BulkError) {
	var errors []BulkError

	// Map row data to create request
	req := CreateAssetRequest{
		AssetTag: rowData["asset_tag"].(string),
		Name:     rowData["name"].(string),
	}

	// Set optional fields
	if description, ok := rowData["description"].(string); ok && description != "" {
		req.Description = &description
	}

	if status, ok := rowData["status"].(string); ok && status != "" {
		assetStatus := AssetStatus(strings.ToLower(status))
		req.Status = assetStatus
	}

	if serialNumber, ok := rowData["serial_number"].(string); ok && serialNumber != "" {
		req.SerialNumber = &serialNumber
	}

	if model, ok := rowData["model"].(string); ok && model != "" {
		req.Model = &model
	}

	if manufacturer, ok := rowData["manufacturer"].(string); ok && manufacturer != "" {
		req.Manufacturer = &manufacturer
	}

	if location, ok := rowData["location"].(string); ok && location != "" {
		req.Location = &location
	}

	// Parse price if provided
	if priceStr, ok := rowData["purchase_price"].(string); ok && priceStr != "" {
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			req.PurchasePrice = &price
		}
	}

	// Create asset (simplified version using tx)
	asset, err := s.CreateAsset(ctx, req, createdBy)
	if err != nil {
		errors = append(errors, BulkError{
			Message: fmt.Sprintf("Failed to create asset: %v", err),
			Data:    rowData,
		})
		return nil, errors
	}

	return asset, errors
}

func (s *Service) generateImportSuggestions(preview *ImportPreview) []string {
	var suggestions []string

	if preview.ErrorRows > 0 {
		suggestions = append(suggestions, "Review validation errors before importing")
	}

	if preview.ValidRows == 0 {
		suggestions = append(suggestions, "No valid rows found - check data format and required columns")
	}

	// Column mapping suggestions
	commonMappings := map[string][]string{
		"asset_tag":     {"tag", "asset_number", "id"},
		"name":          {"asset_name", "title", "description"},
		"serial_number": {"serial", "sn", "serial_no"},
		"model":         {"model_number", "model_name"},
		"manufacturer":  {"make", "brand", "vendor"},
	}

	for targetCol, variations := range commonMappings {
		found := false
		for _, header := range preview.Columns {
			if strings.EqualFold(header, targetCol) {
				found = true
				break
			}
		}
		if !found {
			for _, variation := range variations {
				for _, header := range preview.Columns {
					if strings.EqualFold(header, variation) {
						suggestions = append(suggestions, fmt.Sprintf("Consider renaming column '%s' to '%s'", header, targetCol))
						break
					}
				}
			}
		}
	}

	return suggestions
}

func (s *Service) buildUpdateRequestFromMap(updates map[string]interface{}) UpdateAssetRequest {
	req := UpdateAssetRequest{}

	if name, ok := updates["name"].(string); ok {
		req.Name = &name
	}
	if description, ok := updates["description"].(string); ok {
		req.Description = &description
	}
	if statusStr, ok := updates["status"].(string); ok {
		status := AssetStatus(statusStr)
		req.Status = &status
	}
	if conditionStr, ok := updates["condition"].(string); ok {
		condition := AssetCondition(conditionStr)
		req.Condition = &condition
	}
	if location, ok := updates["location"].(string); ok {
		req.Location = &location
	}
	if manufacturer, ok := updates["manufacturer"].(string); ok {
		req.Manufacturer = &manufacturer
	}
	if model, ok := updates["model"].(string); ok {
		req.Model = &model
	}
	if serialNumber, ok := updates["serial_number"].(string); ok {
		req.SerialNumber = &serialNumber
	}

	return req
}

func (s *Service) exportToCSV(assets []Asset, columns []string) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// Default columns if none specified
	if len(columns) == 0 {
		columns = []string{"asset_tag", "name", "status", "category", "manufacturer", "model", "serial_number", "location", "assigned_to", "created_at"}
	}

	// Write header
	writer.Write(columns)

	// Write data
	for _, asset := range assets {
		record := make([]string, len(columns))
		for i, col := range columns {
			record[i] = s.getAssetFieldValue(asset, col)
		}
		writer.Write(record)
	}

	writer.Flush()
	return []byte(buf.String()), writer.Error()
}

func (s *Service) exportToJSON(assets []Asset, columns []string) ([]byte, error) {
	if len(columns) == 0 {
		// Export all fields
		return json.MarshalIndent(assets, "", "  ")
	}

	// Export only specified columns
	var filteredAssets []map[string]interface{}
	for _, asset := range assets {
		filtered := make(map[string]interface{})
		for _, col := range columns {
			filtered[col] = s.getAssetFieldValue(asset, col)
		}
		filteredAssets = append(filteredAssets, filtered)
	}

	return json.MarshalIndent(filteredAssets, "", "  ")
}

func (s *Service) getAssetFieldValue(asset Asset, fieldName string) string {
	switch fieldName {
	case "asset_tag":
		return asset.AssetTag
	case "name":
		return asset.Name
	case "status":
		return string(asset.Status)
	case "condition":
		if asset.Condition != nil {
			return string(*asset.Condition)
		}
	case "manufacturer":
		if asset.Manufacturer != nil {
			return *asset.Manufacturer
		}
	case "model":
		if asset.Model != nil {
			return *asset.Model
		}
	case "serial_number":
		if asset.SerialNumber != nil {
			return *asset.SerialNumber
		}
	case "location":
		if asset.Location != nil {
			return *asset.Location
		}
	case "category":
		if asset.Category != nil {
			return asset.Category.Name
		}
	case "assigned_to":
		if asset.AssignedUser != nil {
			return asset.AssignedUser.Email
		}
	case "created_at":
		return asset.CreatedAt.Format("2006-01-02 15:04:05")
	case "updated_at":
		return asset.UpdatedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// Database operations for bulk operations
func (s *Service) saveBulkOperation(ctx context.Context, operation *BulkOperation) error {
	parametersJSON, _ := json.Marshal(operation.Parameters)
	errorsJSON, _ := json.Marshal(operation.Errors)
	resultsJSON, _ := json.Marshal(operation.Results)

	query := `
		INSERT INTO asset_bulk_operations (
			id, type, status, requested_by, parameters, progress, total_items,
			processed_items, success_count, error_count, errors, results, created_at, started_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err := s.db.Exec(ctx, query,
		operation.ID, operation.Type, operation.Status, operation.RequestedBy,
		parametersJSON, operation.Progress, operation.TotalItems,
		operation.ProcessedItems, operation.SuccessCount, operation.ErrorCount,
		errorsJSON, resultsJSON, operation.CreatedAt, operation.StartedAt, operation.CompletedAt)

	return err
}

func (s *Service) updateBulkOperation(ctx context.Context, operation *BulkOperation) error {
	parametersJSON, _ := json.Marshal(operation.Parameters)
	errorsJSON, _ := json.Marshal(operation.Errors)
	resultsJSON, _ := json.Marshal(operation.Results)

	query := `
		UPDATE asset_bulk_operations 
		SET status = $1, parameters = $2, progress = $3, processed_items = $4,
		    success_count = $5, error_count = $6, errors = $7, results = $8, completed_at = $9
		WHERE id = $10`

	_, err := s.db.Exec(ctx, query,
		operation.Status, parametersJSON, operation.Progress, operation.ProcessedItems,
		operation.SuccessCount, operation.ErrorCount, errorsJSON, resultsJSON,
		operation.CompletedAt, operation.ID)

	return err
}

func (s *Service) getBulkOperation(ctx context.Context, operationID uuid.UUID) (*BulkOperation, error) {
	operation := &BulkOperation{}
	var parametersJSON, errorsJSON, resultsJSON []byte

	query := `
		SELECT id, type, status, requested_by, parameters, progress, total_items,
		       processed_items, success_count, error_count, errors, results,
		       created_at, started_at, completed_at
		FROM asset_bulk_operations WHERE id = $1`

	err := s.db.QueryRow(ctx, query, operationID).Scan(
		&operation.ID, &operation.Type, &operation.Status, &operation.RequestedBy,
		&parametersJSON, &operation.Progress, &operation.TotalItems,
		&operation.ProcessedItems, &operation.SuccessCount, &operation.ErrorCount,
		&errorsJSON, &resultsJSON, &operation.CreatedAt, &operation.StartedAt, &operation.CompletedAt)

	if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	if len(parametersJSON) > 0 {
		_ = json.Unmarshal(parametersJSON, &operation.Parameters)
	}
	if len(errorsJSON) > 0 {
		_ = json.Unmarshal(errorsJSON, &operation.Errors)
	}
	if len(resultsJSON) > 0 {
		_ = json.Unmarshal(resultsJSON, &operation.Results)
	}

	return operation, nil
}

func (s *Service) markOperationFailed(ctx context.Context, operationID uuid.UUID, errorMessage string) {
	now := time.Now()
	_, _ = s.db.Exec(ctx, `
		UPDATE asset_bulk_operations 
		SET status = 'failed', completed_at = $1, results = $2
		WHERE id = $3`,
		now, fmt.Sprintf(`{"error": "%s"}`, errorMessage), operationID)
}
