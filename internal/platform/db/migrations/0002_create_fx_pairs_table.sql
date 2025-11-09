create table fx_pairs (
    id    bigserial primary key,
    base  text not null references currencies(code),
    quote text not null references currencies(code)
    -- add unique base/quote constraint
);