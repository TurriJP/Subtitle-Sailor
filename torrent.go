/*
https://www.linuxbabe.com/ubuntu/install-qbittorrent-ubuntu-18-04-desktop-server

sudo add-apt-repository ppa:qbittorrent-team/qbittorrent-stable
sudo apt install qbittorrent

sudo adduser --system --group qbittorrent-nox

sudo adduser your-username qbittorrent-nox

sudo nano /etc/systemd/system/qbittorrent-nox.service

FILE:
[Unit]
Description=qBittorrent Command Line Client
After=network.target

[Service]
#Do not change to "simple"
Type=forking
User=qbittorrent-nox
Group=qbittorrent-nox
UMask=007
ExecStart=/usr/bin/qbittorrent-nox -d --webui-port=8080
Restart=on-failure

[Install]
WantedBy=multi-user.target
ENDFILE


sudo systemctl start qbittorrent-nox
systemctl status qbittorrent-nox

Enable at boot time: sudo systemctl enable qbittorrent-nox

get my private IP address: ip route get 1.1.1.1 | awk '{print $7}' | head -1)

localhost:8080

user: admin
password: adminadmin

qbittorrent-nox magnet:?xt=urn:btih:A42111C5890A343FF45731B90C98ACD4ED426E42&dn=Prospero%27s+Books+(Peter+Greenaway%2C+1991)&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337%2Fannounce&tr=udp%3A%2F%2Fopen.tracker.cl%3A1337%2Fannounce&tr=udp%3A%2F%2Fopen.demonii.com%3A1337%2Fannounce&tr=udp%3A%2F%2Fopen.stealth.si%3A80%2Fannounce&tr=udp%3A%2F%2Ftracker.torrent.eu.org%3A451%2Fannounce&tr=udp%3A%2F%2Fexodus.desync.com%3A6969%2Fannounce&tr=udp%3A%2F%2Fopen.dstud.io%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.ololosh.space%3A6969%2Fannounce&tr=udp%3A%2F%2Fexplodie.org%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker-udp.gbitt.info%3A80%2Fannounce --save-path=/media/jellyfin/MOVIES/

https://github.com/webtorrent/webtorrent-cli
*/

/*
https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)#get-torrent-list
qBitTorrent api:
curl -i --header 'Referer: http://localhost:8080' --data 'username=admin&password=adminadmin' http://localhost:8080/api/v2/auth/login
curl http://localhost:8080/api/v2/torrents/info --cookie "SID=hBc7TxF76ERhvIw0jQQ4LZ7Z1jQUV0tQ"
*/
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
