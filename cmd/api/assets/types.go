package assets

import (
	"time"

	"github.com/google/uuid"
)

// AssetStatus represents the current status of an asset
type AssetStatus string

const (
	AssetStatusActive      AssetStatus = "active"
	AssetStatusInactive    AssetStatus = "inactive"
	AssetStatusMaintenance AssetStatus = "maintenance"
	AssetStatusRetired     AssetStatus = "retired"
	AssetStatusDisposed    AssetStatus = "disposed"
)

// AssetCondition represents the physical condition of an asset
type AssetCondition string

const (
	AssetConditionExcellent AssetCondition = "excellent"
	AssetConditionGood      AssetCondition = "good"
	AssetConditionFair      AssetCondition = "fair"
	AssetConditionPoor      AssetCondition = "poor"
	AssetConditionBroken    AssetCondition = "broken"
)

// RelationshipType represents the type of relationship between assets
type RelationshipType string

const (
	RelationshipComponent  RelationshipType = "component"
	RelationshipDependency RelationshipType = "dependency"
	RelationshipRelated    RelationshipType = "related"
	RelationshipUpgrade    RelationshipType = "upgrade"
)

// AssetHistoryAction represents the type of action recorded in asset history
type AssetHistoryAction string

const (
	ActionCreated       AssetHistoryAction = "created"
	ActionUpdated       AssetHistoryAction = "updated"
	ActionAssigned      AssetHistoryAction = "assigned"
	ActionUnassigned    AssetHistoryAction = "unassigned"
	ActionStatusChanged AssetHistoryAction = "status_changed"
	ActionMaintenance   AssetHistoryAction = "maintenance"
	ActionDisposed      AssetHistoryAction = "disposed"
)

// AssignmentStatus represents the status of an asset assignment
type AssignmentStatus string

const (
	AssignmentStatusActive    AssignmentStatus = "active"
	AssignmentStatusCompleted AssignmentStatus = "completed"
	AssignmentStatusCancelled AssignmentStatus = "cancelled"
)

// AssetCategory represents an asset category
type AssetCategory struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	Name         string                 `json:"name" db:"name"`
	Description  *string                `json:"description" db:"description"`
	ParentID     *uuid.UUID             `json:"parent_id" db:"parent_id"`
	CustomFields map[string]interface{} `json:"custom_fields" db:"custom_fields"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

// Asset represents a managed asset
type Asset struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	AssetTag    string          `json:"asset_tag" db:"asset_tag"`
	Name        string          `json:"name" db:"name"`
	Description *string         `json:"description" db:"description"`
	CategoryID  *uuid.UUID      `json:"category_id" db:"category_id"`
	Status      AssetStatus     `json:"status" db:"status"`
	Condition   *AssetCondition `json:"condition" db:"condition"`

	// Financial information
	PurchasePrice    *float64   `json:"purchase_price" db:"purchase_price"`
	PurchaseDate     *time.Time `json:"purchase_date" db:"purchase_date"`
	WarrantyExpiry   *time.Time `json:"warranty_expiry" db:"warranty_expiry"`
	DepreciationRate *float64   `json:"depreciation_rate" db:"depreciation_rate"`
	CurrentValue     *float64   `json:"current_value" db:"current_value"`

	// Physical details
	SerialNumber *string `json:"serial_number" db:"serial_number"`
	Model        *string `json:"model" db:"model"`
	Manufacturer *string `json:"manufacturer" db:"manufacturer"`
	Location     *string `json:"location" db:"location"`

	// Assignment
	AssignedToUserID *uuid.UUID `json:"assigned_to_user_id" db:"assigned_to_user_id"`
	AssignedAt       *time.Time `json:"assigned_at" db:"assigned_at"`

	// Custom fields for flexibility
	CustomFields map[string]interface{} `json:"custom_fields" db:"custom_fields"`

	// Metadata
	CreatedBy uuid.UUID `json:"created_by" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Joined fields (not stored in DB)
	Category      *AssetCategory `json:"category,omitempty"`
	AssignedUser  *AssetUser     `json:"assigned_user,omitempty"`
	CreatedByUser *AssetUser     `json:"created_by_user,omitempty"`
}

// AssetUser represents user information for asset context
type AssetUser struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name"`
}

// AssetAssignment represents an asset assignment record
type AssetAssignment struct {
	ID               uuid.UUID        `json:"id" db:"id"`
	AssetID          uuid.UUID        `json:"asset_id" db:"asset_id"`
	AssignedToUserID *uuid.UUID       `json:"assigned_to_user_id" db:"assigned_to_user_id"`
	AssignedByUserID uuid.UUID        `json:"assigned_by_user_id" db:"assigned_by_user_id"`
	AssignedAt       time.Time        `json:"assigned_at" db:"assigned_at"`
	UnassignedAt     *time.Time       `json:"unassigned_at" db:"unassigned_at"`
	Notes            *string          `json:"notes" db:"notes"`
	Status           AssignmentStatus `json:"status" db:"status"`

	// Joined fields
	Asset        *Asset     `json:"asset,omitempty"`
	AssignedUser *AssetUser `json:"assigned_user,omitempty"`
	AssignedBy   *AssetUser `json:"assigned_by,omitempty"`
}

// AssetRelationship represents a relationship between assets
type AssetRelationship struct {
	ID               uuid.UUID        `json:"id" db:"id"`
	ParentAssetID    uuid.UUID        `json:"parent_asset_id" db:"parent_asset_id"`
	ChildAssetID     uuid.UUID        `json:"child_asset_id" db:"child_asset_id"`
	RelationshipType RelationshipType `json:"relationship_type" db:"relationship_type"`
	Notes            *string          `json:"notes" db:"notes"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`

	// Joined fields
	ParentAsset *Asset `json:"parent_asset,omitempty"`
	ChildAsset  *Asset `json:"child_asset,omitempty"`
}

// AssetHistory represents a record in the asset audit trail
type AssetHistory struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	AssetID   uuid.UUID              `json:"asset_id" db:"asset_id"`
	Action    AssetHistoryAction     `json:"action" db:"action"`
	ActorID   *uuid.UUID             `json:"actor_id" db:"actor_id"`
	OldValues map[string]interface{} `json:"old_values" db:"old_values"`
	NewValues map[string]interface{} `json:"new_values" db:"new_values"`
	Notes     *string                `json:"notes" db:"notes"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`

	// Joined fields
	Asset *Asset     `json:"asset,omitempty"`
	Actor *AssetUser `json:"actor,omitempty"`
}

// CreateAssetRequest represents a request to create a new asset
type CreateAssetRequest struct {
	AssetTag         string                 `json:"asset_tag" binding:"required"`
	Name             string                 `json:"name" binding:"required"`
	Description      *string                `json:"description"`
	CategoryID       *uuid.UUID             `json:"category_id"`
	Status           AssetStatus            `json:"status"`
	Condition        *AssetCondition        `json:"condition"`
	PurchasePrice    *float64               `json:"purchase_price"`
	PurchaseDate     *time.Time             `json:"purchase_date"`
	WarrantyExpiry   *time.Time             `json:"warranty_expiry"`
	DepreciationRate *float64               `json:"depreciation_rate"`
	SerialNumber     *string                `json:"serial_number"`
	Model            *string                `json:"model"`
	Manufacturer     *string                `json:"manufacturer"`
	Location         *string                `json:"location"`
	CustomFields     map[string]interface{} `json:"custom_fields"`
}

// UpdateAssetRequest represents a request to update an asset
type UpdateAssetRequest struct {
	Name             *string                `json:"name"`
	Description      *string                `json:"description"`
	CategoryID       *uuid.UUID             `json:"category_id"`
	Status           *AssetStatus           `json:"status"`
	Condition        *AssetCondition        `json:"condition"`
	PurchasePrice    *float64               `json:"purchase_price"`
	PurchaseDate     *time.Time             `json:"purchase_date"`
	WarrantyExpiry   *time.Time             `json:"warranty_expiry"`
	DepreciationRate *float64               `json:"depreciation_rate"`
	CurrentValue     *float64               `json:"current_value"`
	SerialNumber     *string                `json:"serial_number"`
	Model            *string                `json:"model"`
	Manufacturer     *string                `json:"manufacturer"`
	Location         *string                `json:"location"`
	CustomFields     map[string]interface{} `json:"custom_fields"`
}

// AssignAssetRequest represents a request to assign an asset
type AssignAssetRequest struct {
	AssignedToUserID *uuid.UUID `json:"assigned_to_user_id"`
	Notes            *string    `json:"notes"`
}

// CreateCategoryRequest represents a request to create an asset category
type CreateCategoryRequest struct {
	Name         string                 `json:"name" binding:"required"`
	Description  *string                `json:"description"`
	ParentID     *uuid.UUID             `json:"parent_id"`
	CustomFields map[string]interface{} `json:"custom_fields"`
}

// UpdateCategoryRequest represents a request to update an asset category
type UpdateCategoryRequest struct {
	Name         *string                `json:"name"`
	Description  *string                `json:"description"`
	ParentID     *uuid.UUID             `json:"parent_id"`
	CustomFields map[string]interface{} `json:"custom_fields"`
}

// AssetSearchFilters represents filters for searching assets
type AssetSearchFilters struct {
	Query        string          `form:"q"`
	CategoryID   *uuid.UUID      `form:"category_id"`
	Status       []AssetStatus   `form:"status"`
	Condition    *AssetCondition `form:"condition"`
	AssignedTo   *uuid.UUID      `form:"assigned_to"`
	Manufacturer *string         `form:"manufacturer"`
	Location     *string         `form:"location"`
	Page         int             `form:"page"`
	Limit        int             `form:"limit"`
	SortBy       string          `form:"sort_by"`
	SortOrder    string          `form:"sort_order"`
}

// AssetListResponse represents a paginated list of assets
type AssetListResponse struct {
	Assets []Asset `json:"assets"`
	Total  int     `json:"total"`
	Page   int     `json:"page"`
	Limit  int     `json:"limit"`
	Pages  int     `json:"pages"`
}
