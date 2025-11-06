create table rates (
    id bigserial primary key,
    name varchar(255) not null,
    code varchar(10) not null,
    price decimal(20, 8) not null,
    created_at timestamp not null default current_timestamp,
    updated_at timestamp not null default current_timestamp
);