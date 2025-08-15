package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"moul.io/http2curl"
)

// TODO: generalizar isso
func NewJacketClient() (*OpenSubtitlesClient, error) {
	apiKey := os.Getenv("JACKETT_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("JACKETT_API_KEY environment variable not set")
	}

	return &OpenSubtitlesClient{
		apiKey:  apiKey,
		baseURL: "http://localhost:9117/api/v2.0",
	}, nil
}

func SearchFiles(c *OpenSubtitlesClient, params FileSearchParams) (*SearchResult, error) {
	query := ""
	if params.Type == "movie" {
		query = params.Title + " " + params.Year
	}
	if params.Type == "show" {
		// Handle the case where we have a specific episode search (like "Breaking Bad S01E03")
		// This happens when downloadNextEpisode calls SearchFiles with a formatted title
		if strings.Contains(params.Title, " S") && strings.Contains(params.Title, "E") {
			query = params.Title + " " + params.Year
		} else {
			// Legacy support: use MinSeason and MinEpisode if available
			season := 1
			episode := 1
			if params.MinSeason != nil {
				season = *params.MinSeason
			}
			if params.MinEpisode != nil {
				episode = *params.MinEpisode
			}
			query = params.Title + " " + fmt.Sprintf("S%02dE%02d", season, episode)
		}
	}

	trackerList := os.Getenv("TRACKER_LIST")

	urlParams := url.Values{}
	urlParams.Set("Query", query)
	urlParams.Set("apikey", c.apiKey)
	urlParams.Set("Tracker[]", trackerList)

	requestURL := c.baseURL + "/indexers/all/results?" + urlParams.Encode()

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "subwaycrawlertest v1.0")
	req.Header.Set("Host", "localhost")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", requestURL)

	command, _ := http2curl.GetCurlCommand(req)
	fmt.Println(command)

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

	var searchResult SearchResult
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &searchResult, nil

}
