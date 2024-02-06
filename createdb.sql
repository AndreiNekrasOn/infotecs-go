CREATE TABLE IF NOT EXISTS wallets (
    id VARCHAR(256) PRIMARY KEY,
    balance DOUBLE PRECISION
);

CREATE TABLE IF NOT EXISTS transactions (
    from_id VARCHAR(256),
    to_id VARCHAR(256),
    amount DOUBLE PRECISION,
    time TIMESTAMP WITH TIME ZONE
);

CREATE index EXISTS ON transactions(from_id);

-- test values
INSERT INTO wallets (id, balance) VALUES ('abcd', 100);
INSERT INTO wallets (id, balance) VALUES ('bcda', 100);

