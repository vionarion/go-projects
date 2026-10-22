# myPet

Small Go projects I use to practice the language and its standard library.

| Project | What it is | Status |
|---|---|---|
| [simpleWeb](simpleWeb/) | HTTP API on `net/http` only: `GET/POST /people` (JSON), `/health` endpoint | works |
| [goWordGame](goWordGame/) | CLI helper for a word game: random letter + 4 random categories | works |
| [urlShortener](urlShortener/) | URL shortener on Fiber with Redis storage, rate limiting and link expiry, settings from environment variables | in progress |
| [simpleUrlShortener](simpleUrlShortener/) | First sketch of the shortener idea | sketch |

## Run

```bash
cd simpleWeb && go run .
curl -X POST localhost:8080/people -d '{"Name":"Ann","Age":30}'
curl localhost:8080/people
```

```bash
cd goWordGame && go run .
```

## Configuration

`urlShortener/api` reads its settings from a `.env` file. Copy `.env.example` to `.env` and fill in the values;
`.env` is ignored by git, so passwords never get into the repository.
