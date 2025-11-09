create table fx_last_rates (
    pair_id   bigint primary key references fx_pairs(id),
    value      numeric(20,10) not null,
    updated_at timestamptz not null
);