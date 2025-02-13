# OpenHeart Server

A Go implementation of the [Open Heart Protocol](https://openheart.fyi/).

A hosted version of this is available at https://openheart.tylery.com/  
Knock yourself out 😉

### Differences
- JSON is permitted `POST https://openheart.tylery.com/example.com { "emoji": "🌾"}`
- You may look up a specific emoji count: `GET https://https://openheart.tylery.com/example.com/🌾`

### Endpoints
```
GET https://openheart.tylery.com/example.com (200)
GET https://openheart.tylery.com/example.com/👩🏾‍❤️‍💋‍👩🏻 (200 | 404)
POST https://openheart.tylery.com/example.com (201 | 200)
```

### Examples

#### Creating a Reaction

Using plain text:
```bash
# Using curl
curl -X POST -d "💖" https://openheart.tylery.com/example.com

# Using fetch
fetch('https://openheart.tylery.com/example.com', {
  method: 'POST',
  body: '💖'
})
```

Using form data:
```bash
# Using curl
curl -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "💖=" \
  'https://openheart.tylery.com/example.com'

# Using fetch
fetch('https://openheart.tylery.com/example.com', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/x-www-form-urlencoded'
  },
  body: '💖='
})
```

Using JSON:
```bash
# Using curl
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"emoji": "💖"}' \
  'https://openheart.tylery.com/example.com'

# Using fetch
fetch('https://openheart.tylery.com/example.com', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({ emoji: '💖' })
})
```

#### Getting All Reactions

```bash
# Using curl
curl 'https://openheart.tylery.com/example.com'

# Using fetch
fetch('https://openheart.tylery.com/example.com')

# Response
{
  "💖": 5,
  "👍": 3,
  "🌟": 1
}
```

#### Getting Count for Specific Emoji

```bash
# Using curl
curl 'https://openheart.tylery.com/example.com/💖'

# Using fetch
fetch('https://openheart.tylery.com/example.com/❤')

# Response
{
  "💖": 5
}
```

## Configuration

The server can be configured through command line flags or environment variables. Command line flags take precedence over environment variables.

### Available Configuration Options

| Flag         | Environment Variable | Default                                 | Description                     |
|--------------|----------------------|-----------------------------------------|---------------------------------|
| `-http-port` | `HTTP_PORT`          | 4444                                    | Port number for the HTTP server |
| `-dsn`       | `DB_DSN`             | `user:password@tcp(host:port)/database` | Database connection string      |
| `-version`   | -                    | -                                       | Display version and exit        |

### Database Configuration

The database connection string (DSN) must be in the format: `user:password@tcp(host:port)/database`

For local development with Docker, you can use the included docker-compose.yml which requires the following environment variables:

```bash
DB_ROOT_PASSWORD=<root password>
DB_NAME=<database name>
DB_USER=<database user>
DB_PASSWORD=<database password>
```

### Example Usage

Using command line flags:
```bash
./openheart-protocol -http-port 8080 -dsn "user:pass@tcp(localhost:3306)/mydb"
```

Using environment variables:
```bash
export HTTP_PORT=8080
export DB_DSN=user:pass@tcp(localhost:3306)/mydb
./openheart-protocol
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/status` | Health check endpoint |
| GET | `/{url}` | Get all emoji reactions for a URL |
| GET | `/{url}/{emoji}` | Get count for specific emoji on a URL |
| POST | `/{url}` | Add emoji reaction to a URL |

## Development

1. Start the database:
```bash
docker compose up -d
```

2. Run the server:
```bash
go run ./cmd/api
```

The server includes automatic database migrations on startup.