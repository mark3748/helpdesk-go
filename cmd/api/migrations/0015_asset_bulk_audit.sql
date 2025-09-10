-- +goose Up

-- Asset bulk operations tracking
create table if not exists asset_bulk_operations (
    id uuid primary key default gen_random_uuid(),
    type text not null check (type in ('import', 'export', 'update', 'delete', 'assign')),
    status text not null default 'pending' check (status in ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    requested_by uuid not null references users(id),
    parameters jsonb not null default '{}'::jsonb,
    progress int not null default 0 check (progress >= 0 and progress <= 100),
    total_items int not null default 0,
    processed_items int not null default 0,
    success_count int not null default 0,
    error_count int not null default 0,
    errors jsonb not null default '[]'::jsonb,
    results jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now(),
    started_at timestamptz,
    completed_at timestamptz
);

-- Enhanced audit events table
create table if not exists asset_audit_events (
    id uuid primary key default gen_random_uuid(),
    asset_id uuid references assets(id) on delete cascade,
    action text not null,
    actor_id uuid references users(id),
    actor_type text not null default 'user' check (actor_type in ('user', 'system', 'api')),
    category text not null default 'general' check (category in ('lifecycle', 'assignment', 'maintenance', 'financial', 'relationship', 'general')),
    severity text not null default 'info' check (severity in ('info', 'warning', 'error', 'critical')),
    old_values jsonb not null default '{}'::jsonb,
    new_values jsonb not null default '{}'::jsonb,
    context jsonb not null default '{}'::jsonb,
    ip_address inet,
    user_agent text,
    notes text,
    created_at timestamptz not null default now()
);

-- Asset depreciation calculations
create table if not exists asset_depreciation_schedules (
    id uuid primary key default gen_random_uuid(),
    asset_id uuid not null references assets(id) on delete cascade,
    depreciation_method text not null default 'straight_line' check (depreciation_method in ('straight_line', 'declining_balance', 'sum_of_years', 'units_of_production', 'manual')),
    useful_life_years int,
    useful_life_units int,
    salvage_value decimal(12,2) not null default 0,
    depreciation_rate decimal(5,2),
    start_date date not null default current_date,
    is_active boolean not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

-- Asset metrics and KPIs
create table if not exists asset_metrics (
    id uuid primary key default gen_random_uuid(),
    asset_id uuid not null references assets(id) on delete cascade,
    metric_type text not null check (metric_type in ('utilization', 'downtime', 'maintenance_cost', 'total_cost_ownership', 'performance')),
    metric_name text not null,
    metric_value decimal(15,4) not null,
    unit text,
    recorded_date date not null default current_date,
    recorded_by uuid references users(id),
    metadata jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

-- Asset compliance tracking
create table if not exists asset_compliance (
    id uuid primary key default gen_random_uuid(),
    asset_id uuid not null references assets(id) on delete cascade,
    compliance_type text not null check (compliance_type in ('regulatory', 'policy', 'standard', 'certification')),
    requirement_name text not null,
    status text not null check (status in ('compliant', 'non_compliant', 'pending', 'not_applicable')),
    last_audit_date date,
    next_audit_date date,
    auditor text,
    notes text,
    evidence_attachments text[], -- Array of attachment IDs
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

-- Asset location history (for tracking movement)
create table if not exists asset_location_history (
    id uuid primary key default gen_random_uuid(),
    asset_id uuid not null references assets(id) on delete cascade,
    from_location text,
    to_location text not null,
    moved_by uuid references users(id),
    moved_at timestamptz not null default now(),
    reason text,
    notes text,
    created_at timestamptz not null default now()
);

-- Asset tags and labels (for flexible categorization)
create table if not exists asset_tags (
    id uuid primary key default gen_random_uuid(),
    asset_id uuid not null references assets(id) on delete cascade,
    tag_name text not null,
    tag_value text,
    tag_type text not null default 'custom' check (tag_type in ('system', 'custom', 'auto')),
    created_by uuid references users(id),
    created_at timestamptz not null default now(),
    unique(asset_id, tag_name)
);

-- Indexes for performance
create index if not exists idx_asset_bulk_operations_requested_by on asset_bulk_operations(requested_by);
create index if not exists idx_asset_bulk_operations_status on asset_bulk_operations(status);
create index if not exists idx_asset_bulk_operations_type on asset_bulk_operations(type);
create index if not exists idx_asset_bulk_operations_created_at on asset_bulk_operations(created_at);

create index if not exists idx_asset_audit_events_asset_id on asset_audit_events(asset_id);
create index if not exists idx_asset_audit_events_actor_id on asset_audit_events(actor_id);
create index if not exists idx_asset_audit_events_action on asset_audit_events(action);
create index if not exists idx_asset_audit_events_category on asset_audit_events(category);
create index if not exists idx_asset_audit_events_created_at on asset_audit_events(created_at);
create index if not exists idx_asset_audit_events_severity on asset_audit_events(severity);

create index if not exists idx_asset_depreciation_schedules_asset_id on asset_depreciation_schedules(asset_id);
create index if not exists idx_asset_depreciation_schedules_active on asset_depreciation_schedules(is_active);

create index if not exists idx_asset_metrics_asset_id on asset_metrics(asset_id);
create index if not exists idx_asset_metrics_type on asset_metrics(metric_type);
create index if not exists idx_asset_metrics_date on asset_metrics(recorded_date);

create index if not exists idx_asset_compliance_asset_id on asset_compliance(asset_id);
create index if not exists idx_asset_compliance_type on asset_compliance(compliance_type);
create index if not exists idx_asset_compliance_status on asset_compliance(status);

create index if not exists idx_asset_location_history_asset_id on asset_location_history(asset_id);
create index if not exists idx_asset_location_history_moved_at on asset_location_history(moved_at);

create index if not exists idx_asset_tags_asset_id on asset_tags(asset_id);
create index if not exists idx_asset_tags_name on asset_tags(tag_name);
create index if not exists idx_asset_tags_type on asset_tags(tag_type);

-- Functions for automated depreciation calculation
create or replace function calculate_straight_line_depreciation(
    purchase_price decimal,
    salvage_value decimal,
    useful_life_years int,
    months_elapsed int
) returns decimal as $$
begin
    if useful_life_years = 0 then
        return 0;
    end if;
    return ((purchase_price - salvage_value) / useful_life_years) * (months_elapsed / 12.0);
end;
$$ language plpgsql;

-- Trigger to automatically record location changes
create or replace function record_location_change() returns trigger as $$
begin
    if old.location is distinct from new.location then
        insert into asset_location_history (asset_id, from_location, to_location, moved_at)
        values (new.id, old.location, new.location, now());
    end if;
    return new;
end;
$$ language plpgsql;

create trigger asset_location_change_trigger
    after update on assets
    for each row
    execute function record_location_change();

-- Trigger to update asset updated_at timestamp
create or replace function update_asset_timestamp() returns trigger as $$
begin
    new.updated_at = now();
    return new;
end;
$$ language plpgsql;

create trigger asset_update_timestamp_trigger
    before update on assets
    for each row
    execute function update_asset_timestamp();

-- +goose Down

-- Remove triggers
drop trigger if exists asset_update_timestamp_trigger on assets;
drop trigger if exists asset_location_change_trigger on assets;

-- Remove functions
drop function if exists update_asset_timestamp();
drop function if exists record_location_change();
drop function if exists calculate_straight_line_depreciation(decimal, decimal, int, int);

-- Remove indexes
drop index if exists idx_asset_tags_type;
drop index if exists idx_asset_tags_name;
drop index if exists idx_asset_tags_asset_id;
drop index if exists idx_asset_location_history_moved_at;
drop index if exists idx_asset_location_history_asset_id;
drop index if exists idx_asset_compliance_status;
drop index if exists idx_asset_compliance_type;
drop index if exists idx_asset_compliance_asset_id;
drop index if exists idx_asset_metrics_date;
drop index if exists idx_asset_metrics_type;
drop index if exists idx_asset_metrics_asset_id;
drop index if exists idx_asset_depreciation_schedules_active;
drop index if exists idx_asset_depreciation_schedules_asset_id;
drop index if exists idx_asset_audit_events_severity;
drop index if exists idx_asset_audit_events_created_at;
drop index if exists idx_asset_audit_events_category;
drop index if exists idx_asset_audit_events_action;
drop index if exists idx_asset_audit_events_actor_id;
drop index if exists idx_asset_audit_events_asset_id;
drop index if exists idx_asset_bulk_operations_created_at;
drop index if exists idx_asset_bulk_operations_type;
drop index if exists idx_asset_bulk_operations_status;
drop index if exists idx_asset_bulk_operations_requested_by;

-- Drop tables
drop table if exists asset_tags;
drop table if exists asset_location_history;
drop table if exists asset_compliance;
drop table if exists asset_metrics;
drop table if exists asset_depreciation_schedules;
drop table if exists asset_audit_events;
drop table if exists asset_bulk_operations;