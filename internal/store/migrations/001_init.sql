CREATE TABLE quotes (
    id          INTEGER PRIMARY KEY,
    symbol      TEXT NOT NULL,
    name        TEXT NOT NULL,
    price       REAL NOT NULL,
    prev_close  REAL NOT NULL,
    updated_at  TEXT NOT NULL
);
