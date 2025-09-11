-- +goose Up
insert into teams (name) values ('Support'), ('IT') on conflict do nothing;
insert into sla_policies (name, priority, response_target_mins, resolution_target_mins, update_cadence_mins) values
    ('Standard', 3, 60, 1440, 30),
    ('High Priority', 1, 15, 480, 15)
on conflict do nothing;

-- +goose Down
delete from sla_policies where name in ('Standard', 'High Priority');
delete from teams where name in ('Support', 'IT');
