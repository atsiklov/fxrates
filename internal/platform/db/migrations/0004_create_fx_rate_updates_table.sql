-- +goose Up
create table fx_rate_updates (
    id         bigserial primary key,
    pair_id    bigint not null references fx_pairs(id) on delete cascade,
    update_id  uuid   unique not null,
    status     text   not null,
    value      numeric(16,8),
    updated_at timestamptz not null default now(),
    constraint must_have_value_when_status_applied check ((status = 'applied') = (value is not null))
);

create unique index fx_rate_updates_one_pending_per_pair
    on fx_rate_updates(pair_id)
    where status = 'pending';
