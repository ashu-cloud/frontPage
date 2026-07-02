CREATE TABLE watchlist (
    id          INTEGER PRIMARY KEY,
    user_id     INTEGER NOT NULL,
    symbol      TEXT NOT NULL,
    created_at  TEXT NOT NULL
);

INSERT INTO watchlist (id, user_id, symbol, created_at) VALUES
    (1, 42, 'AAPL', '2026-08-02T11:04:00Z'),
    (2, 42, 'NVDA', '2026-08-02T11:05:12Z'),
    (3, 42, 'GOOGL', '2026-08-04T09:22:41Z'),
    (4, 42, 'JPM', '2026-08-11T15:38:07Z'),
    (5, 7, 'MSFT', '2026-08-03T10:01:33Z'),
    (6, 7, 'AMZN', '2026-08-03T10:02:09Z'),
    (7, 101, 'COIN', '2026-08-19T18:44:50Z'),
    (8, 101, 'SQ', '2026-08-19T18:45:22Z');
