package assets

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AssetAnalytics represents comprehensive asset analytics data
type AssetAnalytics struct {
	Summary              AssetSummary              `json:"summary"`
	StatusDistribution   []StatusDistribution      `json:"status_distribution"`
	ConditionDistribution []ConditionDistribution  `json:"condition_distribution"`
	CategoryBreakdown    []CategoryBreakdown       `json:"category_breakdown"`
	AgeAnalysis          []AgeAnalysis             `json:"age_analysis"`
	MonthlyAcquisitions  []MonthlyAcquisition      `json:"monthly_acquisitions"`
	TopManufacturers     []ManufacturerStats       `json:"top_manufacturers"`
	LocationDistribution []LocationDistribution    `json:"location_distribution"`
	DepreciationTrend    []DepreciationTrend       `json:"depreciation_trend"`
}

type AssetSummary struct {
	TotalAssets       int     `json:"total_assets"`
	ActiveAssets      int     `json:"active_assets"`
	MaintenanceAssets int     `json:"maintenance_assets"`
	RetiredAssets     int     `json:"retired_assets"`
	TotalValue        float64 `json:"total_value"`
	DepreciatedValue  float64 `json:"depreciated_value"`
}

type StatusDistribution struct {
	Status     string  `json:"status"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type ConditionDistribution struct {
	Condition  string  `json:"condition"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type CategoryBreakdown struct {
	Category      string  `json:"category"`
	Count         int     `json:"count"`
	TotalValue    float64 `json:"total_value"`
	AvgAgeMonths  float64 `json:"avg_age_months"`
}

type AgeAnalysis struct {
	AgeRange   string  `json:"age_range"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type MonthlyAcquisition struct {
	Month string  `json:"month"`
	Count int     `json:"count"`
	Value float64 `json:"value"`
}

type ManufacturerStats struct {
	Manufacturer string  `json:"manufacturer"`
	Count        int     `json:"count"`
	Percentage   float64 `json:"percentage"`
}

type LocationDistribution struct {
	Location   string  `json:"location"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type DepreciationTrend struct {
	Month       string  `json:"month"`
	BookValue   float64 `json:"book_value"`
	MarketValue float64 `json:"market_value"`
}

// AnalyticsFilters represents filters for analytics queries
type AnalyticsFilters struct {
	StartDate  *time.Time
	EndDate    *time.Time
	CategoryID *uuid.UUID
}

// GetAssetAnalytics retrieves comprehensive analytics data for assets
func (s *Service) GetAssetAnalytics(ctx context.Context, filters AnalyticsFilters) (*AssetAnalytics, error) {
	analytics := &AssetAnalytics{}

	// Build base WHERE clause for filters
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filters.StartDate != nil {
		whereClause += fmt.Sprintf(" AND a.created_at >= $%d", argIndex)
		args = append(args, *filters.StartDate)
		argIndex++
	}

	if filters.EndDate != nil {
		whereClause += fmt.Sprintf(" AND a.created_at <= $%d", argIndex)
		args = append(args, *filters.EndDate)
		argIndex++
	}

	if filters.CategoryID != nil {
		whereClause += fmt.Sprintf(" AND a.category_id = $%d", argIndex)
		args = append(args, *filters.CategoryID)
		argIndex++
	}

	// Get summary statistics
	summary, err := s.getAssetSummary(ctx, whereClause, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset summary: %w", err)
	}
	analytics.Summary = *summary

	// Get status distribution
	statusDist, err := s.getStatusDistribution(ctx, whereClause, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get status distribution: %w", err)
	}
	analytics.StatusDistribution = statusDist

	// Get condition distribution
	conditionDist, err := s.getConditionDistribution(ctx, whereClause, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get condition distribution: %w", err)
	}
	analytics.ConditionDistribution = conditionDist

	// Get category breakdown
	categoryBreakdown, err := s.getCategoryBreakdown(ctx, whereClause, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get category breakdown: %w", err)
	}
	analytics.CategoryBreakdown = categoryBreakdown

	// Get age analysis
	ageAnalysis, err := s.getAgeAnalysis(ctx, whereClause, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get age analysis: %w", err)
	}
	analytics.AgeAnalysis = ageAnalysis

	// Get monthly acquisitions
	monthlyAcq, err := s.getMonthlyAcquisitions(ctx, whereClause, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly acquisitions: %w", err)
	}
	analytics.MonthlyAcquisitions = monthlyAcq

	// Get top manufacturers
	topMfg, err := s.getTopManufacturers(ctx, whereClause, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get top manufacturers: %w", err)
	}
	analytics.TopManufacturers = topMfg

	// Get location distribution
	locationDist, err := s.getLocationDistribution(ctx, whereClause, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get location distribution: %w", err)
	}
	analytics.LocationDistribution = locationDist

	// Get depreciation trend
	depTrend, err := s.getDepreciationTrend(ctx, whereClause, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get depreciation trend: %w", err)
	}
	analytics.DepreciationTrend = depTrend

	return analytics, nil
}

func (s *Service) getAssetSummary(ctx context.Context, whereClause string, args []interface{}) (*AssetSummary, error) {
	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_assets,
			COUNT(CASE WHEN a.status = 'active' THEN 1 END) as active_assets,
			COUNT(CASE WHEN a.status = 'maintenance' THEN 1 END) as maintenance_assets,
			COUNT(CASE WHEN a.status = 'retired' THEN 1 END) as retired_assets,
			COALESCE(SUM(a.purchase_price), 0) as total_value,
			COALESCE(SUM(
				CASE 
					WHEN a.purchase_price IS NOT NULL AND a.depreciation_rate IS NOT NULL AND a.purchase_date IS NOT NULL
					THEN a.purchase_price * (1 - (a.depreciation_rate / 100) * EXTRACT(YEAR FROM AGE(CURRENT_DATE, a.purchase_date)))
					ELSE a.purchase_price
				END
			), 0) as depreciated_value
		FROM assets a
		%s`, whereClause)

	summary := &AssetSummary{}
	err := s.db.QueryRow(ctx, query, args...).Scan(
		&summary.TotalAssets,
		&summary.ActiveAssets,
		&summary.MaintenanceAssets,
		&summary.RetiredAssets,
		&summary.TotalValue,
		&summary.DepreciatedValue,
	)

	return summary, err
}

func (s *Service) getStatusDistribution(ctx context.Context, whereClause string, args []interface{}) ([]StatusDistribution, error) {
	query := fmt.Sprintf(`
		SELECT 
			a.status,
			COUNT(*) as count,
			ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage
		FROM assets a
		%s
		GROUP BY a.status
		ORDER BY count DESC`, whereClause)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var distribution []StatusDistribution
	for rows.Next() {
		var item StatusDistribution
		err := rows.Scan(&item.Status, &item.Count, &item.Percentage)
		if err != nil {
			return nil, err
		}
		distribution = append(distribution, item)
	}

	return distribution, rows.Err()
}

func (s *Service) getConditionDistribution(ctx context.Context, whereClause string, args []interface{}) ([]ConditionDistribution, error) {
	query := fmt.Sprintf(`
		SELECT 
			COALESCE(a.condition, 'unknown') as condition,
			COUNT(*) as count,
			ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage
		FROM assets a
		%s
		GROUP BY a.condition
		ORDER BY count DESC`, whereClause)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var distribution []ConditionDistribution
	for rows.Next() {
		var item ConditionDistribution
		err := rows.Scan(&item.Condition, &item.Count, &item.Percentage)
		if err != nil {
			return nil, err
		}
		distribution = append(distribution, item)
	}

	return distribution, rows.Err()
}

func (s *Service) getCategoryBreakdown(ctx context.Context, whereClause string, args []interface{}) ([]CategoryBreakdown, error) {
	query := fmt.Sprintf(`
		SELECT 
			COALESCE(c.name, 'Uncategorized') as category,
			COUNT(a.id) as count,
			COALESCE(SUM(a.purchase_price), 0) as total_value,
			COALESCE(AVG(EXTRACT(EPOCH FROM AGE(CURRENT_DATE, a.purchase_date)) / (30.44 * 24 * 3600)), 0) as avg_age_months
		FROM assets a
		LEFT JOIN asset_categories c ON a.category_id = c.id
		%s
		GROUP BY c.name
		ORDER BY count DESC`, whereClause)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var breakdown []CategoryBreakdown
	for rows.Next() {
		var item CategoryBreakdown
		err := rows.Scan(&item.Category, &item.Count, &item.TotalValue, &item.AvgAgeMonths)
		if err != nil {
			return nil, err
		}
		breakdown = append(breakdown, item)
	}

	return breakdown, rows.Err()
}

func (s *Service) getAgeAnalysis(ctx context.Context, whereClause string, args []interface{}) ([]AgeAnalysis, error) {
	query := fmt.Sprintf(`
		SELECT 
			age_range,
			COUNT(*) as count,
			ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage
		FROM (
			SELECT 
				CASE 
					WHEN a.purchase_date IS NULL THEN 'Unknown'
					WHEN AGE(CURRENT_DATE, a.purchase_date) < INTERVAL '1 year' THEN '< 1 year'
					WHEN AGE(CURRENT_DATE, a.purchase_date) < INTERVAL '2 years' THEN '1-2 years'
					WHEN AGE(CURRENT_DATE, a.purchase_date) < INTERVAL '3 years' THEN '2-3 years'
					WHEN AGE(CURRENT_DATE, a.purchase_date) < INTERVAL '5 years' THEN '3-5 years'
					ELSE '5+ years'
				END as age_range
			FROM assets a
			%s
		) age_data
		GROUP BY age_range
		ORDER BY 
			CASE age_range
				WHEN '< 1 year' THEN 1
				WHEN '1-2 years' THEN 2
				WHEN '2-3 years' THEN 3
				WHEN '3-5 years' THEN 4
				WHEN '5+ years' THEN 5
				ELSE 6
			END`, whereClause)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var analysis []AgeAnalysis
	for rows.Next() {
		var item AgeAnalysis
		err := rows.Scan(&item.AgeRange, &item.Count, &item.Percentage)
		if err != nil {
			return nil, err
		}
		analysis = append(analysis, item)
	}

	return analysis, rows.Err()
}

func (s *Service) getMonthlyAcquisitions(ctx context.Context, whereClause string, args []interface{}) ([]MonthlyAcquisition, error) {
	query := fmt.Sprintf(`
		SELECT 
			TO_CHAR(a.created_at, 'YYYY-MM') as month,
			COUNT(*) as count,
			COALESCE(SUM(a.purchase_price), 0) as value
		FROM assets a
		%s AND a.created_at >= CURRENT_DATE - INTERVAL '12 months'
		GROUP BY TO_CHAR(a.created_at, 'YYYY-MM')
		ORDER BY month`, whereClause)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var acquisitions []MonthlyAcquisition
	for rows.Next() {
		var item MonthlyAcquisition
		err := rows.Scan(&item.Month, &item.Count, &item.Value)
		if err != nil {
			return nil, err
		}
		acquisitions = append(acquisitions, item)
	}

	return acquisitions, rows.Err()
}

func (s *Service) getTopManufacturers(ctx context.Context, whereClause string, args []interface{}) ([]ManufacturerStats, error) {
	query := fmt.Sprintf(`
		SELECT 
			COALESCE(a.manufacturer, 'Unknown') as manufacturer,
			COUNT(*) as count,
			ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage
		FROM assets a
		%s
		GROUP BY a.manufacturer
		ORDER BY count DESC
		LIMIT 10`, whereClause)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var manufacturers []ManufacturerStats
	for rows.Next() {
		var item ManufacturerStats
		err := rows.Scan(&item.Manufacturer, &item.Count, &item.Percentage)
		if err != nil {
			return nil, err
		}
		manufacturers = append(manufacturers, item)
	}

	return manufacturers, rows.Err()
}

func (s *Service) getLocationDistribution(ctx context.Context, whereClause string, args []interface{}) ([]LocationDistribution, error) {
	query := fmt.Sprintf(`
		SELECT 
			COALESCE(a.location, 'Unknown') as location,
			COUNT(*) as count,
			ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage
		FROM assets a
		%s
		GROUP BY a.location
		ORDER BY count DESC`, whereClause)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var distribution []LocationDistribution
	for rows.Next() {
		var item LocationDistribution
		err := rows.Scan(&item.Location, &item.Count, &item.Percentage)
		if err != nil {
			return nil, err
		}
		distribution = append(distribution, item)
	}

	return distribution, rows.Err()
}

func (s *Service) getDepreciationTrend(ctx context.Context, whereClause string, args []interface{}) ([]DepreciationTrend, error) {
	query := fmt.Sprintf(`
		SELECT 
			TO_CHAR(month_series, 'YYYY-MM') as month,
			COALESCE(SUM(
				CASE 
					WHEN a.purchase_price IS NOT NULL 
					THEN a.purchase_price
					ELSE 0
				END
			), 0) as book_value,
			COALESCE(SUM(
				CASE 
					WHEN a.purchase_price IS NOT NULL AND a.depreciation_rate IS NOT NULL AND a.purchase_date IS NOT NULL
					THEN a.purchase_price * (1 - (a.depreciation_rate / 100) * EXTRACT(YEAR FROM AGE(month_series, a.purchase_date)))
					ELSE a.purchase_price
				END
			), 0) as market_value
		FROM generate_series(
			CURRENT_DATE - INTERVAL '12 months',
			CURRENT_DATE,
			INTERVAL '1 month'
		) month_series
		LEFT JOIN assets a ON a.created_at <= month_series %s
		GROUP BY month_series
		ORDER BY month_series`, 
		func() string {
			if whereClause != "WHERE 1=1" {
				return "AND " + whereClause[10:] // Remove "WHERE 1=1 " prefix
			}
			return ""
		}())

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trend []DepreciationTrend
	for rows.Next() {
		var item DepreciationTrend
		err := rows.Scan(&item.Month, &item.BookValue, &item.MarketValue)
		if err != nil {
			return nil, err
		}
		trend = append(trend, item)
	}

	return trend, rows.Err()
}