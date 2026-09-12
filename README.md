# quotesvc

A small service for stock quotes and user watchlists.

## Running it

    go run ./cmd/server

Needs Go 1.23 or newer. Nothing else to install -- the database is
SQLite and gets created on first run.

    curl localhost:8080/quotes/1

Set `QUOTESVC_DB` to change the database path and `QUOTESVC_ADDR` to change
the listen address.

## Endpoints

    GET  /quotes/{id}          one quote
    GET  /quotes?limit=N       list quotes
    GET  /watchlist/{userID}   the symbols a user follows
    POST /watchlist            add a symbol to a user's watchlist

## Data

Quotes come from the market data feed. We track around 2M symbols, with
prices updating continuously through market hours -- roughly 2M writes an
hour.

The seed here is a 5,000 symbol slice so the service runs locally.

## Tests

    go test ./...

## Your task

Right now users can only find a stock if they already know its id.

Add a search endpoint so they can find quotes by symbol or by company name.

Use whatever editor and AI tools you normally work with.
