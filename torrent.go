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

type QbittorrentClient struct {
	rootUrl string
	cookie  *string
	auth    string
	new     string
	list    string
}

type AuthRequest struct {
	username string
	password string
}

type NewRequest struct {
	urls     string
	savepath *string
}

type ListRequest struct {
	filter *string
}

type Torrent struct {
	completion_on int
	magnet_uri    string
	completed     int
	size          int
}

func NewQbittorrentClient() (*QbittorrentClient, error) {
	c := &QbittorrentClient{
		rootUrl: "http://localhost:8080/api/v2",
		auth:    "/auth/login",
		new:     "/torrents/add",
		list:    "/torrents/info",
	}

	return c, nil
}

func Authenticate(c *QbittorrentClient) error {
	requestUrl := c.rootUrl + c.auth
	fmt.Print(requestUrl)

	payload := AuthRequest{
		username: os.Getenv("QBITTORRENT_USER"),
		password: os.Getenv("QBITTORRENT_PASSWORD"),
	}

	data := url.Values{}
	data.Set("username", payload.username)
	data.Set("password", payload.password)

	req, err := http.NewRequest("POST", requestUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "subwaycrawlertest v1.0")
	req.Header.Set("Host", "localhost")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", requestUrl)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	c.cookie = &resp.Header["Set-Cookie"][0]

	return nil
}

func List(c *QbittorrentClient) (*[]Torrent, error) {
	requestUrl := c.rootUrl + c.list
	fmt.Print(requestUrl)

	status := "downloading"
	requestData := ListRequest{&status}

	data := url.Values{}
	data.Set("filter", *requestData.filter)

	req, err := http.NewRequest("POST", requestUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "subwaycrawlertest v1.0")
	req.Header.Set("Host", "localhost")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", requestUrl)
	req.Header.Set("Cookie", *c.cookie)

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

	var response []Torrent
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

func New(c *QbittorrentClient, params NewRequest) error {
	requestUrl := c.rootUrl + c.new
	fmt.Print(requestUrl)

	data := url.Values{}
	data.Set("urls", params.urls)
	if params.savepath != nil {
		data.Set("savepath", *params.savepath)
	}

	req, err := http.NewRequest("POST", requestUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "subwaycrawlertest v1.0")
	req.Header.Set("Host", "localhost")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", requestUrl)
	req.Header.Set("Cookie", *c.cookie)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var response string
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if response == "OK" {
		return nil
	}

	return fmt.Errorf("erro desconhecido")
}
