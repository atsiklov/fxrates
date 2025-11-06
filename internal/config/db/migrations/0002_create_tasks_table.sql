create table tasks (
    id bigserial primary key,
    code varchar(10) not null, -- todo что-то с кодами
    status varchar(50) not null,
    update_id uuid not null,
    new_price decimal(20, 8),
    created_at timestamp not null default current_timestamp,
    updated_at timestamp not null default current_timestamp
);

-- подумать над индексом