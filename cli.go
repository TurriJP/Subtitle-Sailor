package main

import (
	"fmt"
	"os"
	"strconv"
)

func HandleCLI() error {
	if err := LoadEnv(); err != nil {
		return fmt.Errorf("failed to load .env file: %w", err)
	}

	if len(os.Args) < 2 {
		return fmt.Errorf("usage: %s <movie|show> <title> [year|season] [episode]", os.Args[0])
	}

	command := os.Args[1]

	switch command {
	case "movie":
		return handleMovie()
	case "show":
		return handleShow()
	case "server":
		Server()
		return nil
	default:
		return fmt.Errorf("unknown command: %s. Use 'movie', 'show' or 'error'", command)
	}
}

func handleMovie() error {
	if len(os.Args) < 4 {
		return fmt.Errorf("usage: %s movie <title> <year> [filename]", os.Args[0])
	}

	title := os.Args[2]
	yearStr := os.Args[3]

	var filename string
	if len(os.Args) >= 5 {
		filename = os.Args[4]
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return fmt.Errorf("invalid year: %s", yearStr)
	}

	client, err := NewOMDBClient()
	if err != nil {
		return err
	}

	movie, err := client.GetMovieByTitle(title, fmt.Sprintf("%d", year))
	if err != nil {
		return err
	}

	fmt.Printf("Movie: %s (%s)\n", movie.Title, movie.Year)
	fmt.Printf("IMDB ID: %s\n", movie.ImdbID)

	subtitleService, err := NewSubtitleService()
	if err != nil {
		return err
	}

	var movieHash string
	if filename != "" {
		movieHash = GetHash(filename)
	}

	subtitle, err := subtitleService.FindSubtitlesForMovie(movie.ImdbID, title, year, movieHash, filename)
	if err != nil {
		fmt.Printf("Subtitle search failed: %v\n", err)
		return nil
	}

	fmt.Printf("\nBest subtitle match:\n")
	fmt.Printf("  ID: %s\n", subtitle.ID)
	fmt.Printf("  Slug: %s\n", subtitle.Attributes.Slug)
	fmt.Printf("  Release: %s\n", subtitle.Attributes.Release)
	fmt.Printf("  Language: %s\n", subtitle.Attributes.Language)
	fmt.Printf("  Download Count: %d\n", subtitle.Attributes.DownloadCount)
	if subtitle.Attributes.FromTrusted {
		fmt.Printf("  ✓ From trusted uploader\n")
	}
	if len(subtitle.Attributes.Files) > 0 {
		fmt.Printf("  Filename: %s\n", subtitle.Attributes.Files[0].FileName)
	}

	downloadLink, err := subtitleService.DownloadSubtitles(subtitle.ID)
	if err != nil {
		fmt.Printf("Failed to get download link: %v\n", err)
		return nil
	}

	fmt.Printf("Download Link: %s\n", *downloadLink)

	return nil
}

func handleShow() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: %s show <title> [min_season] [min_episode]", os.Args[0])
	}

	title := os.Args[2]
	minSeason := 1
	minEpisode := 1

	if len(os.Args) >= 4 {
		var err error
		minSeason, err = strconv.Atoi(os.Args[3])
		if err != nil {
			return fmt.Errorf("invalid season: %s", os.Args[3])
		}
	}

	if len(os.Args) >= 5 {
		var err error
		minEpisode, err = strconv.Atoi(os.Args[4])
		if err != nil {
			return fmt.Errorf("invalid episode: %s", os.Args[4])
		}
	}

	client, err := NewOMDBClient()
	if err != nil {
		return err
	}

	series, err := client.GetSeries(title)
	if err != nil {
		return err
	}

	totalSeasons, err := strconv.Atoi(series.TotalSeasons)
	if err != nil {
		return fmt.Errorf("invalid total seasons: %s", series.TotalSeasons)
	}

	fmt.Printf("Series: %s\n", series.Title)
	fmt.Printf("Total Seasons: %d\n", totalSeasons)
	fmt.Printf("IMDB ID: %s\n", series.ImdbID)
	fmt.Println()

	subtitleService, err := NewSubtitleService()
	if err != nil {
		return err
	}

	for season := minSeason; season <= totalSeasons; season++ {
		seasonData, err := client.GetSeason(title, season)
		if err != nil {
			fmt.Printf("Error getting season %d: %v\n", season, err)
			continue
		}

		fmt.Printf("Season %d:\n", season)

		startEpisode := 1
		if season == minSeason {
			startEpisode = minEpisode
		}

		for i, episode := range seasonData.Episodes {
			episodeNum := i + 1
			if episodeNum >= startEpisode {
				fmt.Printf("  Episode %s: %s (IMDB ID: %s)\n",
					episode.Episode, episode.Title, episode.ImdbID)

				subtitle, err := subtitleService.FindSubtitlesForEpisode(
					episode.ImdbID, title, season, episodeNum, "", "")
				if err != nil {
					fmt.Printf("    Subtitle search failed: %v\n", err)
				} else {
					fmt.Printf("    Best subtitle: %s", subtitle.Attributes.Slug)
					if subtitle.Attributes.FromTrusted {
						fmt.Printf(" ✓")
					}
					fmt.Println()
				}
			}
		}
		fmt.Println()
	}

	return nil
}
