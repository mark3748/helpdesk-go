package assets

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service provides asset management operations
type Service struct {
	db *pgxpool.Pool
}

// NewService creates a new asset service
func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// CreateCategory creates a new asset category
func (s *Service) CreateCategory(ctx context.Context, req CreateCategoryRequest) (*AssetCategory, error) {
	category := &AssetCategory{
		ID:           uuid.New(),
		Name:         req.Name,
		Description:  req.Description,
		ParentID:     req.ParentID,
		CustomFields: req.CustomFields,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if category.CustomFields == nil {
		category.CustomFields = make(map[string]interface{})
	}

	customFieldsJSON, _ := json.Marshal(category.CustomFields)

	query := `
		INSERT INTO asset_categories (id, name, description, parent_id, custom_fields, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	err := s.db.QueryRow(ctx, query, category.ID, category.Name, category.Description,
		category.ParentID, customFieldsJSON, category.CreatedAt, category.UpdatedAt).
		Scan(&category.ID, &category.CreatedAt, &category.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return category, nil
}

// GetCategory retrieves a category by ID
func (s *Service) GetCategory(ctx context.Context, id uuid.UUID) (*AssetCategory, error) {
	category := &AssetCategory{}
	var customFieldsJSON []byte

	query := `
		SELECT id, name, description, parent_id, custom_fields, created_at, updated_at
		FROM asset_categories WHERE id = $1`

	err := s.db.QueryRow(ctx, query, id).Scan(
		&category.ID, &category.Name, &category.Description, &category.ParentID,
		&customFieldsJSON, &category.CreatedAt, &category.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("category not found")
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	if len(customFieldsJSON) > 0 {
		_ = json.Unmarshal(customFieldsJSON, &category.CustomFields)
	}

	return category, nil
}

// ListCategories retrieves all asset categories
func (s *Service) ListCategories(ctx context.Context) ([]AssetCategory, error) {
	query := `
		SELECT id, name, description, parent_id, custom_fields, created_at, updated_at
		FROM asset_categories ORDER BY name`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	defer rows.Close()

	var categories []AssetCategory
	for rows.Next() {
		var category AssetCategory
		var customFieldsJSON []byte

		err := rows.Scan(&category.ID, &category.Name, &category.Description,
			&category.ParentID, &customFieldsJSON, &category.CreatedAt, &category.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		if len(customFieldsJSON) > 0 {
			_ = json.Unmarshal(customFieldsJSON, &category.CustomFields)
		}

		categories = append(categories, category)
	}

	return categories, nil
}

// CreateAsset creates a new asset
func (s *Service) CreateAsset(ctx context.Context, req CreateAssetRequest, createdBy uuid.UUID) (*Asset, error) {
	asset := &Asset{
		ID:               uuid.New(),
		AssetTag:         req.AssetTag,
		Name:             req.Name,
		Description:      req.Description,
		CategoryID:       req.CategoryID,
		Status:           req.Status,
		Condition:        req.Condition,
		PurchasePrice:    req.PurchasePrice,
		PurchaseDate:     req.PurchaseDate,
		WarrantyExpiry:   req.WarrantyExpiry,
		DepreciationRate: req.DepreciationRate,
		SerialNumber:     req.SerialNumber,
		Model:            req.Model,
		Manufacturer:     req.Manufacturer,
		Location:         req.Location,
		CustomFields:     req.CustomFields,
		CreatedBy:        createdBy,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if asset.Status == "" {
		asset.Status = AssetStatusActive
	}
	if asset.CustomFields == nil {
		asset.CustomFields = make(map[string]interface{})
	}

	customFieldsJSON, _ := json.Marshal(asset.CustomFields)

	query := `
		INSERT INTO assets (
			id, asset_tag, name, description, category_id, status, condition,
			purchase_price, purchase_date, warranty_expiry, depreciation_rate,
			serial_number, model, manufacturer, location, custom_fields,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		) RETURNING id, created_at, updated_at`

	err := s.db.QueryRow(ctx, query,
		asset.ID, asset.AssetTag, asset.Name, asset.Description, asset.CategoryID,
		asset.Status, asset.Condition, asset.PurchasePrice, asset.PurchaseDate,
		asset.WarrantyExpiry, asset.DepreciationRate, asset.SerialNumber,
		asset.Model, asset.Manufacturer, asset.Location, customFieldsJSON,
		asset.CreatedBy, asset.CreatedAt, asset.UpdatedAt).
		Scan(&asset.ID, &asset.CreatedAt, &asset.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create asset: %w", err)
	}

	// Record history
	_ = s.recordHistory(ctx, asset.ID, ActionCreated, &createdBy, nil, map[string]interface{}{
		"asset_tag": asset.AssetTag,
		"name":      asset.Name,
		"status":    asset.Status,
	}, nil)

	return asset, nil
}

// GetAsset retrieves an asset by ID with optional joins
func (s *Service) GetAsset(ctx context.Context, id uuid.UUID) (*Asset, error) {
	query := `
		SELECT 
			a.id, a.asset_tag, a.name, a.description, a.category_id, a.status, a.condition,
			a.purchase_price, a.purchase_date, a.warranty_expiry, a.depreciation_rate, a.current_value,
			a.serial_number, a.model, a.manufacturer, a.location, a.assigned_to_user_id, a.assigned_at,
			a.custom_fields, a.created_by, a.created_at, a.updated_at,
			c.id, c.name, c.description,
			u.id, u.email, u.display_name,
			cb.id, cb.email, cb.display_name
		FROM assets a
		LEFT JOIN asset_categories c ON a.category_id = c.id
		LEFT JOIN users u ON a.assigned_to_user_id = u.id
		LEFT JOIN users cb ON a.created_by = cb.id
		WHERE a.id = $1`

	row := s.db.QueryRow(ctx, query, id)
	asset, err := s.scanAssetWithJoins(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("asset not found")
		}
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	return asset, nil
}

// UpdateAsset updates an existing asset
func (s *Service) UpdateAsset(ctx context.Context, id uuid.UUID, req UpdateAssetRequest, updatedBy uuid.UUID) (*Asset, error) {
	// Get current asset for comparison
	currentAsset, err := s.GetAsset(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build update query dynamically
	setParts := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	oldValues := make(map[string]interface{})
	newValues := make(map[string]interface{})

	if req.Name != nil && *req.Name != currentAsset.Name {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
		oldValues["name"] = currentAsset.Name
		newValues["name"] = *req.Name
	}

	if req.Description != nil {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, req.Description)
		argIndex++
		oldValues["description"] = currentAsset.Description
		newValues["description"] = req.Description
	}

	if req.CategoryID != nil {
		setParts = append(setParts, fmt.Sprintf("category_id = $%d", argIndex))
		args = append(args, req.CategoryID)
		argIndex++
		oldValues["category_id"] = currentAsset.CategoryID
		newValues["category_id"] = req.CategoryID
	}

	if req.Status != nil && *req.Status != currentAsset.Status {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *req.Status)
		argIndex++
		oldValues["status"] = currentAsset.Status
		newValues["status"] = *req.Status
	}

	if req.Condition != nil {
		setParts = append(setParts, fmt.Sprintf("condition = $%d", argIndex))
		args = append(args, req.Condition)
		argIndex++
		oldValues["condition"] = currentAsset.Condition
		newValues["condition"] = req.Condition
	}

	// Add other fields...
	if req.PurchasePrice != nil {
		setParts = append(setParts, fmt.Sprintf("purchase_price = $%d", argIndex))
		args = append(args, req.PurchasePrice)
		argIndex++
	}

	if req.CurrentValue != nil {
		setParts = append(setParts, fmt.Sprintf("current_value = $%d", argIndex))
		args = append(args, req.CurrentValue)
		argIndex++
	}

	if req.SerialNumber != nil {
		setParts = append(setParts, fmt.Sprintf("serial_number = $%d", argIndex))
		args = append(args, req.SerialNumber)
		argIndex++
	}

	if req.Model != nil {
		setParts = append(setParts, fmt.Sprintf("model = $%d", argIndex))
		args = append(args, req.Model)
		argIndex++
	}

	if req.Manufacturer != nil {
		setParts = append(setParts, fmt.Sprintf("manufacturer = $%d", argIndex))
		args = append(args, req.Manufacturer)
		argIndex++
	}

	if req.Location != nil {
		setParts = append(setParts, fmt.Sprintf("location = $%d", argIndex))
		args = append(args, req.Location)
		argIndex++
	}

	if req.CustomFields != nil {
		customFieldsJSON, _ := json.Marshal(req.CustomFields)
		setParts = append(setParts, fmt.Sprintf("custom_fields = $%d", argIndex))
		args = append(args, customFieldsJSON)
		argIndex++
	}

	if len(setParts) == 1 { // Only updated_at
		return currentAsset, nil
	}

	query := fmt.Sprintf("UPDATE assets SET %s WHERE id = $%d", strings.Join(setParts, ", "), argIndex)
	args = append(args, id)

	_, err = s.db.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update asset: %w", err)
	}

	// Record history if there were changes
	if len(oldValues) > 0 {
		action := ActionUpdated
		if req.Status != nil && *req.Status != currentAsset.Status {
			action = ActionStatusChanged
		}
		_ = s.recordHistory(ctx, id, action, &updatedBy, oldValues, newValues, nil)
	}

	return s.GetAsset(ctx, id)
}

// ListAssets retrieves assets with filtering and pagination
func (s *Service) ListAssets(ctx context.Context, filters AssetSearchFilters) (*AssetListResponse, error) {
	// Build WHERE clause
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if filters.Query != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("to_tsvector('english', coalesce(a.name,'') || ' ' || coalesce(a.description,'') || ' ' || coalesce(a.asset_tag,'') || ' ' || coalesce(a.serial_number,'') || ' ' || coalesce(a.model,'') || ' ' || coalesce(a.manufacturer,'')) @@ plainto_tsquery('english', $%d)", argIndex))
		args = append(args, filters.Query)
		argIndex++
	}

	if filters.CategoryID != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("a.category_id = $%d", argIndex))
		args = append(args, *filters.CategoryID)
		argIndex++
	}

	if len(filters.Status) > 0 {
		placeholders := make([]string, len(filters.Status))
		for i, status := range filters.Status {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, status)
			argIndex++
		}
		whereConditions = append(whereConditions, fmt.Sprintf("a.status IN (%s)", strings.Join(placeholders, ",")))
	}

	if filters.Condition != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("a.condition = $%d", argIndex))
		args = append(args, *filters.Condition)
		argIndex++
	}

	if filters.AssignedTo != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("a.assigned_to_user_id = $%d", argIndex))
		args = append(args, *filters.AssignedTo)
		argIndex++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Build ORDER BY clause
	sortBy := "a.created_at"
	if filters.SortBy != "" {
		switch filters.SortBy {
		case "name", "asset_tag", "status", "created_at", "updated_at":
			sortBy = "a." + filters.SortBy
		}
	}

	sortOrder := "DESC"
	if filters.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	// Set pagination defaults
	if filters.Limit <= 0 || filters.Limit > 100 {
		filters.Limit = 20
	}
	if filters.Page < 1 {
		filters.Page = 1
	}

	offset := (filters.Page - 1) * filters.Limit

	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM assets a 
		LEFT JOIN asset_categories c ON a.category_id = c.id 
		%s`, whereClause)

	var total int
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count assets: %w", err)
	}

	// Get assets
	query := fmt.Sprintf(`
		SELECT 
			a.id, a.asset_tag, a.name, a.description, a.category_id, a.status, a.condition,
			a.purchase_price, a.purchase_date, a.warranty_expiry, a.depreciation_rate, a.current_value,
			a.serial_number, a.model, a.manufacturer, a.location, a.assigned_to_user_id, a.assigned_at,
			a.custom_fields, a.created_by, a.created_at, a.updated_at,
			c.id, c.name, c.description,
			u.id, u.email, u.display_name,
			cb.id, cb.email, cb.display_name
		FROM assets a
		LEFT JOIN asset_categories c ON a.category_id = c.id
		LEFT JOIN users u ON a.assigned_to_user_id = u.id
		LEFT JOIN users cb ON a.created_by = cb.id
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, sortBy, sortOrder, argIndex, argIndex+1)

	args = append(args, filters.Limit, offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		asset, err := s.scanAssetWithJoins(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan asset: %w", err)
		}
		assets = append(assets, *asset)
	}

	pages := int(math.Ceil(float64(total) / float64(filters.Limit)))

	return &AssetListResponse{
		Assets: assets,
		Total:  total,
		Page:   filters.Page,
		Limit:  filters.Limit,
		Pages:  pages,
	}, nil
}

// AssignAsset assigns an asset to a user
func (s *Service) AssignAsset(ctx context.Context, assetID uuid.UUID, req AssignAssetRequest, assignedBy uuid.UUID) error {
	// Start transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get current assignment
	var currentUserID *uuid.UUID
	err = tx.QueryRow(ctx, "SELECT assigned_to_user_id FROM assets WHERE id = $1", assetID).Scan(&currentUserID)
	if err != nil {
		return fmt.Errorf("failed to get current assignment: %w", err)
	}

	// End current assignment if exists
	if currentUserID != nil {
		_, err = tx.Exec(ctx, `
			UPDATE asset_assignments 
			SET status = 'completed', unassigned_at = NOW() 
			WHERE asset_id = $1 AND status = 'active'`, assetID)
		if err != nil {
			return fmt.Errorf("failed to complete current assignment: %w", err)
		}
	}

	now := time.Now()

	// Update asset assignment
	_, err = tx.Exec(ctx, `
		UPDATE assets 
		SET assigned_to_user_id = $1, assigned_at = $2, updated_at = NOW() 
		WHERE id = $3`, req.AssignedToUserID, &now, assetID)
	if err != nil {
		return fmt.Errorf("failed to update asset assignment: %w", err)
	}

	// Create assignment record if assigning to someone
	if req.AssignedToUserID != nil {
		_, err = tx.Exec(ctx, `
			INSERT INTO asset_assignments (asset_id, assigned_to_user_id, assigned_by_user_id, notes, assigned_at)
			VALUES ($1, $2, $3, $4, $5)`,
			assetID, req.AssignedToUserID, assignedBy, req.Notes, now)
		if err != nil {
			return fmt.Errorf("failed to create assignment record: %w", err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Record history
	action := ActionAssigned
	if req.AssignedToUserID == nil {
		action = ActionUnassigned
	}

	_ = s.recordHistory(ctx, assetID, action, &assignedBy, 
		map[string]interface{}{"assigned_to": currentUserID},
		map[string]interface{}{"assigned_to": req.AssignedToUserID}, 
		req.Notes)

	return nil
}

// DeleteAsset soft deletes an asset
func (s *Service) DeleteAsset(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	// For now, we'll set status to disposed instead of hard delete
	_, err := s.db.Exec(ctx, `
		UPDATE assets 
		SET status = 'disposed', updated_at = NOW() 
		WHERE id = $1`, id)
	
	if err != nil {
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	_ = s.recordHistory(ctx, id, ActionDisposed, &deletedBy, nil, 
		map[string]interface{}{"status": "disposed"}, nil)

	return nil
}

// recordHistory records an action in the asset history
func (s *Service) recordHistory(ctx context.Context, assetID uuid.UUID, action AssetHistoryAction, actorID *uuid.UUID, oldValues, newValues map[string]interface{}, notes *string) error {
	oldJSON, _ := json.Marshal(oldValues)
	newJSON, _ := json.Marshal(newValues)

	_, err := s.db.Exec(ctx, `
		INSERT INTO asset_history (asset_id, action, actor_id, old_values, new_values, notes)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		assetID, action, actorID, oldJSON, newJSON, notes)

	return err
}

// scanAssetWithJoins scans an asset with joined tables
func (s *Service) scanAssetWithJoins(row pgx.Row) (*Asset, error) {
	asset := &Asset{}
	var customFieldsJSON []byte
	var category AssetCategory
	var assignedUser AssetUser
	var createdByUser AssetUser

	var categoryID, assignedUserID, createdByUserID sql.NullString
	var categoryName, categoryDescription sql.NullString
	var assignedUserEmail, assignedUserDisplayName sql.NullString
	var createdByEmail, createdByDisplayName sql.NullString

	err := row.Scan(
		&asset.ID, &asset.AssetTag, &asset.Name, &asset.Description, &asset.CategoryID,
		&asset.Status, &asset.Condition, &asset.PurchasePrice, &asset.PurchaseDate,
		&asset.WarrantyExpiry, &asset.DepreciationRate, &asset.CurrentValue,
		&asset.SerialNumber, &asset.Model, &asset.Manufacturer, &asset.Location,
		&asset.AssignedToUserID, &asset.AssignedAt, &customFieldsJSON,
		&asset.CreatedBy, &asset.CreatedAt, &asset.UpdatedAt,
		&categoryID, &categoryName, &categoryDescription,
		&assignedUserID, &assignedUserEmail, &assignedUserDisplayName,
		&createdByUserID, &createdByEmail, &createdByDisplayName,
	)

	if err != nil {
		return nil, err
	}

	// Unmarshal custom fields
	if len(customFieldsJSON) > 0 {
		_ = json.Unmarshal(customFieldsJSON, &asset.CustomFields)
	}

	// Set category if exists
	if categoryID.Valid {
		categoryUUID, _ := uuid.Parse(categoryID.String)
		category.ID = categoryUUID
		category.Name = categoryName.String
		if categoryDescription.Valid {
			category.Description = &categoryDescription.String
		}
		asset.Category = &category
	}

	// Set assigned user if exists
	if assignedUserID.Valid {
		assignedUUID, _ := uuid.Parse(assignedUserID.String)
		assignedUser.ID = assignedUUID
		assignedUser.Email = assignedUserEmail.String
		if assignedUserDisplayName.Valid {
			assignedUser.DisplayName = &assignedUserDisplayName.String
		}
		asset.AssignedUser = &assignedUser
	}

	// Set created by user if exists
	if createdByUserID.Valid {
		createdByUUID, _ := uuid.Parse(createdByUserID.String)
		createdByUser.ID = createdByUUID
		createdByUser.Email = createdByEmail.String
		if createdByDisplayName.Valid {
			createdByUser.DisplayName = &createdByDisplayName.String
		}
		asset.CreatedByUser = &createdByUser
	}

	return asset, nil
}