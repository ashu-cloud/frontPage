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

### 1. Search

`GET /quotes/{id}` is the only way to fetch a specific stock, so you have to
know its id already.

Add a search endpoint so users can find quotes by symbol or by company name.

### 2. A bug report

Support forwarded this:

> "One of the stocks on my watchlist has disappeared. I didn't remove it,
> and it was definitely there yesterday."

That's user id 7. Have a look at why this is happening.

---

Use whatever editor and AI tools you normally work with.

## Submitting

1. Clone this repo rather than forking it.
2. Push your work to a new public repo on your own GitHub account.
3. Work on a branch and open a PR against your own `main`, then send us the link.
