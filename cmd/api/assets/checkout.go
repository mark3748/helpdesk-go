package assets

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CheckoutRequest represents a request to checkout an asset
type CheckoutRequest struct {
	AssetID             uuid.UUID       `json:"asset_id" binding:"required"`
	CheckedOutToUserID  uuid.UUID       `json:"checked_out_to_user_id" binding:"required"`
	ExpectedReturnDate  *time.Time      `json:"expected_return_date"`
	CheckoutNotes       *string         `json:"checkout_notes"`
	ConditionAtCheckout *AssetCondition `json:"condition_at_checkout"`
	RequiresApproval    bool            `json:"requires_approval"`
}

// CheckinRequest represents a request to checkin an asset
type CheckinRequest struct {
	CheckoutID        uuid.UUID       `json:"checkout_id" binding:"required"`
	ReturnNotes       *string         `json:"return_notes"`
	ConditionAtReturn *AssetCondition `json:"condition_at_return"`
	MaintenanceNeeded bool            `json:"maintenance_needed"`
}

// AssetCheckout represents a checkout record
type AssetCheckout struct {
	ID                  uuid.UUID       `json:"id" db:"id"`
	AssetID             uuid.UUID       `json:"asset_id" db:"asset_id"`
	CheckedOutToUserID  uuid.UUID       `json:"checked_out_to_user_id" db:"checked_out_to_user_id"`
	CheckedOutByUserID  uuid.UUID       `json:"checked_out_by_user_id" db:"checked_out_by_user_id"`
	CheckedOutAt        time.Time       `json:"checked_out_at" db:"checked_out_at"`
	ExpectedReturnDate  *time.Time      `json:"expected_return_date" db:"expected_return_date"`
	ActualReturnDate    *time.Time      `json:"actual_return_date" db:"actual_return_date"`
	CheckoutNotes       *string         `json:"checkout_notes" db:"checkout_notes"`
	ReturnNotes         *string         `json:"return_notes" db:"return_notes"`
	ConditionAtCheckout *AssetCondition `json:"condition_at_checkout" db:"condition_at_checkout"`
	ConditionAtReturn   *AssetCondition `json:"condition_at_return" db:"condition_at_return"`
	Status              string          `json:"status" db:"status"`

	// Joined fields
	Asset            *Asset     `json:"asset,omitempty"`
	CheckedOutToUser *AssetUser `json:"checked_out_to_user,omitempty"`
	CheckedOutByUser *AssetUser `json:"checked_out_by_user,omitempty"`
}

// AssetAlert represents an alert for asset management
type AssetAlert struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	AssetID        uuid.UUID              `json:"asset_id" db:"asset_id"`
	AlertType      string                 `json:"alert_type" db:"alert_type"`
	Severity       string                 `json:"severity" db:"severity"`
	Title          string                 `json:"title" db:"title"`
	Description    *string                `json:"description" db:"description"`
	TriggerDate    time.Time              `json:"trigger_date" db:"trigger_date"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at" db:"acknowledged_at"`
	AcknowledgedBy *uuid.UUID             `json:"acknowledged_by" db:"acknowledged_by"`
	ResolvedAt     *time.Time             `json:"resolved_at" db:"resolved_at"`
	ResolvedBy     *uuid.UUID             `json:"resolved_by" db:"resolved_by"`
	IsActive       bool                   `json:"is_active" db:"is_active"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`

	// Joined fields
	Asset *Asset `json:"asset,omitempty"`
}

// CheckoutAsset handles the asset checkout process
func (s *Service) CheckoutAsset(ctx context.Context, req CheckoutRequest, checkedOutBy uuid.UUID) (*AssetCheckout, error) {
	// Start transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if asset exists and is available
	var currentStatus AssetStatus
	var currentlyCheckedOut bool

	err = tx.QueryRow(ctx, `
		SELECT a.status, 
		       EXISTS(SELECT 1 FROM asset_checkouts ac WHERE ac.asset_id = a.id AND ac.status = 'active')
		FROM assets a WHERE a.id = $1`, req.AssetID).
		Scan(&currentStatus, &currentlyCheckedOut)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("asset not found")
		}
		return nil, fmt.Errorf("failed to check asset availability: %w", err)
	}

	if currentlyCheckedOut {
		return nil, fmt.Errorf("asset is already checked out")
	}

	if currentStatus != AssetStatusActive {
		return nil, fmt.Errorf("asset is not available for checkout (status: %s)", currentStatus)
	}

	// Check if approval is required for this checkout
	if req.RequiresApproval {
		return s.createCheckoutWorkflow(ctx, req, checkedOutBy)
	}

	// Create checkout record
	checkout := &AssetCheckout{
		ID:                  uuid.New(),
		AssetID:             req.AssetID,
		CheckedOutToUserID:  req.CheckedOutToUserID,
		CheckedOutByUserID:  checkedOutBy,
		CheckedOutAt:        time.Now(),
		ExpectedReturnDate:  req.ExpectedReturnDate,
		CheckoutNotes:       req.CheckoutNotes,
		ConditionAtCheckout: req.ConditionAtCheckout,
		Status:              "active",
	}

	// Insert checkout record
	err = tx.QueryRow(ctx, `
		INSERT INTO asset_checkouts (
			id, asset_id, checked_out_to_user_id, checked_out_by_user_id,
			checked_out_at, expected_return_date, checkout_notes, condition_at_checkout, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, checked_out_at`,
		checkout.ID, checkout.AssetID, checkout.CheckedOutToUserID, checkout.CheckedOutByUserID,
		checkout.CheckedOutAt, checkout.ExpectedReturnDate, checkout.CheckoutNotes,
		checkout.ConditionAtCheckout, checkout.Status).
		Scan(&checkout.ID, &checkout.CheckedOutAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create checkout record: %w", err)
	}

	// Update asset assignment
	_, err = tx.Exec(ctx, `
		UPDATE assets 
		SET assigned_to_user_id = $1, assigned_at = $2, updated_at = NOW()
		WHERE id = $3`,
		req.CheckedOutToUserID, checkout.CheckedOutAt, req.AssetID)

	if err != nil {
		return nil, fmt.Errorf("failed to update asset assignment: %w", err)
	}

	// Record in asset history
	historyData := map[string]interface{}{
		"action":                "checkout",
		"checked_out_to":        req.CheckedOutToUserID,
		"expected_return":       req.ExpectedReturnDate,
		"condition_at_checkout": req.ConditionAtCheckout,
	}
	historyJSON, _ := json.Marshal(historyData)

	_, err = tx.Exec(ctx, `
		INSERT INTO asset_history (asset_id, action, actor_id, new_values, notes)
		VALUES ($1, 'checkout', $2, $3, $4)`,
		req.AssetID, checkedOutBy, historyJSON, req.CheckoutNotes)

	if err != nil {
		return nil, fmt.Errorf("failed to record history: %w", err)
	}

	// Set up return reminder if expected return date is set
	if req.ExpectedReturnDate != nil {
		_ = s.scheduleReturnReminder(ctx, checkout.ID, *req.ExpectedReturnDate)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return checkout, nil
}

// CheckinAsset handles the asset checkin process
func (s *Service) CheckinAsset(ctx context.Context, req CheckinRequest, checkedInBy uuid.UUID) error {
	// Start transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get checkout record
	var checkout AssetCheckout
	var conditionAtCheckout *string
	var conditionAtReturn *string

	if req.ConditionAtReturn != nil {
		condStr := string(*req.ConditionAtReturn)
		conditionAtReturn = &condStr
	}

	err = tx.QueryRow(ctx, `
		SELECT id, asset_id, checked_out_to_user_id, condition_at_checkout
		FROM asset_checkouts 
		WHERE id = $1 AND status = 'active'`, req.CheckoutID).
		Scan(&checkout.ID, &checkout.AssetID, &checkout.CheckedOutToUserID, &conditionAtCheckout)

	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("checkout record not found or already returned")
		}
		return fmt.Errorf("failed to get checkout record: %w", err)
	}

	now := time.Now()

	// Update checkout record
	_, err = tx.Exec(ctx, `
		UPDATE asset_checkouts 
		SET actual_return_date = $1, return_notes = $2, condition_at_return = $3, status = 'returned'
		WHERE id = $4`,
		now, req.ReturnNotes, conditionAtReturn, req.CheckoutID)

	if err != nil {
		return fmt.Errorf("failed to update checkout record: %w", err)
	}

	// Update asset assignment (unassign)
	_, err = tx.Exec(ctx, `
		UPDATE assets 
		SET assigned_to_user_id = NULL, assigned_at = NULL, updated_at = NOW()
		WHERE id = $1`,
		checkout.AssetID)

	if err != nil {
		return fmt.Errorf("failed to update asset assignment: %w", err)
	}

	// Update asset condition if provided
	if req.ConditionAtReturn != nil {
		_, err = tx.Exec(ctx, `
			UPDATE assets 
			SET condition = $1, updated_at = NOW()
			WHERE id = $2`,
			*req.ConditionAtReturn, checkout.AssetID)

		if err != nil {
			return fmt.Errorf("failed to update asset condition: %w", err)
		}
	}

	// Record in asset history
	historyData := map[string]interface{}{
		"action":              "checkin",
		"returned_by":         checkedInBy,
		"condition_at_return": req.ConditionAtReturn,
		"maintenance_needed":  req.MaintenanceNeeded,
	}
	historyJSON, _ := json.Marshal(historyData)

	_, err = tx.Exec(ctx, `
		INSERT INTO asset_history (asset_id, action, actor_id, new_values, notes)
		VALUES ($1, 'checkin', $2, $3, $4)`,
		checkout.AssetID, checkedInBy, historyJSON, req.ReturnNotes)

	if err != nil {
		return fmt.Errorf("failed to record history: %w", err)
	}

	// If maintenance is needed, create maintenance workflow
	if req.MaintenanceNeeded {
		_ = s.createMaintenanceWorkflow(ctx, checkout.AssetID, checkedInBy, "Post-checkout maintenance required")
	}

	// Check for condition degradation and create alerts
	if conditionAtCheckout != nil && req.ConditionAtReturn != nil {
		checkoutCond := AssetCondition(*conditionAtCheckout)
		if s.isConditionDegraded(&checkoutCond, req.ConditionAtReturn) {
			_ = s.createConditionDegradationAlert(ctx, checkout.AssetID, &checkoutCond, req.ConditionAtReturn)
		}
	}

	return tx.Commit(ctx)
}

// GetActiveCheckouts returns all active checkouts
func (s *Service) GetActiveCheckouts(ctx context.Context, filters map[string]interface{}) ([]AssetCheckout, error) {
	query := `
		SELECT 
			ac.id, ac.asset_id, ac.checked_out_to_user_id, ac.checked_out_by_user_id,
			ac.checked_out_at, ac.expected_return_date, ac.actual_return_date,
			ac.checkout_notes, ac.return_notes, ac.condition_at_checkout, ac.condition_at_return, ac.status,
			a.asset_tag, a.name as asset_name,
			uto.email as checked_out_to_email, uto.display_name as checked_out_to_name,
			uby.email as checked_out_by_email, uby.display_name as checked_out_by_name
		FROM asset_checkouts ac
		JOIN assets a ON ac.asset_id = a.id
		LEFT JOIN users uto ON ac.checked_out_to_user_id = uto.id
		LEFT JOIN users uby ON ac.checked_out_by_user_id = uby.id
		WHERE ac.status = 'active'
		ORDER BY ac.checked_out_at DESC`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active checkouts: %w", err)
	}
	defer rows.Close()

	var checkouts []AssetCheckout
	for rows.Next() {
		var checkout AssetCheckout
		var asset Asset
		var checkedOutToUser, checkedOutByUser AssetUser

		var checkedOutToEmail, checkedOutToName, checkedOutByEmail, checkedOutByName *string

		err := rows.Scan(
			&checkout.ID, &checkout.AssetID, &checkout.CheckedOutToUserID, &checkout.CheckedOutByUserID,
			&checkout.CheckedOutAt, &checkout.ExpectedReturnDate, &checkout.ActualReturnDate,
			&checkout.CheckoutNotes, &checkout.ReturnNotes, &checkout.ConditionAtCheckout, &checkout.ConditionAtReturn, &checkout.Status,
			&asset.AssetTag, &asset.Name,
			&checkedOutToEmail, &checkedOutToName,
			&checkedOutByEmail, &checkedOutByName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan checkout: %w", err)
		}

		// Set asset info
		asset.ID = checkout.AssetID
		checkout.Asset = &asset

		// Set user info
		if checkedOutToEmail != nil {
			checkedOutToUser.ID = checkout.CheckedOutToUserID
			checkedOutToUser.Email = *checkedOutToEmail
			checkedOutToUser.DisplayName = checkedOutToName
			checkout.CheckedOutToUser = &checkedOutToUser
		}

		if checkedOutByEmail != nil {
			checkedOutByUser.ID = checkout.CheckedOutByUserID
			checkedOutByUser.Email = *checkedOutByEmail
			checkedOutByUser.DisplayName = checkedOutByName
			checkout.CheckedOutByUser = &checkedOutByUser
		}

		checkouts = append(checkouts, checkout)
	}

	return checkouts, nil
}

// GetOverdueCheckouts returns checkouts that are past their expected return date
func (s *Service) GetOverdueCheckouts(ctx context.Context) ([]AssetCheckout, error) {
	query := `
		SELECT 
			ac.id, ac.asset_id, ac.checked_out_to_user_id, ac.expected_return_date,
			a.asset_tag, a.name as asset_name,
			u.email, u.display_name
		FROM asset_checkouts ac
		JOIN assets a ON ac.asset_id = a.id
		JOIN users u ON ac.checked_out_to_user_id = u.id
		WHERE ac.status = 'active' 
		AND ac.expected_return_date < NOW()
		ORDER BY ac.expected_return_date ASC`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query overdue checkouts: %w", err)
	}
	defer rows.Close()

	var checkouts []AssetCheckout
	for rows.Next() {
		var checkout AssetCheckout
		var asset Asset
		var user AssetUser
		var email, displayName *string

		err := rows.Scan(
			&checkout.ID, &checkout.AssetID, &checkout.CheckedOutToUserID, &checkout.ExpectedReturnDate,
			&asset.AssetTag, &asset.Name,
			&email, &displayName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan overdue checkout: %w", err)
		}

		asset.ID = checkout.AssetID
		checkout.Asset = &asset

		if email != nil {
			user.ID = checkout.CheckedOutToUserID
			user.Email = *email
			user.DisplayName = displayName
			checkout.CheckedOutToUser = &user
		}

		checkouts = append(checkouts, checkout)
	}

	return checkouts, nil
}

// Helper methods
func (s *Service) createCheckoutWorkflow(ctx context.Context, req CheckoutRequest, checkedOutBy uuid.UUID) (*AssetCheckout, error) {
	// Create workflow for approval
	workflow := &AssetWorkflow{
		ID:          uuid.New(),
		AssetID:     req.AssetID,
		Type:        "checkout",
		Status:      "pending",
		RequestedBy: checkedOutBy,
		RequestData: map[string]interface{}{
			"checked_out_to_user_id": req.CheckedOutToUserID,
			"expected_return_date":   req.ExpectedReturnDate,
			"checkout_notes":         req.CheckoutNotes,
			"condition_at_checkout":  req.ConditionAtCheckout,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.createWorkflow(ctx, workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout workflow: %w", err)
	}

	return &AssetCheckout{
		ID:                  workflow.ID,
		AssetID:             req.AssetID,
		CheckedOutToUserID:  req.CheckedOutToUserID,
		CheckedOutByUserID:  checkedOutBy,
		CheckedOutAt:        workflow.CreatedAt,
		ExpectedReturnDate:  req.ExpectedReturnDate,
		CheckoutNotes:       req.CheckoutNotes,
		ConditionAtCheckout: req.ConditionAtCheckout,
		Status:              "pending_approval",
	}, nil
}

func (s *Service) scheduleReturnReminder(ctx context.Context, checkoutID uuid.UUID, expectedReturnDate time.Time) error {
	// Create reminder alert one day before expected return
	reminderDate := expectedReturnDate.AddDate(0, 0, -1)

	if reminderDate.After(time.Now()) {
		alert := &AssetAlert{
			ID:          uuid.New(),
			AlertType:   "checkout_reminder",
			Severity:    "medium",
			Title:       "Asset Return Reminder",
			Description: &[]string{"Asset checkout is due for return soon"}[0],
			TriggerDate: reminderDate,
			IsActive:    true,
			Metadata: map[string]interface{}{
				"checkout_id":          checkoutID,
				"expected_return_date": expectedReturnDate,
			},
			CreatedAt: time.Now(),
		}

		return s.createAlert(ctx, alert)
	}

	return nil
}

func (s *Service) createMaintenanceWorkflow(ctx context.Context, assetID uuid.UUID, requestedBy uuid.UUID, notes string) error {
	workflow := &AssetWorkflow{
		ID:          uuid.New(),
		AssetID:     assetID,
		Type:        "maintenance",
		Status:      "pending",
		RequestedBy: requestedBy,
		RequestData: map[string]interface{}{
			"reason": "post_checkout",
			"notes":  notes,
		},
		Comments:  &notes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.createWorkflow(ctx, workflow)
}

func (s *Service) isConditionDegraded(before, after *AssetCondition) bool {
	if before == nil || after == nil {
		return false
	}

	conditionOrder := map[AssetCondition]int{
		AssetConditionExcellent: 5,
		AssetConditionGood:      4,
		AssetConditionFair:      3,
		AssetConditionPoor:      2,
		AssetConditionBroken:    1,
	}

	return conditionOrder[*after] < conditionOrder[*before]
}

func (s *Service) createConditionDegradationAlert(ctx context.Context, assetID uuid.UUID, before, after *AssetCondition) error {
	alert := &AssetAlert{
		ID:          uuid.New(),
		AssetID:     assetID,
		AlertType:   "condition_degraded",
		Severity:    "high",
		Title:       "Asset Condition Degraded",
		Description: &[]string{fmt.Sprintf("Asset condition changed from %s to %s", *before, *after)}[0],
		TriggerDate: time.Now(),
		IsActive:    true,
		Metadata: map[string]interface{}{
			"condition_before": *before,
			"condition_after":  *after,
		},
		CreatedAt: time.Now(),
	}

	return s.createAlert(ctx, alert)
}

func (s *Service) createAlert(ctx context.Context, alert *AssetAlert) error {
	metadataJSON, _ := json.Marshal(alert.Metadata)

	_, err := s.db.Exec(ctx, `
		INSERT INTO asset_alerts (
			id, asset_id, alert_type, severity, title, description, 
			trigger_date, is_active, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		alert.ID, alert.AssetID, alert.AlertType, alert.Severity,
		alert.Title, alert.Description, alert.TriggerDate,
		alert.IsActive, metadataJSON, alert.CreatedAt)

	return err
}
