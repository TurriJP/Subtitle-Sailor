package main

import (
	"fmt"
	"strconv"
)

type SubtitleService struct {
	openSubtitlesClient *OpenSubtitlesClient
}

func NewSubtitleService() (*SubtitleService, error) {
	client, err := NewOpenSubtitlesClient()
	if err != nil {
		return nil, err
	}

	return &SubtitleService{
		openSubtitlesClient: client,
	}, nil
}

func (s *SubtitleService) FindSubtitlesForMovie(imdbID, title string, year int, movieHash, filename string) (*SubtitleEntry, error) {
	params := SubtitleSearchParams{
		Languages: "en",
		ImdbID:    imdbID,
		MovieHash: movieHash,
	}

	subtitles, err := s.openSubtitlesClient.SearchSubtitles(params)
	if err != nil {
		return nil, fmt.Errorf("failed to search subtitles: %w", err)
	}

	if subtitles.TotalCount == 0 && imdbID != "" {
		params = SubtitleSearchParams{
			Languages: "en",
			Query:     title,
			Year:      strconv.Itoa(year),
			MovieHash: movieHash,
		}

		subtitles, err = s.openSubtitlesClient.SearchSubtitles(params)
		if err != nil {
			return nil, fmt.Errorf("failed to search subtitles with fallback: %w", err)
		}
	}

	if subtitles.TotalCount == 0 {
		return nil, fmt.Errorf("no subtitles found")
	}

	bestSubtitle := SelectBestSubtitle(subtitles.Data, filename)
	return bestSubtitle, nil
}

func (s *SubtitleService) FindSubtitlesForEpisode(imdbID, title string, season, episode int, movieHash, filename string) (*SubtitleEntry, error) {
	params := SubtitleSearchParams{
		Languages: "en",
		ImdbID:    imdbID,
		MovieHash: movieHash,
	}

	subtitles, err := s.openSubtitlesClient.SearchSubtitles(params)
	if err != nil {
		return nil, fmt.Errorf("failed to search subtitles: %w", err)
	}

	if subtitles.TotalCount == 0 && imdbID != "" {
		params = SubtitleSearchParams{
			Languages:     "en",
			Query:         title,
			SeasonNumber:  strconv.Itoa(season),
			EpisodeNumber: strconv.Itoa(episode),
			MovieHash:     movieHash,
		}

		subtitles, err = s.openSubtitlesClient.SearchSubtitles(params)
		if err != nil {
			return nil, fmt.Errorf("failed to search subtitles with fallback: %w", err)
		}
	}

	if subtitles.TotalCount == 0 {
		return nil, fmt.Errorf("no subtitles found")
	}

	bestSubtitle := SelectBestSubtitle(subtitles.Data, filename)
	return bestSubtitle, nil
}

func (s *SubtitleService) DownloadSubtitles(fileId string) (*string, error) {
	downloadLink, err := s.openSubtitlesClient.GetSubtitlesDownloadLink(fileId)
	if err != nil {
		return nil, fmt.Errorf("failed to get download link: %w", err)
	}

	return &downloadLink.Link, nil
}
