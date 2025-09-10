-- +goose Up

-- Asset workflows for approval processes and automation
create table if not exists asset_workflows (
    id uuid primary key default gen_random_uuid(),
    asset_id uuid not null references assets(id) on delete cascade,
    type text not null check (type in ('assignment', 'checkout', 'maintenance', 'disposal', 'status_change', 'maintenance_reminder', 'warranty_expiry')),
    status text not null default 'pending' check (status in ('pending', 'approved', 'rejected', 'completed', 'scheduled', 'cancelled')),
    requested_by uuid references users(id),
    approved_by uuid references users(id),
    request_data jsonb not null default '{}'::jsonb,
    comments text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    completed_at timestamptz
);

-- Asset maintenance schedules
create table if not exists asset_maintenance_schedules (
    id uuid primary key default gen_random_uuid(),
    asset_id uuid not null references assets(id) on delete cascade,
    schedule_type text not null check (schedule_type in ('preventive', 'condition_based', 'time_based', 'usage_based')),
    frequency_type text check (frequency_type in ('daily', 'weekly', 'monthly', 'quarterly', 'yearly', 'hours', 'cycles')),
    frequency_value int,
    last_maintenance_date date,
    next_maintenance_date date not null,
    maintenance_notes text,
    is_active boolean not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

-- Asset checkout/checkin records
create table if not exists asset_checkouts (
    id uuid primary key default gen_random_uuid(),
    asset_id uuid not null references assets(id) on delete cascade,
    checked_out_to_user_id uuid not null references users(id),
    checked_out_by_user_id uuid not null references users(id),
    checked_out_at timestamptz not null default now(),
    expected_return_date timestamptz,
    actual_return_date timestamptz,
    checkout_notes text,
    return_notes text,
    condition_at_checkout text check (condition_at_checkout in ('excellent', 'good', 'fair', 'poor', 'broken')),
    condition_at_return text check (condition_at_return in ('excellent', 'good', 'fair', 'poor', 'broken')),
    status text not null default 'active' check (status in ('active', 'returned', 'overdue', 'lost'))
);

-- Asset depreciation tracking
create table if not exists asset_depreciation_records (
    id uuid primary key default gen_random_uuid(),
    asset_id uuid not null references assets(id) on delete cascade,
    recorded_date date not null default current_date,
    book_value decimal(12,2) not null,
    accumulated_depreciation decimal(12,2) not null default 0,
    depreciation_method text not null default 'straight_line' check (depreciation_method in ('straight_line', 'declining_balance', 'sum_of_years', 'manual')),
    useful_life_months int,
    notes text,
    created_at timestamptz not null default now()
);

-- Asset alerts and notifications
create table if not exists asset_alerts (
    id uuid primary key default gen_random_uuid(),
    asset_id uuid not null references assets(id) on delete cascade,
    alert_type text not null check (alert_type in ('warranty_expiry', 'maintenance_due', 'checkout_overdue', 'condition_degraded', 'custom')),
    severity text not null default 'medium' check (severity in ('low', 'medium', 'high', 'critical')),
    title text not null,
    description text,
    trigger_date timestamptz not null default now(),
    acknowledged_at timestamptz,
    acknowledged_by uuid references users(id),
    resolved_at timestamptz,
    resolved_by uuid references users(id),
    is_active boolean not null default true,
    metadata jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

-- Indexes for performance
create index if not exists idx_asset_workflows_asset_id on asset_workflows(asset_id);
create index if not exists idx_asset_workflows_status on asset_workflows(status);
create index if not exists idx_asset_workflows_type on asset_workflows(type);
create index if not exists idx_asset_workflows_requested_by on asset_workflows(requested_by);

create index if not exists idx_asset_maintenance_schedules_asset_id on asset_maintenance_schedules(asset_id);
create index if not exists idx_asset_maintenance_schedules_next_date on asset_maintenance_schedules(next_maintenance_date);
create index if not exists idx_asset_maintenance_schedules_active on asset_maintenance_schedules(is_active);

create index if not exists idx_asset_checkouts_asset_id on asset_checkouts(asset_id);
create index if not exists idx_asset_checkouts_user_id on asset_checkouts(checked_out_to_user_id);
create index if not exists idx_asset_checkouts_status on asset_checkouts(status);
create index if not exists idx_asset_checkouts_return_date on asset_checkouts(expected_return_date);

create index if not exists idx_asset_depreciation_asset_id on asset_depreciation_records(asset_id);
create index if not exists idx_asset_depreciation_date on asset_depreciation_records(recorded_date);

create index if not exists idx_asset_alerts_asset_id on asset_alerts(asset_id);
create index if not exists idx_asset_alerts_type on asset_alerts(alert_type);
create index if not exists idx_asset_alerts_active on asset_alerts(is_active);
create index if not exists idx_asset_alerts_severity on asset_alerts(severity);

-- +goose Down

-- Remove indexes
drop index if exists idx_asset_alerts_severity;
drop index if exists idx_asset_alerts_active;
drop index if exists idx_asset_alerts_type;
drop index if exists idx_asset_alerts_asset_id;
drop index if exists idx_asset_depreciation_date;
drop index if exists idx_asset_depreciation_asset_id;
drop index if exists idx_asset_checkouts_return_date;
drop index if exists idx_asset_checkouts_status;
drop index if exists idx_asset_checkouts_user_id;
drop index if exists idx_asset_checkouts_asset_id;
drop index if exists idx_asset_maintenance_schedules_active;
drop index if exists idx_asset_maintenance_schedules_next_date;
drop index if exists idx_asset_maintenance_schedules_asset_id;
drop index if exists idx_asset_workflows_requested_by;
drop index if exists idx_asset_workflows_type;
drop index if exists idx_asset_workflows_status;
drop index if exists idx_asset_workflows_asset_id;

-- Drop tables
drop table if exists asset_alerts;
drop table if exists asset_depreciation_records;
drop table if exists asset_checkouts;
drop table if exists asset_maintenance_schedules;
drop table if exists asset_workflows;