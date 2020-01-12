alter table balances
    add created_at timestamp with time zone default now() not null;
