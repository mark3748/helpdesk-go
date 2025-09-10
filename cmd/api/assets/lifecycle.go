package assets

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AssetLifecycleRule represents a rule for asset lifecycle transitions
type AssetLifecycleRule struct {
	FromStatus       AssetStatus   `json:"from_status"`
	ToStatus         AssetStatus   `json:"to_status"`
	RequiredRole     string        `json:"required_role,omitempty"`
	RequiresApproval bool          `json:"requires_approval,omitempty"`
	AutoTriggers     []AutoTrigger `json:"auto_triggers,omitempty"`
}

// AutoTrigger represents conditions that automatically trigger status changes
type AutoTrigger struct {
	Type      string        `json:"type"` // "time_based", "condition_based", "maintenance_due"
	Condition string        `json:"condition"`
	Action    AssetStatus   `json:"action"`
	Delay     time.Duration `json:"delay,omitempty"`
}

// AssetWorkflow represents a workflow for asset operations
type AssetWorkflow struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	AssetID     uuid.UUID              `json:"asset_id" db:"asset_id"`
	Type        string                 `json:"type" db:"type"`     // "assignment", "checkout", "maintenance", "disposal"
	Status      string                 `json:"status" db:"status"` // "pending", "approved", "rejected", "completed"
	RequestedBy uuid.UUID              `json:"requested_by" db:"requested_by"`
	ApprovedBy  *uuid.UUID             `json:"approved_by" db:"approved_by"`
	RequestData map[string]interface{} `json:"request_data" db:"request_data"`
	Comments    *string                `json:"comments" db:"comments"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	CompletedAt *time.Time             `json:"completed_at" db:"completed_at"`
}

// GetLifecycleRules returns the defined lifecycle rules
func GetLifecycleRules() []AssetLifecycleRule {
	return []AssetLifecycleRule{
		// Active transitions
		{FromStatus: AssetStatusActive, ToStatus: AssetStatusMaintenance, RequiredRole: "manager"},
		{FromStatus: AssetStatusActive, ToStatus: AssetStatusRetired, RequiredRole: "admin", RequiresApproval: true},
		{FromStatus: AssetStatusActive, ToStatus: AssetStatusInactive, RequiredRole: "manager"},

		// Maintenance transitions
		{FromStatus: AssetStatusMaintenance, ToStatus: AssetStatusActive, RequiredRole: "manager"},
		{FromStatus: AssetStatusMaintenance, ToStatus: AssetStatusRetired, RequiredRole: "admin"},

		// Inactive transitions
		{FromStatus: AssetStatusInactive, ToStatus: AssetStatusActive, RequiredRole: "manager"},
		{FromStatus: AssetStatusInactive, ToStatus: AssetStatusRetired, RequiredRole: "admin"},
		{FromStatus: AssetStatusInactive, ToStatus: AssetStatusDisposed, RequiredRole: "admin", RequiresApproval: true},

		// Retired transitions
		{FromStatus: AssetStatusRetired, ToStatus: AssetStatusDisposed, RequiredRole: "admin", RequiresApproval: true},
		{FromStatus: AssetStatusRetired, ToStatus: AssetStatusActive, RequiredRole: "admin"}, // Reactivation
	}
}

// ValidateStatusTransition checks if a status transition is allowed
func (s *Service) ValidateStatusTransition(from, to AssetStatus, userRoles []string) error {
	rules := GetLifecycleRules()

	for _, rule := range rules {
		if rule.FromStatus == from && rule.ToStatus == to {
			// Check role requirement
			if rule.RequiredRole != "" {
				hasRole := false
				for _, role := range userRoles {
					if role == rule.RequiredRole || role == "admin" { // Admin can override
						hasRole = true
						break
					}
				}
				if !hasRole {
					return fmt.Errorf("insufficient permissions: requires %s role", rule.RequiredRole)
				}
			}

			return nil // Valid transition
		}
	}

	return fmt.Errorf("invalid status transition from %s to %s", from, to)
}

// RequestStatusChange creates a workflow for status changes that require approval
func (s *Service) RequestStatusChange(ctx context.Context, assetID uuid.UUID, toStatus AssetStatus, requestedBy uuid.UUID, comments *string) (*AssetWorkflow, error) {
	// Get current asset status
	asset, err := s.GetAsset(ctx, assetID)
	if err != nil {
		return nil, err
	}

	// Check if approval is required
	rules := GetLifecycleRules()
	requiresApproval := false

	for _, rule := range rules {
		if rule.FromStatus == asset.Status && rule.ToStatus == toStatus {
			requiresApproval = rule.RequiresApproval
			break
		}
	}

	if !requiresApproval {
		return nil, fmt.Errorf("status change does not require workflow approval")
	}

	// Create workflow request
	workflow := &AssetWorkflow{
		ID:          uuid.New(),
		AssetID:     assetID,
		Type:        "status_change",
		Status:      "pending",
		RequestedBy: requestedBy,
		RequestData: map[string]interface{}{
			"from_status": asset.Status,
			"to_status":   toStatus,
		},
		Comments:  comments,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save workflow
	err = s.createWorkflow(ctx, workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	return workflow, nil
}

// ApproveWorkflow approves a pending workflow
func (s *Service) ApproveWorkflow(ctx context.Context, workflowID uuid.UUID, approvedBy uuid.UUID, comments *string) error {
	// Get workflow
	workflow, err := s.getWorkflow(ctx, workflowID)
	if err != nil {
		return err
	}

	if workflow.Status != "pending" {
		return fmt.Errorf("workflow is not in pending status")
	}

	// Update workflow status
	workflow.Status = "approved"
	workflow.ApprovedBy = &approvedBy
	workflow.UpdatedAt = time.Now()

	if comments != nil {
		workflow.Comments = comments
	}

	err = s.updateWorkflow(ctx, workflow)
	if err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	// Execute the approved action
	return s.executeWorkflow(ctx, workflow)
}

// RejectWorkflow rejects a pending workflow
func (s *Service) RejectWorkflow(ctx context.Context, workflowID uuid.UUID, rejectedBy uuid.UUID, comments *string) error {
	workflow, err := s.getWorkflow(ctx, workflowID)
	if err != nil {
		return err
	}

	if workflow.Status != "pending" {
		return fmt.Errorf("workflow is not in pending status")
	}

	workflow.Status = "rejected"
	workflow.ApprovedBy = &rejectedBy
	workflow.UpdatedAt = time.Now()
	workflow.CompletedAt = &workflow.UpdatedAt

	if comments != nil {
		workflow.Comments = comments
	}

	return s.updateWorkflow(ctx, workflow)
}

// executeWorkflow executes an approved workflow
func (s *Service) executeWorkflow(ctx context.Context, workflow *AssetWorkflow) error {
	switch workflow.Type {
	case "status_change":
		return s.executeStatusChangeWorkflow(ctx, workflow)
	case "assignment":
		return s.executeAssignmentWorkflow(ctx, workflow)
	case "checkout":
		return s.executeCheckoutWorkflow(ctx, workflow)
	case "maintenance":
		return s.executeMaintenanceWorkflow(ctx, workflow)
	default:
		return fmt.Errorf("unknown workflow type: %s", workflow.Type)
	}
}

// executeStatusChangeWorkflow executes a status change workflow
func (s *Service) executeStatusChangeWorkflow(ctx context.Context, workflow *AssetWorkflow) error {
	toStatus := workflow.RequestData["to_status"].(string)

	// Update asset status
	updateReq := UpdateAssetRequest{
		Status: (*AssetStatus)(&toStatus),
	}

	actorID := workflow.RequestedBy
	if workflow.ApprovedBy != nil {
		actorID = *workflow.ApprovedBy
	}

	_, err := s.UpdateAsset(ctx, workflow.AssetID, updateReq, actorID)
	if err != nil {
		return fmt.Errorf("failed to update asset status: %w", err)
	}

	// Complete workflow
	now := time.Now()
	workflow.Status = "completed"
	workflow.CompletedAt = &now
	workflow.UpdatedAt = now

	return s.updateWorkflow(ctx, workflow)
}

// ScheduleMaintenanceReminder schedules maintenance reminders based on asset condition and warranty
func (s *Service) ScheduleMaintenanceReminder(ctx context.Context, assetID uuid.UUID) error {
	asset, err := s.GetAsset(ctx, assetID)
	if err != nil {
		return err
	}

	// Calculate next maintenance date based on condition and other factors
	var nextMaintenanceDate time.Time
	now := time.Now()

	if asset.Condition != nil {
		switch *asset.Condition {
		case AssetConditionExcellent:
			nextMaintenanceDate = now.AddDate(1, 0, 0) // 1 year
		case AssetConditionGood:
			nextMaintenanceDate = now.AddDate(0, 8, 0) // 8 months
		case AssetConditionFair:
			nextMaintenanceDate = now.AddDate(0, 4, 0) // 4 months
		case AssetConditionPoor:
			nextMaintenanceDate = now.AddDate(0, 2, 0) // 2 months
		case AssetConditionBroken:
			nextMaintenanceDate = now.AddDate(0, 0, 7) // 1 week (urgent)
		default:
			nextMaintenanceDate = now.AddDate(0, 6, 0) // Default 6 months
		}
	} else {
		nextMaintenanceDate = now.AddDate(0, 6, 0) // Default 6 months
	}

	// Create a maintenance workflow for future execution
	workflow := &AssetWorkflow{
		ID:          uuid.New(),
		AssetID:     assetID,
		Type:        "maintenance_reminder",
		Status:      "scheduled",
		RequestedBy: uuid.Nil, // System generated
		RequestData: map[string]interface{}{
			"scheduled_date": nextMaintenanceDate,
			"reminder_type":  "condition_based",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	return s.createWorkflow(ctx, workflow)
}

// CheckWarrantyExpiry checks for expiring warranties and creates alerts
func (s *Service) CheckWarrantyExpiry(ctx context.Context) error {
	// Find assets with warranties expiring in the next 30 days
	query := `
		SELECT id FROM assets 
		WHERE warranty_expiry IS NOT NULL 
		AND warranty_expiry <= $1 
		AND warranty_expiry > $2
		AND status IN ('active', 'maintenance')`

	thirtyDaysFromNow := time.Now().AddDate(0, 0, 30)
	today := time.Now()

	rows, err := s.db.Query(ctx, query, thirtyDaysFromNow, today)
	if err != nil {
		return fmt.Errorf("failed to query expiring warranties: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var assetID uuid.UUID
		if err := rows.Scan(&assetID); err != nil {
			continue
		}

		// Create warranty expiry notification workflow
		workflow := &AssetWorkflow{
			ID:          uuid.New(),
			AssetID:     assetID,
			Type:        "warranty_expiry",
			Status:      "pending",
			RequestedBy: uuid.Nil, // System generated
			RequestData: map[string]interface{}{
				"alert_type": "warranty_expiry",
				"checked_at": time.Now(),
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		_ = s.createWorkflow(ctx, workflow)
	}

	return nil
}

// Database operations for workflows
func (s *Service) createWorkflow(ctx context.Context, workflow *AssetWorkflow) error {
	requestDataJSON, _ := json.Marshal(workflow.RequestData)

	query := `
		INSERT INTO asset_workflows (
			id, asset_id, type, status, requested_by, approved_by, request_data, comments, created_at, updated_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := s.db.Exec(ctx, query,
		workflow.ID, workflow.AssetID, workflow.Type, workflow.Status,
		workflow.RequestedBy, workflow.ApprovedBy, requestDataJSON,
		workflow.Comments, workflow.CreatedAt, workflow.UpdatedAt, workflow.CompletedAt)

	return err
}

func (s *Service) updateWorkflow(ctx context.Context, workflow *AssetWorkflow) error {
	requestDataJSON, _ := json.Marshal(workflow.RequestData)

	query := `
		UPDATE asset_workflows 
		SET status = $1, approved_by = $2, request_data = $3, comments = $4, updated_at = $5, completed_at = $6
		WHERE id = $7`

	_, err := s.db.Exec(ctx, query,
		workflow.Status, workflow.ApprovedBy, requestDataJSON,
		workflow.Comments, workflow.UpdatedAt, workflow.CompletedAt, workflow.ID)

	return err
}

func (s *Service) getWorkflow(ctx context.Context, workflowID uuid.UUID) (*AssetWorkflow, error) {
	workflow := &AssetWorkflow{}
	var requestDataJSON []byte

	query := `
		SELECT id, asset_id, type, status, requested_by, approved_by, request_data, comments, created_at, updated_at, completed_at
		FROM asset_workflows WHERE id = $1`

	err := s.db.QueryRow(ctx, query, workflowID).Scan(
		&workflow.ID, &workflow.AssetID, &workflow.Type, &workflow.Status,
		&workflow.RequestedBy, &workflow.ApprovedBy, &requestDataJSON,
		&workflow.Comments, &workflow.CreatedAt, &workflow.UpdatedAt, &workflow.CompletedAt)

	if err != nil {
		return nil, err
	}

	if len(requestDataJSON) > 0 {
		_ = json.Unmarshal(requestDataJSON, &workflow.RequestData)
	}

	return workflow, nil
}

// Placeholder methods for other workflow types
func (s *Service) executeAssignmentWorkflow(_ context.Context, _ *AssetWorkflow) error {
	// Implementation for assignment workflow
	return fmt.Errorf("assignment workflow not yet implemented")
}

func (s *Service) executeCheckoutWorkflow(_ context.Context, _ *AssetWorkflow) error {
	// Implementation for checkout workflow
	return fmt.Errorf("checkout workflow not yet implemented")
}

func (s *Service) executeMaintenanceWorkflow(_ context.Context, _ *AssetWorkflow) error {
	// Implementation for maintenance workflow
	return fmt.Errorf("maintenance workflow not yet implemented")
}
