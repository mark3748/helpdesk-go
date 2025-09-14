package assets

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const maxCriticalPathDepth = 10

// RelationshipRequest represents a request to create or update an asset relationship
type RelationshipRequest struct {
	ParentAssetID    uuid.UUID        `json:"parent_asset_id" binding:"required"`
	ChildAssetID     uuid.UUID        `json:"child_asset_id" binding:"required"`
	RelationshipType RelationshipType `json:"relationship_type" binding:"required"`
	Notes            *string          `json:"notes"`
}

// AssetDependency represents a complex dependency structure
type AssetDependency struct {
	AssetID      uuid.UUID         `json:"asset_id"`
	Dependencies []AssetDependency `json:"dependencies,omitempty"`
	Asset        *Asset            `json:"asset,omitempty"`
	Depth        int               `json:"depth"`
}

// RelationshipGraph represents the full relationship graph for an asset
type RelationshipGraph struct {
	RootAsset    *Asset                       `json:"root_asset"`
	Parents      []AssetRelationship          `json:"parents"`
	Children     []AssetRelationship          `json:"children"`
	Dependencies map[string][]AssetDependency `json:"dependencies"`
	Components   []AssetRelationship          `json:"components"`
	Related      []AssetRelationship          `json:"related"`
}

// CreateRelationship creates a new asset relationship
func (s *Service) CreateRelationship(ctx context.Context, req RelationshipRequest, createdBy uuid.UUID) (*AssetRelationship, error) {
	// Validate assets exist
	if err := s.validateAssetsExist(ctx, req.ParentAssetID, req.ChildAssetID); err != nil {
		return nil, err
	}

	// Prevent self-referencing relationships
	if req.ParentAssetID == req.ChildAssetID {
		return nil, fmt.Errorf("cannot create relationship between asset and itself")
	}

	// Check for circular dependencies
	if err := s.checkCircularDependency(ctx, req.ParentAssetID, req.ChildAssetID, req.RelationshipType); err != nil {
		return nil, err
	}

	// Check if relationship already exists
	exists, err := s.relationshipExists(ctx, req.ParentAssetID, req.ChildAssetID, req.RelationshipType)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("relationship already exists")
	}

	// Create relationship
	relationship := &AssetRelationship{
		ID:               uuid.New(),
		ParentAssetID:    req.ParentAssetID,
		ChildAssetID:     req.ChildAssetID,
		RelationshipType: req.RelationshipType,
		Notes:            req.Notes,
		CreatedAt:        time.Now(),
	}

	query := `
		INSERT INTO asset_relationships (id, parent_asset_id, child_asset_id, relationship_type, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	err = s.db.QueryRow(ctx, query, relationship.ID, relationship.ParentAssetID,
		relationship.ChildAssetID, relationship.RelationshipType, relationship.Notes,
		relationship.CreatedAt).Scan(&relationship.ID, &relationship.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create relationship: %w", err)
	}

	// Record in history for both assets
	s.recordRelationshipHistory(ctx, req.ParentAssetID, "relationship_added", createdBy, map[string]interface{}{
		"relationship_type": req.RelationshipType,
		"child_asset_id":    req.ChildAssetID,
		"action":            "added",
	})

	s.recordRelationshipHistory(ctx, req.ChildAssetID, "relationship_added", createdBy, map[string]interface{}{
		"relationship_type": req.RelationshipType,
		"parent_asset_id":   req.ParentAssetID,
		"action":            "added",
	})

	return relationship, nil
}

// GetRelationshipGraph gets the complete relationship graph for an asset
func (s *Service) GetRelationshipGraph(ctx context.Context, assetID uuid.UUID, maxDepth int) (*RelationshipGraph, error) {
	// Get root asset
	rootAsset, err := s.GetAsset(ctx, assetID)
	if err != nil {
		return nil, err
	}

	graph := &RelationshipGraph{
		RootAsset:    rootAsset,
		Dependencies: make(map[string][]AssetDependency),
	}

	// Get parent relationships (assets this asset depends on)
	parents, err := s.getAssetRelationships(ctx, assetID, "child")
	if err != nil {
		return nil, err
	}
	graph.Parents = parents

	// Get child relationships (assets that depend on this asset)
	children, err := s.getAssetRelationships(ctx, assetID, "parent")
	if err != nil {
		return nil, err
	}
	graph.Children = children

	// Get components (assets that are part of this asset)
	components, err := s.getRelationshipsByType(ctx, assetID, RelationshipComponent, "parent")
	if err != nil {
		return nil, err
	}
	graph.Components = components

	// Get related assets (general relationships)
	related, err := s.getRelationshipsByType(ctx, assetID, RelationshipRelated, "both")
	if err != nil {
		return nil, err
	}
	graph.Related = related

	// Build dependency trees
	if maxDepth > 0 {
		graph.Dependencies["upstream"] = s.buildDependencyTree(ctx, assetID, "upstream", maxDepth, 0)
		graph.Dependencies["downstream"] = s.buildDependencyTree(ctx, assetID, "downstream", maxDepth, 0)
	}

	return graph, nil
}

// GetAssetsByRelationship finds assets related to a given asset
func (s *Service) GetAssetsByRelationship(ctx context.Context, assetID uuid.UUID, relationshipType RelationshipType, direction string) ([]Asset, error) {
	var query string
	var args []interface{}

	baseQuery := `
		SELECT DISTINCT 
			a.id, a.asset_tag, a.name, a.description, a.category_id, a.status, a.condition,
			a.purchase_price, a.purchase_date, a.warranty_expiry, a.depreciation_rate, a.current_value,
			a.serial_number, a.model, a.manufacturer, a.location, a.assigned_to_user_id, a.assigned_at,
			a.custom_fields, a.created_by, a.created_at, a.updated_at
		FROM assets a
		JOIN asset_relationships ar ON `

	switch direction {
	case "parent": // Get assets this asset depends on
		query = baseQuery + "ar.parent_asset_id = a.id WHERE ar.child_asset_id = $1"
		args = []interface{}{assetID}
	case "child": // Get assets that depend on this asset
		query = baseQuery + "ar.child_asset_id = a.id WHERE ar.parent_asset_id = $1"
		args = []interface{}{assetID}
	case "both": // Get all related assets
		query = baseQuery + "(ar.parent_asset_id = a.id OR ar.child_asset_id = a.id) WHERE (ar.parent_asset_id = $1 OR ar.child_asset_id = $1) AND a.id != $1"
		args = []interface{}{assetID}
	default:
		return nil, fmt.Errorf("invalid direction: %s", direction)
	}

	if relationshipType != "" {
		query += " AND ar.relationship_type = $" + fmt.Sprintf("%d", len(args)+1)
		args = append(args, relationshipType)
	}

	query += " ORDER BY a.name"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query related assets: %w", err)
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		asset, err := s.scanAsset(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan asset: %w", err)
		}
		assets = append(assets, *asset)
	}

	return assets, nil
}

// DeleteRelationship removes an asset relationship
func (s *Service) DeleteRelationship(ctx context.Context, relationshipID uuid.UUID, deletedBy uuid.UUID) error {
	// Get relationship details before deletion
	var parentID, childID uuid.UUID
	var relationshipType RelationshipType

	err := s.db.QueryRow(ctx, `
		SELECT parent_asset_id, child_asset_id, relationship_type 
		FROM asset_relationships WHERE id = $1`, relationshipID).
		Scan(&parentID, &childID, &relationshipType)

	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("relationship not found")
		}
		return fmt.Errorf("failed to get relationship: %w", err)
	}

	// Delete relationship
	_, err = s.db.Exec(ctx, "DELETE FROM asset_relationships WHERE id = $1", relationshipID)
	if err != nil {
		return fmt.Errorf("failed to delete relationship: %w", err)
	}

	// Record in history
	s.recordRelationshipHistory(ctx, parentID, "relationship_removed", deletedBy, map[string]interface{}{
		"relationship_type": relationshipType,
		"child_asset_id":    childID,
		"action":            "removed",
	})

	s.recordRelationshipHistory(ctx, childID, "relationship_removed", deletedBy, map[string]interface{}{
		"relationship_type": relationshipType,
		"parent_asset_id":   parentID,
		"action":            "removed",
	})

	return nil
}

// FindCircularDependencies identifies circular dependencies in the asset graph
func (s *Service) FindCircularDependencies(ctx context.Context) ([][]uuid.UUID, error) {
	// Get all dependency relationships
	query := `
		SELECT parent_asset_id, child_asset_id 
		FROM asset_relationships 
		WHERE relationship_type = 'dependency'`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query dependencies: %w", err)
	}
	defer rows.Close()

	// Build adjacency list
	graph := make(map[uuid.UUID][]uuid.UUID)
	allNodes := make(map[uuid.UUID]bool)

	for rows.Next() {
		var parentID, childID uuid.UUID
		if err := rows.Scan(&parentID, &childID); err != nil {
			continue
		}
		graph[parentID] = append(graph[parentID], childID)
		allNodes[parentID] = true
		allNodes[childID] = true
	}

	// Find cycles using DFS
	var cycles [][]uuid.UUID
	visited := make(map[uuid.UUID]bool)
	recStack := make(map[uuid.UUID]bool)
	path := make([]uuid.UUID, 0)

	var dfs func(uuid.UUID) bool
	dfs = func(nodeID uuid.UUID) bool {
		visited[nodeID] = true
		recStack[nodeID] = true
		path = append(path, nodeID)

		for _, neighbor := range graph[nodeID] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				// Found cycle - extract it from path
				cycleStart := -1
				for i, id := range path {
					if id == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := make([]uuid.UUID, len(path[cycleStart:]))
					copy(cycle, path[cycleStart:])
					cycles = append(cycles, cycle)
				}
				return true
			}
		}

		recStack[nodeID] = false
		if len(path) > 0 {
			path = path[:len(path)-1]
		}
		return false
	}

	for nodeID := range allNodes {
		if !visited[nodeID] {
			path = path[:0] // Reset path for new component
			dfs(nodeID)
		}
	}

	return cycles, nil
}

// GetAssetImpactAnalysis analyzes the impact of changes to an asset
func (s *Service) GetAssetImpactAnalysis(ctx context.Context, assetID uuid.UUID) (*map[string]interface{}, error) {
	analysis := make(map[string]interface{})

	// Count direct dependencies
	var directDependents int
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM asset_relationships 
		WHERE parent_asset_id = $1 AND relationship_type = 'dependency'`, assetID).
		Scan(&directDependents)
	if err != nil {
		return nil, err
	}

	// Count all downstream assets (recursive)
	downstreamAssets := s.getDownstreamAssetCount(ctx, assetID, make(map[uuid.UUID]bool))

	// Get critical path assets (assets that would be affected)
	criticalAssets, err := s.getCriticalPathAssets(ctx, assetID)
	if err != nil {
		return nil, err
	}

	// Check for single points of failure
	isSpof := s.isSinglePointOfFailure(ctx, assetID)

	analysis["asset_id"] = assetID
	analysis["direct_dependents"] = directDependents
	analysis["total_downstream_assets"] = downstreamAssets
	analysis["critical_assets"] = criticalAssets
	analysis["is_single_point_of_failure"] = isSpof
	analysis["risk_level"] = s.calculateRiskLevel(directDependents, downstreamAssets, isSpof)

	return &analysis, nil
}

// Helper methods
func (s *Service) validateAssetsExist(ctx context.Context, assetID1, assetID2 uuid.UUID) error {
	var count int
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM assets WHERE id IN ($1, $2)`, assetID1, assetID2).Scan(&count)
	if err != nil {
		return err
	}
	if count != 2 {
		return fmt.Errorf("one or both assets do not exist")
	}
	return nil
}

func (s *Service) relationshipExists(ctx context.Context, parentID, childID uuid.UUID, relType RelationshipType) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM asset_relationships 
		WHERE parent_asset_id = $1 AND child_asset_id = $2 AND relationship_type = $3)`,
		parentID, childID, relType).Scan(&exists)
	return exists, err
}

func (s *Service) checkCircularDependency(ctx context.Context, parentID, childID uuid.UUID, relType RelationshipType) error {
	if relType != RelationshipDependency {
		return nil // Only check for dependency relationships
	}

	// Check if adding this relationship would create a cycle
	visited := make(map[uuid.UUID]bool)
	return s.dfsCheckCycle(ctx, childID, parentID, visited)
}

func (s *Service) dfsCheckCycle(ctx context.Context, currentID, targetID uuid.UUID, visited map[uuid.UUID]bool) error {
	if currentID == targetID {
		return fmt.Errorf("circular dependency detected")
	}

	if visited[currentID] {
		return nil
	}
	visited[currentID] = true

	// Get all dependencies of current asset
	rows, err := s.db.Query(ctx, `
		SELECT child_asset_id FROM asset_relationships 
		WHERE parent_asset_id = $1 AND relationship_type = 'dependency'`, currentID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var childID uuid.UUID
		if err := rows.Scan(&childID); err != nil {
			continue
		}
		if err := s.dfsCheckCycle(ctx, childID, targetID, visited); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) getAssetRelationships(ctx context.Context, assetID uuid.UUID, direction string) ([]AssetRelationship, error) {
	var query string
	switch direction {
	case "parent":
		query = `
			SELECT ar.id, ar.parent_asset_id, ar.child_asset_id, ar.relationship_type, ar.notes, ar.created_at,
			       a.asset_tag, a.name
			FROM asset_relationships ar
			JOIN assets a ON ar.child_asset_id = a.id
			WHERE ar.parent_asset_id = $1`
	case "child":
		query = `
			SELECT ar.id, ar.parent_asset_id, ar.child_asset_id, ar.relationship_type, ar.notes, ar.created_at,
			       a.asset_tag, a.name
			FROM asset_relationships ar
			JOIN assets a ON ar.parent_asset_id = a.id
			WHERE ar.child_asset_id = $1`
	default:
		return nil, fmt.Errorf("invalid direction")
	}

	rows, err := s.db.Query(ctx, query, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []AssetRelationship
	for rows.Next() {
		var rel AssetRelationship
		var asset Asset
		err := rows.Scan(&rel.ID, &rel.ParentAssetID, &rel.ChildAssetID,
			&rel.RelationshipType, &rel.Notes, &rel.CreatedAt,
			&asset.AssetTag, &asset.Name)
		if err != nil {
			continue
		}

		if direction == "parent" {
			asset.ID = rel.ChildAssetID
			rel.ChildAsset = &asset
		} else {
			asset.ID = rel.ParentAssetID
			rel.ParentAsset = &asset
		}

		relationships = append(relationships, rel)
	}

	return relationships, nil
}

func (s *Service) getRelationshipsByType(ctx context.Context, assetID uuid.UUID, relType RelationshipType, direction string) ([]AssetRelationship, error) {
	var whereClause string
	switch direction {
	case "parent":
		whereClause = "ar.parent_asset_id = $1"
	case "child":
		whereClause = "ar.child_asset_id = $1"
	case "both":
		whereClause = "(ar.parent_asset_id = $1 OR ar.child_asset_id = $1)"
	default:
		return nil, fmt.Errorf("invalid direction")
	}

	query := fmt.Sprintf(`
		SELECT ar.id, ar.parent_asset_id, ar.child_asset_id, ar.relationship_type, ar.notes, ar.created_at
		FROM asset_relationships ar
		WHERE %s AND ar.relationship_type = $2`, whereClause)

	rows, err := s.db.Query(ctx, query, assetID, relType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []AssetRelationship
	for rows.Next() {
		var rel AssetRelationship
		err := rows.Scan(&rel.ID, &rel.ParentAssetID, &rel.ChildAssetID,
			&rel.RelationshipType, &rel.Notes, &rel.CreatedAt)
		if err != nil {
			continue
		}
		relationships = append(relationships, rel)
	}

	return relationships, nil
}

func (s *Service) buildDependencyTree(ctx context.Context, assetID uuid.UUID, direction string, maxDepth, currentDepth int) []AssetDependency {
	if currentDepth >= maxDepth {
		return nil
	}

	var query string
	switch direction {
	case "upstream":
		query = `SELECT parent_asset_id FROM asset_relationships WHERE child_asset_id = $1 AND relationship_type = 'dependency'`
	case "downstream":
		query = `SELECT child_asset_id FROM asset_relationships WHERE parent_asset_id = $1 AND relationship_type = 'dependency'`
	default:
		return nil
	}

	rows, err := s.db.Query(ctx, query, assetID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var dependencies []AssetDependency
	for rows.Next() {
		var relatedAssetID uuid.UUID
		if err := rows.Scan(&relatedAssetID); err != nil {
			continue
		}

		dependency := AssetDependency{
			AssetID: relatedAssetID,
			Depth:   currentDepth + 1,
		}

		// Get asset details
		if asset, err := s.GetAsset(ctx, relatedAssetID); err == nil {
			dependency.Asset = asset
		}

		// Recursively build sub-dependencies
		dependency.Dependencies = s.buildDependencyTree(ctx, relatedAssetID, direction, maxDepth, currentDepth+1)

		dependencies = append(dependencies, dependency)
	}

	return dependencies
}

func (s *Service) recordRelationshipHistory(ctx context.Context, assetID uuid.UUID, action string, actorID uuid.UUID, data map[string]interface{}) {
	historyData, _ := json.Marshal(data)
	_, _ = s.db.Exec(ctx, `
		INSERT INTO asset_history (asset_id, action, actor_id, new_values)
		VALUES ($1, $2, $3, $4)`,
		assetID, action, actorID, historyData)
}

// Additional helper methods for impact analysis
func (s *Service) getDownstreamAssetCount(ctx context.Context, assetID uuid.UUID, visited map[uuid.UUID]bool) int {
	if visited[assetID] {
		return 0
	}
	visited[assetID] = true

	rows, err := s.db.Query(ctx, `
		SELECT child_asset_id FROM asset_relationships 
		WHERE parent_asset_id = $1 AND relationship_type = 'dependency'`, assetID)
	if err != nil {
		return 0
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var childID uuid.UUID
		if err := rows.Scan(&childID); err != nil {
			continue
		}
		count += 1 + s.getDownstreamAssetCount(ctx, childID, visited)
	}

	return count
}

func (s *Service) getCriticalPathAssets(ctx context.Context, rootAssetID uuid.UUID) ([]uuid.UUID, error) {
	// Implement critical path analysis using recursive CTE to find dependency chains
	query := fmt.Sprintf(`
               WITH RECURSIVE dependency_path AS (
                       -- Base case: start with the root asset
                       SELECT
                               ar.parent_asset_id as asset_id,
                               ar.child_asset_id as dependent_id,
                               1 as depth,
                               ARRAY[ar.parent_asset_id] as path,
                               ar.relationship_type
                       FROM asset_relationships ar
                       WHERE ar.parent_asset_id = $1 AND ar.relationship_type IN ('depends_on', 'requires')

                       UNION ALL

                       -- Recursive case: find assets that depend on current assets
                       SELECT
                               ar.parent_asset_id,
                               ar.child_asset_id,
                               dp.depth + 1,
                               dp.path || ar.parent_asset_id,
                               ar.relationship_type
                       FROM asset_relationships ar
                       JOIN dependency_path dp ON ar.parent_asset_id = dp.dependent_id
                       WHERE dp.depth < %d -- Prevent infinite loops
                       AND NOT ar.parent_asset_id = ANY(dp.path) -- Prevent cycles
                       AND ar.relationship_type IN ('depends_on', 'requires')
               ),
               critical_assets AS (
			-- Find assets that are single points of failure
			SELECT DISTINCT dp.asset_id
			FROM dependency_path dp
			WHERE dp.asset_id IN (
				-- Assets with multiple dependents (high impact if they fail)
                               SELECT ar.parent_asset_id
                               FROM asset_relationships ar
                               WHERE ar.relationship_type IN ('depends_on', 'requires')
                               GROUP BY ar.parent_asset_id
                               HAVING COUNT(ar.child_asset_id) > 1
			)
			OR dp.asset_id IN (
				-- Assets that are themselves critical (marked as such)
				SELECT a.id
				FROM assets a
				WHERE a.custom_fields->>'is_critical' = 'true'
			)
		)
		SELECT DISTINCT ca.asset_id
		FROM critical_assets ca
		JOIN assets a ON ca.asset_id = a.id
		WHERE a.status = 'active' -- Only include active assets
               ORDER BY ca.asset_id`, maxCriticalPathDepth)

	rows, err := s.db.Query(ctx, query, rootAssetID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute critical path query: %w", err)
	}
	defer rows.Close()

	var criticalAssets []uuid.UUID
	for rows.Next() {
		var assetID uuid.UUID
		if err := rows.Scan(&assetID); err != nil {
			return nil, fmt.Errorf("failed to scan critical asset ID: %w", err)
		}
		criticalAssets = append(criticalAssets, assetID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating critical path results: %w", err)
	}

	return criticalAssets, nil
}

func (s *Service) isSinglePointOfFailure(ctx context.Context, assetID uuid.UUID) bool {
	var count int
	s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM asset_relationships 
		WHERE parent_asset_id = $1 AND relationship_type = 'dependency'`, assetID).Scan(&count)
	return count > 0
}

func (s *Service) calculateRiskLevel(directDependents, totalDownstream int, isSpof bool) string {
	if isSpof && totalDownstream > 10 {
		return "critical"
	}
	if totalDownstream > 5 {
		return "high"
	}
	if directDependents > 0 {
		return "medium"
	}
	return "low"
}

func (s *Service) scanAsset(row pgx.Row) (*Asset, error) {
	asset := &Asset{}
	var customFieldsJSON []byte

	err := row.Scan(
		&asset.ID, &asset.AssetTag, &asset.Name, &asset.Description, &asset.CategoryID,
		&asset.Status, &asset.Condition, &asset.PurchasePrice, &asset.PurchaseDate,
		&asset.WarrantyExpiry, &asset.DepreciationRate, &asset.CurrentValue,
		&asset.SerialNumber, &asset.Model, &asset.Manufacturer, &asset.Location,
		&asset.AssignedToUserID, &asset.AssignedAt, &customFieldsJSON,
		&asset.CreatedBy, &asset.CreatedAt, &asset.UpdatedAt)

	if err != nil {
		return nil, err
	}

	if len(customFieldsJSON) > 0 {
		_ = json.Unmarshal(customFieldsJSON, &asset.CustomFields)
	}

	return asset, nil
}
