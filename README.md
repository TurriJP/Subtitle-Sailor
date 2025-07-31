# Subtitle Sailor - Movie & TV Show Subtitle Finder

Sailor is a command-line tool that helps you find IMDB information and subtitles for movies and TV shows using the OMDb API and OpenSubtitles API.

For now this is a WIP as I learn golang - it's just searching for available subtitles, but the idea is to implement proper selection and download of the best matching subtitle.

## Prerequisites

- Go 1.18 or higher
- API keys for:
  - OMDb API (Open Movie Database)
  - OpenSubtitles API

## Usage

### Movies
Search for a movie and find subtitles:
```bash
go run . movie "Taxi Driver" 1976
```

With movie file for hash-based matching:
```bash
go run . movie "Taxi Driver" 1976 "/path/to/movie.mp4"
```

### TV Shows
Search for a TV show and find subtitles for all episodes:
```bash
go run . show "The Curse"
```

Search starting from a specific season and episode:
```bash
go run . show "The Curse" 1 6
```

## API Keys

### OMDb API
1. Register at https://www.omdbapi.com/apikey.aspx
2. Add your API key to `.env` as `OPEN_MOVIE_API_KEY`

### OpenSubtitles API
1. Register at https://www.opensubtitles.com/
2. Get your API key from your account settings
3. Add your API key to `.env` as `OPENSUBTITLES_API_KEY`

## Output Information

### Movies
- Movie title, year, and IMDB ID
- Best matching subtitle with:
  - Subtitle ID and slug
  - Release name
  - Download count
  - Filename
  - Trust indicator (✓ for trusted uploaders)

### TV Shows
- Series information (title, total seasons, IMDB ID)
- For each episode:
  - Episode title and IMDB ID
  - Best matching subtitle with trust indicator

## File Structure

```
sailor/
├── main.go              # Entry point and CLI handling
├── cli.go               # Command line interface logic
├── models.go            # Data structures for API responses
├── omdb.go              # OMDb API client
├── opensubtitles.go     # OpenSubtitles API client
├── subtitle_service.go  # Subtitle search service
├── subtitles.go         # Movie hash functionality
├── .env                 # Environment variables (not in git)
├── .gitignore           # Git ignore rules
└── README.md            # This file
```

## Dependencies

- `github.com/opensubtitlescli/moviehash` - For calculating movie file hashes