package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type OpenSubtitlesClient struct {
	apiKey  string
	baseURL string
}

func NewOpenSubtitlesClient() (*OpenSubtitlesClient, error) {
	apiKey := os.Getenv("OPENSUBTITLES_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENSUBTITLES_API_KEY environment variable not set")
	}
	
	return &OpenSubtitlesClient{
		apiKey:  apiKey,
		baseURL: "https://api.opensubtitles.com/api/v1",
	}, nil
}

func (c *OpenSubtitlesClient) SearchSubtitles(params SubtitleSearchParams) (*SubtitleResponse, error) {
	urlParams := url.Values{}
	urlParams.Set("languages", "en")
	
	if params.ImdbID != "" {
		urlParams.Set("imdb_id", params.ImdbID)
	}
	
	if params.MovieHash != "" {
		urlParams.Set("moviehash", params.MovieHash)
	}
	
	if params.Query != "" {
		urlParams.Set("query", params.Query)
	}
	
	if params.Year != "" {
		urlParams.Set("year", params.Year)
	}
	
	if params.SeasonNumber != "" {
		urlParams.Set("season_number", params.SeasonNumber)
	}
	
	if params.EpisodeNumber != "" {
		urlParams.Set("episode_number", params.EpisodeNumber)
	}
	
	requestURL := c.baseURL + "/subtitles?" + urlParams.Encode()
	
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "subwaycrawlertest v1.0")
	req.Header.Set("Api-Key", c.apiKey)
	req.Header.Set("Host", "api.opensubtitles.com")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", requestURL)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var subtitleResp SubtitleResponse
	if err := json.Unmarshal(body, &subtitleResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return &subtitleResp, nil
}

func SelectBestSubtitle(subtitles []SubtitleEntry, filename string) *SubtitleEntry {
	if len(subtitles) == 0 {
		return nil
	}
	
	bestMatch := &subtitles[0]
	if filename != "" {
		bestScore := calculateSimilarity(filename, bestMatch.Attributes.Slug)
		
		for i := 1; i < len(subtitles); i++ {
			score := calculateSimilarity(filename, subtitles[i].Attributes.Slug)
			if score > bestScore {
				bestScore = score
				bestMatch = &subtitles[i]
			}
		}
	} else {
		for i := range subtitles {
			if subtitles[i].Attributes.FromTrusted {
				bestMatch = &subtitles[i]
				break
			}
		}
	}
	
	return bestMatch
}

func calculateSimilarity(filename, slug string) float64 {
	filename = strings.ToLower(filename)
	slug = strings.ToLower(slug)
	
	if filename == slug {
		return 1.0
	}
	
	if strings.Contains(slug, filename) || strings.Contains(filename, slug) {
		return 0.8
	}
	
	commonWords := 0
	filenameWords := strings.Fields(filename)
	slugWords := strings.Fields(slug)
	
	for _, fWord := range filenameWords {
		for _, sWord := range slugWords {
			if fWord == sWord && len(fWord) > 2 {
				commonWords++
			}
		}
	}
	
	maxWords := len(filenameWords)
	if len(slugWords) > maxWords {
		maxWords = len(slugWords)
	}
	
	if maxWords == 0 {
		return 0.0
	}
	
	return float64(commonWords) / float64(maxWords)
}