package assets

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AuditEvent represents a comprehensive audit event
type AuditEvent struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	AssetID      uuid.UUID              `json:"asset_id" db:"asset_id"`
	Action       string                 `json:"action" db:"action"`
	ActorID      *uuid.UUID             `json:"actor_id" db:"actor_id"`
	ActorType    string                 `json:"actor_type" db:"actor_type"` // "user", "system", "api"
	Category     string                 `json:"category" db:"category"`     // "lifecycle", "assignment", "maintenance", "financial"
	Severity     string                 `json:"severity" db:"severity"`     // "info", "warning", "error", "critical"
	OldValues    map[string]interface{} `json:"old_values" db:"old_values"`
	NewValues    map[string]interface{} `json:"new_values" db:"new_values"`
	Changes      []FieldChange          `json:"changes"`
	Context      map[string]interface{} `json:"context" db:"context"`
	IPAddress    *string                `json:"ip_address" db:"ip_address"`
	UserAgent    *string                `json:"user_agent" db:"user_agent"`
	Notes        *string                `json:"notes" db:"notes"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	
	// Joined fields
	Asset *Asset     `json:"asset,omitempty"`
	Actor *AssetUser `json:"actor,omitempty"`
}

// FieldChange represents a change to a specific field
type FieldChange struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
	Type     string      `json:"type"` // "created", "updated", "deleted"
}

// AuditFilter represents filters for audit queries
type AuditFilter struct {
	AssetID     *uuid.UUID `form:"asset_id"`
	ActorID     *uuid.UUID `form:"actor_id"`
	Action      []string   `form:"action"`
	Category    []string   `form:"category"`
	Severity    []string   `form:"severity"`
	DateFrom    *time.Time `form:"date_from"`
	DateTo      *time.Time `form:"date_to"`
	IPAddress   *string    `form:"ip_address"`
	SearchQuery *string    `form:"q"`
	Page        int        `form:"page"`
	Limit       int        `form:"limit"`
}

// AuditSummary represents a summary of audit events
type AuditSummary struct {
	TotalEvents      int                    `json:"total_events"`
	EventsByAction   map[string]int         `json:"events_by_action"`
	EventsByCategory map[string]int         `json:"events_by_category"`
	EventsBySeverity map[string]int         `json:"events_by_severity"`
	TopActors        []ActorSummary         `json:"top_actors"`
	RecentEvents     []AuditEvent           `json:"recent_events"`
	TimeRange        map[string]interface{} `json:"time_range"`
}

// ActorSummary represents actor activity summary
type ActorSummary struct {
	ActorID     uuid.UUID `json:"actor_id"`
	ActorEmail  string    `json:"actor_email"`
	EventCount  int       `json:"event_count"`
	LastAction  time.Time `json:"last_action"`
}

// RecordAuditEvent records a comprehensive audit event
func (s *Service) RecordAuditEvent(ctx context.Context, event *AuditEvent) error {
	if event.ID == (uuid.UUID{}) {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	// Serialize JSON fields
	oldValuesJSON, _ := json.Marshal(event.OldValues)
	newValuesJSON, _ := json.Marshal(event.NewValues)
	contextJSON, _ := json.Marshal(event.Context)

	query := `
		INSERT INTO asset_audit_events (
			id, asset_id, action, actor_id, actor_type, category, severity,
			old_values, new_values, context, ip_address, user_agent, notes, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := s.db.Exec(ctx, query,
		event.ID, event.AssetID, event.Action, event.ActorID, event.ActorType,
		event.Category, event.Severity, oldValuesJSON, newValuesJSON,
		contextJSON, event.IPAddress, event.UserAgent, event.Notes, event.CreatedAt)

	return err
}

// GetAuditHistory retrieves audit history with filtering and pagination
func (s *Service) GetAuditHistory(ctx context.Context, filter AuditFilter) (*AuditListResponse, error) {
	// Build WHERE clause
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if filter.AssetID != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("ae.asset_id = $%d", argIndex))
		args = append(args, *filter.AssetID)
		argIndex++
	}

	if filter.ActorID != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("ae.actor_id = $%d", argIndex))
		args = append(args, *filter.ActorID)
		argIndex++
	}

	if len(filter.Action) > 0 {
		placeholders := make([]string, len(filter.Action))
		for i, action := range filter.Action {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, action)
			argIndex++
		}
		whereConditions = append(whereConditions, fmt.Sprintf("ae.action IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(filter.Category) > 0 {
		placeholders := make([]string, len(filter.Category))
		for i, category := range filter.Category {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, category)
			argIndex++
		}
		whereConditions = append(whereConditions, fmt.Sprintf("ae.category IN (%s)", strings.Join(placeholders, ",")))
	}

	if filter.DateFrom != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("ae.created_at >= $%d", argIndex))
		args = append(args, *filter.DateFrom)
		argIndex++
	}

	if filter.DateTo != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("ae.created_at <= $%d", argIndex))
		args = append(args, *filter.DateTo)
		argIndex++
	}

	if filter.SearchQuery != nil && *filter.SearchQuery != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("(ae.notes ILIKE $%d OR ae.action ILIKE $%d)", argIndex, argIndex))
		searchTerm := "%" + *filter.SearchQuery + "%"
		args = append(args, searchTerm)
		argIndex++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Set pagination defaults
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Page < 1 {
		filter.Page = 1
	}

	offset := (filter.Page - 1) * filter.Limit

	// Count total events
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM asset_audit_events ae %s`, whereClause)

	var total int
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count audit events: %w", err)
	}

	// Get events with joins
	query := fmt.Sprintf(`
		SELECT 
			ae.id, ae.asset_id, ae.action, ae.actor_id, ae.actor_type, ae.category, ae.severity,
			ae.old_values, ae.new_values, ae.context, ae.ip_address, ae.user_agent, ae.notes, ae.created_at,
			a.asset_tag, a.name as asset_name,
			u.email as actor_email, u.display_name as actor_display_name
		FROM asset_audit_events ae
		LEFT JOIN assets a ON ae.asset_id = a.id
		LEFT JOIN users u ON ae.actor_id = u.id
		%s
		ORDER BY ae.created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex, argIndex+1)

	args = append(args, filter.Limit, offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit events: %w", err)
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		event, err := s.scanAuditEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit event: %w", err)
		}
		events = append(events, *event)
	}

	// Calculate pages
	pages := (total + filter.Limit - 1) / filter.Limit

	return &AuditListResponse{
		Events: events,
		Total:  total,
		Page:   filter.Page,
		Limit:  filter.Limit,
		Pages:  pages,
	}, nil
}

// GetAuditSummary provides a summary of audit activity
func (s *Service) GetAuditSummary(ctx context.Context, assetID *uuid.UUID, dateFrom, dateTo *time.Time) (*AuditSummary, error) {
	summary := &AuditSummary{
		EventsByAction:   make(map[string]int),
		EventsByCategory: make(map[string]int),
		EventsBySeverity: make(map[string]int),
	}

	// Build base where clause for date/asset filtering
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if assetID != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("asset_id = $%d", argIndex))
		args = append(args, *assetID)
		argIndex++
	}

	if dateFrom != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *dateFrom)
		argIndex++
	}

	if dateTo != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *dateTo)
		argIndex++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM asset_audit_events %s", whereClause)
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&summary.TotalEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get events by action
	actionQuery := fmt.Sprintf(`
		SELECT action, COUNT(*) 
		FROM asset_audit_events %s 
		GROUP BY action 
		ORDER BY COUNT(*) DESC`, whereClause)

	rows, err := s.db.Query(ctx, actionQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by action: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var action string
		var count int
		if err := rows.Scan(&action, &count); err == nil {
			summary.EventsByAction[action] = count
		}
	}
	rows.Close()

	// Get events by category
	categoryQuery := fmt.Sprintf(`
		SELECT category, COUNT(*) 
		FROM asset_audit_events %s 
		GROUP BY category 
		ORDER BY COUNT(*) DESC`, whereClause)

	rows, err = s.db.Query(ctx, categoryQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by category: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err == nil {
			summary.EventsByCategory[category] = count
		}
	}
	rows.Close()

	// Get top actors
	actorQuery := fmt.Sprintf(`
		SELECT ae.actor_id, u.email, COUNT(*) as event_count, MAX(ae.created_at) as last_action
		FROM asset_audit_events ae
		LEFT JOIN users u ON ae.actor_id = u.id
		%s AND ae.actor_id IS NOT NULL
		GROUP BY ae.actor_id, u.email
		ORDER BY event_count DESC
		LIMIT 10`, whereClause)

	rows, err = s.db.Query(ctx, actorQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query top actors: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var actor ActorSummary
		var email *string
		if err := rows.Scan(&actor.ActorID, &email, &actor.EventCount, &actor.LastAction); err == nil {
			if email != nil {
				actor.ActorEmail = *email
			}
			summary.TopActors = append(summary.TopActors, actor)
		}
	}
	rows.Close()

	// Get recent events (last 10)
	recentQuery := fmt.Sprintf(`
		SELECT 
			ae.id, ae.asset_id, ae.action, ae.actor_id, ae.actor_type, ae.category, ae.severity,
			ae.old_values, ae.new_values, ae.context, ae.ip_address, ae.user_agent, ae.notes, ae.created_at,
			a.asset_tag, a.name as asset_name,
			u.email as actor_email, u.display_name as actor_display_name
		FROM asset_audit_events ae
		LEFT JOIN assets a ON ae.asset_id = a.id
		LEFT JOIN users u ON ae.actor_id = u.id
		%s
		ORDER BY ae.created_at DESC
		LIMIT 10`, whereClause)

	rows, err = s.db.Query(ctx, recentQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent events: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		event, err := s.scanAuditEvent(rows)
		if err != nil {
			continue
		}
		summary.RecentEvents = append(summary.RecentEvents, *event)
	}

	// Set time range
	summary.TimeRange = map[string]interface{}{
		"from": dateFrom,
		"to":   dateTo,
	}

	return summary, nil
}

// CompareAssetStates compares two asset states and generates field changes
func (s *Service) CompareAssetStates(oldAsset, newAsset *Asset) []FieldChange {
	var changes []FieldChange

	// Compare using reflection for comprehensive field comparison
	oldVal := reflect.ValueOf(*oldAsset)
	newVal := reflect.ValueOf(*newAsset)
	typ := reflect.TypeOf(*oldAsset)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldName := field.Name

		// Skip certain fields that shouldn't be compared
		if fieldName == "UpdatedAt" || fieldName == "CreatedAt" || 
		   fieldName == "Category" || fieldName == "AssignedUser" || fieldName == "CreatedByUser" {
			continue
		}

		oldFieldVal := oldVal.Field(i).Interface()
		newFieldVal := newVal.Field(i).Interface()

		if !reflect.DeepEqual(oldFieldVal, newFieldVal) {
			changes = append(changes, FieldChange{
				Field:    fieldName,
				OldValue: oldFieldVal,
				NewValue: newFieldVal,
				Type:     "updated",
			})
		}
	}

	return changes
}

// Enhanced audit recording methods for specific actions
func (s *Service) RecordAssetCreated(ctx context.Context, asset *Asset, createdBy uuid.UUID, context map[string]interface{}) error {
	event := &AuditEvent{
		AssetID:   asset.ID,
		Action:    "asset_created",
		ActorID:   &createdBy,
		ActorType: "user",
		Category:  "lifecycle",
		Severity:  "info",
		NewValues: s.assetToMap(asset),
		Context:   context,
		Notes:     &[]string{"Asset created"}[0],
	}

	return s.RecordAuditEvent(ctx, event)
}

func (s *Service) RecordAssetUpdated(ctx context.Context, oldAsset, newAsset *Asset, updatedBy uuid.UUID, context map[string]interface{}) error {
	changes := s.CompareAssetStates(oldAsset, newAsset)
	if len(changes) == 0 {
		return nil // No changes to record
	}

	event := &AuditEvent{
		AssetID:   newAsset.ID,
		Action:    "asset_updated",
		ActorID:   &updatedBy,
		ActorType: "user",
		Category:  "lifecycle",
		Severity:  "info",
		OldValues: s.assetToMap(oldAsset),
		NewValues: s.assetToMap(newAsset),
		Changes:   changes,
		Context:   context,
	}

	return s.RecordAuditEvent(ctx, event)
}

func (s *Service) RecordAssetAssigned(ctx context.Context, assetID, fromUserID, toUserID, assignedBy uuid.UUID, context map[string]interface{}) error {
	event := &AuditEvent{
		AssetID:   assetID,
		Action:    "asset_assigned",
		ActorID:   &assignedBy,
		ActorType: "user",
		Category:  "assignment",
		Severity:  "info",
		OldValues: map[string]interface{}{"assigned_to_user_id": fromUserID},
		NewValues: map[string]interface{}{"assigned_to_user_id": toUserID},
		Context:   context,
	}

	return s.RecordAuditEvent(ctx, event)
}

func (s *Service) RecordAssetCheckedOut(ctx context.Context, checkout *AssetCheckout, context map[string]interface{}) error {
	event := &AuditEvent{
		AssetID:   checkout.AssetID,
		Action:    "asset_checked_out",
		ActorID:   &checkout.CheckedOutByUserID,
		ActorType: "user",
		Category:  "assignment",
		Severity:  "info",
		NewValues: map[string]interface{}{
			"checked_out_to":       checkout.CheckedOutToUserID,
			"expected_return_date": checkout.ExpectedReturnDate,
			"condition_at_checkout": checkout.ConditionAtCheckout,
		},
		Context: context,
	}

	return s.RecordAuditEvent(ctx, event)
}

func (s *Service) RecordSystemEvent(ctx context.Context, assetID uuid.UUID, action string, category string, severity string, data map[string]interface{}, notes *string) error {
	event := &AuditEvent{
		AssetID:   assetID,
		Action:    action,
		ActorType: "system",
		Category:  category,
		Severity:  severity,
		NewValues: data,
		Notes:     notes,
	}

	return s.RecordAuditEvent(ctx, event)
}

// Helper methods
func (s *Service) scanAuditEvent(row pgx.Row) (*AuditEvent, error) {
	event := &AuditEvent{}
	var oldValuesJSON, newValuesJSON, contextJSON []byte
	var asset Asset
	var actor AssetUser

	var assetTag, assetName, actorEmail, actorDisplayName *string

	err := row.Scan(
		&event.ID, &event.AssetID, &event.Action, &event.ActorID, &event.ActorType,
		&event.Category, &event.Severity, &oldValuesJSON, &newValuesJSON, &contextJSON,
		&event.IPAddress, &event.UserAgent, &event.Notes, &event.CreatedAt,
		&assetTag, &assetName, &actorEmail, &actorDisplayName,
	)

	if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	if len(oldValuesJSON) > 0 {
		_ = json.Unmarshal(oldValuesJSON, &event.OldValues)
	}
	if len(newValuesJSON) > 0 {
		_ = json.Unmarshal(newValuesJSON, &event.NewValues)
	}
	if len(contextJSON) > 0 {
		_ = json.Unmarshal(contextJSON, &event.Context)
	}

	// Set asset info if available
	if assetTag != nil && assetName != nil {
		asset.ID = event.AssetID
		asset.AssetTag = *assetTag
		asset.Name = *assetName
		event.Asset = &asset
	}

	// Set actor info if available
	if event.ActorID != nil && actorEmail != nil {
		actor.ID = *event.ActorID
		actor.Email = *actorEmail
		actor.DisplayName = actorDisplayName
		event.Actor = &actor
	}

	// Generate changes from old/new values
	event.Changes = s.generateFieldChanges(event.OldValues, event.NewValues)

	return event, nil
}

func (s *Service) assetToMap(asset *Asset) map[string]interface{} {
	result := make(map[string]interface{})
	
	val := reflect.ValueOf(*asset)
	typ := reflect.TypeOf(*asset)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldName := strings.ToLower(field.Name)
		fieldValue := val.Field(i).Interface()
		
		// Skip unexported fields and joined fields
		if !field.IsExported() || 
		   fieldName == "category" || fieldName == "assigneduser" || fieldName == "createdbyuser" {
			continue
		}
		
		result[fieldName] = fieldValue
	}
	
	return result
}

func (s *Service) generateFieldChanges(oldValues, newValues map[string]interface{}) []FieldChange {
	var changes []FieldChange
	
	// Check for updates and additions
	for key, newVal := range newValues {
		if oldVal, exists := oldValues[key]; exists {
			if !reflect.DeepEqual(oldVal, newVal) {
				changes = append(changes, FieldChange{
					Field:    key,
					OldValue: oldVal,
					NewValue: newVal,
					Type:     "updated",
				})
			}
		} else {
			changes = append(changes, FieldChange{
				Field:    key,
				OldValue: nil,
				NewValue: newVal,
				Type:     "created",
			})
		}
	}
	
	// Check for deletions
	for key, oldVal := range oldValues {
		if _, exists := newValues[key]; !exists {
			changes = append(changes, FieldChange{
				Field:    key,
				OldValue: oldVal,
				NewValue: nil,
				Type:     "deleted",
			})
		}
	}
	
	return changes
}

// AuditListResponse represents paginated audit events
type AuditListResponse struct {
	Events []AuditEvent `json:"events"`
	Total  int          `json:"total"`
	Page   int          `json:"page"`
	Limit  int          `json:"limit"`
	Pages  int          `json:"pages"`
}