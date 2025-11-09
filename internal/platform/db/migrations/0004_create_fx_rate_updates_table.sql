create table fx_rate_updates (
    pair_id    bigint not null references fx_pairs(id),
    update_id  uuid   unique not null,
    status     text   not null,
    value       numeric(20,10),
    updated_at timestamptz not null,
    primary key (pair_id, update_id)
    -- todo: add unique status && pair_id
);