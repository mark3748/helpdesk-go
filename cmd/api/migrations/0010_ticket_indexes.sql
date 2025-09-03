-- +goose Up
create index if not exists tickets_updated_at_id_idx on tickets (updated_at desc, id desc);
create index if not exists tickets_status_idx on tickets(status);
create index if not exists tickets_priority_idx on tickets(priority);
create index if not exists tickets_team_id_idx on tickets(team_id);
create index if not exists tickets_assignee_id_idx on tickets(assignee_id);
-- +goose Down
drop index if exists tickets_assignee_id_idx;
drop index if exists tickets_team_id_idx;
drop index if exists tickets_priority_idx;
drop index if exists tickets_status_idx;
drop index if exists tickets_updated_at_id_idx;
