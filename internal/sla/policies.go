package sla

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type policyDB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Policy represents an SLA policy.
type Policy struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Priority             int    `json:"priority"`
	ResponseTargetMins   int    `json:"response_target_mins"`
	ResolutionTargetMins int    `json:"resolution_target_mins"`
	UpdateCadenceMins    *int   `json:"update_cadence_mins,omitempty"`
}

// ListPolicies returns all SLA policies.
func ListPolicies(ctx context.Context, db policyDB) ([]Policy, error) {
	rows, err := db.Query(ctx, `select id::text, name, priority, response_target_mins, resolution_target_mins, update_cadence_mins from sla_policies order by priority`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Policy{}
	for rows.Next() {
		var p Policy
		if err := rows.Scan(&p.ID, &p.Name, &p.Priority, &p.ResponseTargetMins, &p.ResolutionTargetMins, &p.UpdateCadenceMins); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
