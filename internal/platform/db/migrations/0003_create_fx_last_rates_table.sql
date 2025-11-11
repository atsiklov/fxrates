-- +goose Up
create table fx_last_rates (
    pair_id    bigint primary key references fx_pairs(id) on delete cascade,
    value      numeric(16,8) not null,
    updated_at timestamptz not null default now()
);
