# Sync Audio Platforms

A Go service to sync playlists between different audio streaming platforms.

## Features

- OAuth2 authentication with Spotify
- Fetch user's Spotify playlists
- Fetch tracks from Spotify playlists
- More platforms coming soon...

## Getting Started

### Prerequisites

- Go 1.21 or higher
- A Spotify Developer account and application
- MySQL database
- Redis instance
- Environment variables and configuration set up

### Configuration

The application uses both environment variables for sensitive data and a `config.yml` file for application configuration.


#### 1. Application Configuration

Create a `config.yml` file in the root directory based on the example:

```yaml
# Application Configuration
APP_PORT: "8080"
APP_LOG_LEVEL: "DEBUG"

# MySQL Database Configuration
MYSQL_HOST: "localhost"
MYSQL_PORT: 3306
MYSQL_USER: "your_username"
MYSQL_PASSWORD: "your_password"
MYSQL_DBNAME: "your_database"

# Redis Configuration
REDIS_HOST: "localhost"
REDIS_PORT: 6379
REDIS_PASSWORD: "your_password"
REDIS_DB: 1

# Spotify configuration
SPOTIFY_CLIENT_ID: "your_spotify_client_id"
SPOTIFY_CLIENT_SECRET: "your_spotify_client_secret"
SPOTIFY_REDIRECT_URI: "http://localhost:8080/callback/spotify"
```

### Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/sync-audio-platforms.git
```

2. Install dependencies:
```bash
go mod download
```

3. Set up configuration:
```bash
cp config.yml.example config.yml
#   Edit config.yml with your settings
```

4. Run the application:
```bash
go run cmd/main.go
```

## API Endpoints

### Spotify

#### Authentication
- `GET /auth/spotify`: Initiates Spotify OAuth2 flow
- `GET /callback/spotify`: Handles Spotify OAuth2 callback

#### Playlists
- `GET /api/spotify/playlists`: Get user's Spotify playlists
- `GET /api/spotify/playlists/{id}/tracks`: Get tracks from a specific playlist

## Project Structure

```
.
├── cmd/
│   └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── domain/
│   │   └── spotify.go
│   ├── infrastructure/
│   │   ├── spotify/
│   │   │   └── client.go
│   │   ├── mysql/
│   │   │   └── client.go
│   │   └── redis/
│   │       └── client.go
│   ├── usecase/
│   │   └── spotify_usecase.go
│   └── api/
│       └── handlers/
│           └── spotify_handler.go
├── .env
├── config.yml
├── config.yml.example
└── README.md
```

## Development

### Database Setup

1. Create a MySQL database using the configuration in `config.yml`
2. The application will handle table creation and migrations automatically

### Redis Setup

1. Ensure Redis is running with the configuration specified in `config.yml`
2. The application uses Redis for caching and session management

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Spotify Web API
- OAuth2 for Go
- GORM (Go ORM)
- Go-Redis
