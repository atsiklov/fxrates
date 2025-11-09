create table currencies (
    code text primary key
);

insert into currencies(code) values
('USD'),
('EUR'),
('MXN'),
('GBP'),
('JPY')
on conflict do nothing;