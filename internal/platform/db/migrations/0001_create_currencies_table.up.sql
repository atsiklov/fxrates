-- +goose Up
create table currencies (
    code text primary key
);

-- default initialization
insert into currencies(code) values
('USD'),
('EUR'),
('MXN'),
('GBP'),
('JPY'),
('CHF'),
('AUD'),
('CAD')
on conflict do nothing;
