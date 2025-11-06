create table tasks (
    id bigserial primary key,
    code varchar(10) not null, -- todo что-то с кодами
    status varchar(50) not null,
    update_id uuid not null,
    new_price decimal(20, 8),
    create_date timestamp not null default current_timestamp,
    update_date timestamp not null default current_timestamp
);

create index idx_tasks_update_id on tasks(update_id); -- подумать над индексом