-- +goose Up

-- Asset categories for organizing different types of assets
create table if not exists asset_categories (
    id uuid primary key default gen_random_uuid(),
    name text not null unique,
    description text,
    parent_id uuid references asset_categories(id) on delete set null,
    custom_fields jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

-- Core assets table
create table if not exists assets (
    id uuid primary key default gen_random_uuid(),
    asset_tag text not null unique,
    name text not null,
    description text,
    category_id uuid references asset_categories(id),
    status text not null default 'active' check (status in ('active', 'inactive', 'maintenance', 'retired', 'disposed')),
    condition text check (condition in ('excellent', 'good', 'fair', 'poor', 'broken')),
    
    -- Financial information
    purchase_price decimal(12,2),
    purchase_date date,
    warranty_expiry date,
    depreciation_rate decimal(5,2),
    current_value decimal(12,2),
    
    -- Physical details
    serial_number text,
    model text,
    manufacturer text,
    location text,
    
    -- Assignment
    assigned_to_user_id uuid references users(id),
    assigned_at timestamptz,
    
    -- Custom fields for flexibility
    custom_fields jsonb not null default '{}'::jsonb,
    
    -- Metadata
    created_by uuid references users(id),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

-- Asset assignments history
create table if not exists asset_assignments (
    id uuid primary key default gen_random_uuid(),
    asset_id uuid not null references assets(id) on delete cascade,
    assigned_to_user_id uuid references users(id),
    assigned_by_user_id uuid not null references users(id),
    assigned_at timestamptz not null default now(),
    unassigned_at timestamptz,
    notes text,
    status text not null default 'active' check (status in ('active', 'completed', 'cancelled'))
);

-- Asset relationships (dependencies, parent-child, etc.)
create table if not exists asset_relationships (
    id uuid primary key default gen_random_uuid(),
    parent_asset_id uuid not null references assets(id) on delete cascade,
    child_asset_id uuid not null references assets(id) on delete cascade,
    relationship_type text not null check (relationship_type in ('component', 'dependency', 'related', 'upgrade')),
    notes text,
    created_at timestamptz not null default now(),
    unique(parent_asset_id, child_asset_id, relationship_type)
);

-- Asset history for audit trail
create table if not exists asset_history (
    id uuid primary key default gen_random_uuid(),
    asset_id uuid not null references assets(id) on delete cascade,
    action text not null check (action in ('created', 'updated', 'assigned', 'unassigned', 'status_changed', 'maintenance', 'disposed')),
    actor_id uuid references users(id),
    old_values jsonb,
    new_values jsonb,
    notes text,
    created_at timestamptz not null default now()
);

-- Extend attachments table to support assets
alter table attachments add column if not exists asset_id uuid references assets(id) on delete cascade;
alter table attachments add constraint attachments_entity_check 
    check ((ticket_id is not null and asset_id is null) or (ticket_id is null and asset_id is not null));

-- Indexes for performance
create index if not exists idx_assets_asset_tag on assets(asset_tag);
create index if not exists idx_assets_category_id on assets(category_id);
create index if not exists idx_assets_assigned_to_user_id on assets(assigned_to_user_id);
create index if not exists idx_assets_status on assets(status);
create index if not exists idx_assets_created_at on assets(created_at);
create index if not exists idx_asset_assignments_asset_id on asset_assignments(asset_id);
create index if not exists idx_asset_assignments_assigned_to_user_id on asset_assignments(assigned_to_user_id);
create index if not exists idx_asset_history_asset_id on asset_history(asset_id);
create index if not exists idx_asset_relationships_parent on asset_relationships(parent_asset_id);
create index if not exists idx_asset_relationships_child on asset_relationships(child_asset_id);
create index if not exists idx_attachments_asset_id on attachments(asset_id);

-- Full-text search for assets
create index if not exists assets_fts on assets using gin (
    to_tsvector('english', 
        coalesce(name,'') || ' ' || 
        coalesce(description,'') || ' ' || 
        coalesce(asset_tag,'') || ' ' ||
        coalesce(serial_number,'') || ' ' ||
        coalesce(model,'') || ' ' ||
        coalesce(manufacturer,'')
    )
);

-- Insert default asset categories
insert into asset_categories (name, description) values 
    ('Hardware', 'Physical computing equipment and devices'),
    ('Software', 'Software licenses and applications'),
    ('Network', 'Networking equipment and infrastructure'),
    ('Furniture', 'Office furniture and fixtures'),
    ('Vehicles', 'Company vehicles and transportation'),
    ('Other', 'Miscellaneous assets')
on conflict (name) do nothing;

-- +goose Down

-- Remove indexes
drop index if exists assets_fts;
drop index if exists idx_attachments_asset_id;
drop index if exists idx_asset_relationships_child;
drop index if exists idx_asset_relationships_parent;
drop index if exists idx_asset_history_asset_id;
drop index if exists idx_asset_assignments_assigned_to_user_id;
drop index if exists idx_asset_assignments_asset_id;
drop index if exists idx_assets_created_at;
drop index if exists idx_assets_status;
drop index if exists idx_assets_assigned_to_user_id;
drop index if exists idx_assets_category_id;
drop index if exists idx_assets_asset_tag;

-- Remove constraint from attachments
alter table attachments drop constraint if exists attachments_entity_check;
alter table attachments drop column if exists asset_id;

-- Drop tables
drop table if exists asset_history;
drop table if exists asset_relationships;
drop table if exists asset_assignments;
drop table if exists assets;
drop table if exists asset_categories;