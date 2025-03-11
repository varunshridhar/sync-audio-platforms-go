# Postman Collection

This directory contains Postman collections and environments for testing the Sync Audio Platforms API.

## Files
- `sync-audio-platforms.postman_collection.json`: The main Postman collection containing all API endpoints
- `local.postman_environment.json`: Environment variables for local development

## How to Use

1. Import the collection into Postman:
   - Open Postman
   - Click "Import"
   - Select the `sync-audio-platforms.postman_collection.json` file

2. Import the environment:
   - Click "Import" again
   - Select the `local.postman_environment.json` file
   - Select the "Local Environment" from the environment dropdown

3. Set up your environment variables:
   - Click the "eye" icon next to the environment dropdown
   - Update the variables as needed:
     - `base_url`: Your API base URL (default: http://localhost:8080)
     - `spotify_access_token`: Your Spotify access token (obtained after authentication)

## Available Endpoints

### Health
- GET `/health`: Check the API health status

### Authentication
- GET `/auth/login`: Initiate Spotify OAuth2 login flow
- GET `/callback`: Handle Spotify OAuth2 callback

### Spotify
- GET `/playlists`: Get user's Spotify playlists (requires authentication)

## Authentication Flow

1. Use the "Login with Spotify" endpoint to start the OAuth2 flow
2. After successful login, Spotify will redirect to the callback URL with a code
3. The callback endpoint will exchange the code for an access token
4. Use the access token in the `spotify_access_token` environment variable for authenticated requests 