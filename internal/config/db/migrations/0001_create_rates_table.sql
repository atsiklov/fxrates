create table rates (
    id bigserial primary key,
    name varchar(255) not null,
    code varchar(10) not null,
    price decimal(20, 8) not null,
    create_date timestamp not null default current_timestamp,
    update_date timestamp not null default current_timestamp
);