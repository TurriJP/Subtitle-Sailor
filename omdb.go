package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type OMDBClient struct {
	apiKey  string
	baseURL string
}

func NewOMDBClient() (*OMDBClient, error) {
	apiKey := os.Getenv("OPEN_MOVIE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPEN_MOVIE_API_KEY environment variable not set")
	}
	
	return &OMDBClient{
		apiKey:  apiKey,
		baseURL: "http://www.omdbapi.com/",
	}, nil
}

func (c *OMDBClient) GetMovie(title string, year int) (*MovieResponse, error) {
	params := url.Values{}
	params.Set("apikey", c.apiKey)
	params.Set("t", title)
	params.Set("y", strconv.Itoa(year))
	
	requestURL := c.baseURL + "?" + params.Encode()
	
	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var movieResp MovieResponse
	if err := json.Unmarshal(body, &movieResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if movieResp.Response == "False" {
		return nil, fmt.Errorf("OMDB API error: %s", movieResp.Error)
	}
	
	return &movieResp, nil
}

func (c *OMDBClient) GetSeries(title string) (*SeriesResponse, error) {
	params := url.Values{}
	params.Set("apikey", c.apiKey)
	params.Set("t", title)
	
	requestURL := c.baseURL + "?" + params.Encode()
	
	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var seriesResp SeriesResponse
	if err := json.Unmarshal(body, &seriesResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if seriesResp.Response == "False" {
		return nil, fmt.Errorf("OMDB API error: %s", seriesResp.Error)
	}
	
	return &seriesResp, nil
}

func (c *OMDBClient) GetSeason(title string, season int) (*SeasonResponse, error) {
	params := url.Values{}
	params.Set("apikey", c.apiKey)
	params.Set("t", title)
	params.Set("Season", strconv.Itoa(season))
	
	requestURL := c.baseURL + "?" + params.Encode()
	
	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var seasonResp SeasonResponse
	if err := json.Unmarshal(body, &seasonResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if seasonResp.Response == "False" {
		return nil, fmt.Errorf("OMDB API error: %s", seasonResp.Error)
	}
	
	return &seasonResp, nil
}

func LoadEnv() error {
	data, err := os.ReadFile(".env")
	if err != nil {
		return err
	}
	
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			os.Setenv(key, value)
		}
	}
	
	return nil
}