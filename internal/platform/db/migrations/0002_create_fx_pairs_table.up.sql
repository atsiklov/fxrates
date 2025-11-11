-- +goose Up
create table fx_pairs (
    id    bigserial primary key,
    base  text not null references currencies(code),
    quote text not null references currencies(code),
    constraint fx_pairs_distinct_ck check (base <> quote),
    unique (base, quote)
);
