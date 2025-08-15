package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type EpisodeDownloadItem struct {
	Season               int    `json:"season"`
	Episode              int    `json:"episode"`
	Title                string `json:"title"`
	OfficialTitle        string `json:"officialTitle"` // OMDB canonical name
	Year                 string `json:"year"`
	Type                 string `json:"type"`
	ReferenceTorrentSize int64  `json:"referenceTorrentSize"`
}

type DownloadQueue struct {
	Items []EpisodeDownloadItem `json:"items"`
	mutex sync.Mutex
}

type ShowMapping struct {
	UserTitle     string `json:"userTitle"`     // What user searched for
	OfficialTitle string `json:"officialTitle"` // OMDB canonical name
	SafeDirName   string `json:"safeDirName"`   // Filesystem-safe directory name
	Year          string `json:"year"`
}

type ShowMappingStore struct {
	Mappings []ShowMapping `json:"mappings"`
	mutex    sync.Mutex
}

const queueFileName = "/var/lib/sailor/download_queue.json"
const showMappingFileName = "/var/lib/sailor/show_mappings.json"
const defaultAPIKey = "sailor-local-api-key-2024"

var downloadQueue = &DownloadQueue{
	Items: make([]EpisodeDownloadItem, 0),
}

var showMappings = &ShowMappingStore{
	Mappings: make([]ShowMapping, 0),
}

func isLocalNetworkIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Define local network ranges
	localRanges := []string{
		"192.168.0.0/16", // Private Class C
		"10.0.0.0/8",     // Private Class A
		"172.16.0.0/12",  // Private Class B
		"127.0.0.0/8",    // Loopback
	}

	for _, cidr := range localRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(parsedIP) {
			return true
		}
	}

	return false
}

func enableCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	
	// If no origin header (like Postman, curl), allow for development
	if origin == "" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		// Parse the origin URL to extract hostname
		if originURL, err := url.Parse(origin); err == nil {
			hostname := originURL.Hostname()
			
			// Check if hostname is an IP address directly
			if ip := net.ParseIP(hostname); ip != nil {
				if isLocalNetworkIP(hostname) {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			} else {
				// It's a hostname, resolve it to IP addresses
				if ips, err := net.LookupIP(hostname); err == nil {
					for _, ip := range ips {
						if isLocalNetworkIP(ip.String()) {
							w.Header().Set("Access-Control-Allow-Origin", origin)
							break
						}
					}
				}
			}
		}
	}

	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
}

func getAPIKey() string {
	apiKey := os.Getenv("SAILOR_API_KEY")
	if apiKey == "" {
		return defaultAPIKey
	}
	return apiKey
}

func validateAPIKey(r *http.Request) bool {
	providedKey := r.Header.Get("X-API-Key")
	if providedKey == "" {
		// Check for API key in query parameter as fallback
		providedKey = r.URL.Query().Get("api_key")
	}

	expectedKey := getAPIKey()
	return providedKey == expectedKey
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w, r)
		if r.Method == "OPTIONS" {
			return
		}

		if !validateAPIKey(r) {
			http.Error(w, "Unauthorized: Invalid or missing API key", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (dq *DownloadQueue) saveToFile() error {
	data, err := json.MarshalIndent(dq, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal queue: %v", err)
	}

	return os.WriteFile(queueFileName, data, 0644)
}

func (dq *DownloadQueue) loadFromFile() error {
	data, err := os.ReadFile(queueFileName)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, start with empty queue
			return nil
		}
		return fmt.Errorf("failed to read queue file: %v", err)
	}

	return json.Unmarshal(data, dq)
}

func (dq *DownloadQueue) Enqueue(item EpisodeDownloadItem) {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	dq.Items = append(dq.Items, item)
	fmt.Printf("DEBUG: Enqueued item S%dE%d of %s. Queue now has %d items\n",
		item.Season, item.Episode, item.Title, len(dq.Items))

	// Persist to file (ignore errors to avoid blocking)
	if err := dq.saveToFile(); err != nil {
		fmt.Printf("Warning: Failed to save queue to file: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Successfully saved queue to file with %d items\n", len(dq.Items))
	}
}

func (dq *DownloadQueue) Dequeue() (EpisodeDownloadItem, bool) {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	if len(dq.Items) == 0 {
		return EpisodeDownloadItem{}, false
	}
	item := dq.Items[0]
	dq.Items = dq.Items[1:]

	// Persist to file (ignore errors to avoid blocking)
	if err := dq.saveToFile(); err != nil {
		fmt.Printf("Warning: Failed to save queue to file: %v\n", err)
	}

	return item, true
}

func (dq *DownloadQueue) IsEmpty() bool {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	isEmpty := len(dq.Items) == 0
	fmt.Printf("DEBUG: Queue.IsEmpty() called, has %d items, returning %v\n", len(dq.Items), isEmpty)
	return isEmpty
}

func (sms *ShowMappingStore) saveToFile() error {
	data, err := json.MarshalIndent(sms, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal show mappings: %v", err)
	}

	return os.WriteFile(showMappingFileName, data, 0644)
}

func (sms *ShowMappingStore) loadFromFile() error {
	data, err := os.ReadFile(showMappingFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read show mappings file: %v", err)
	}

	return json.Unmarshal(data, sms)
}

func (sms *ShowMappingStore) findOrCreateMapping(userTitle, officialTitle, year string) ShowMapping {
	sms.mutex.Lock()
	defer sms.mutex.Unlock()

	// First, try to find existing mapping by official title and year
	for _, mapping := range sms.Mappings {
		if mapping.OfficialTitle == officialTitle && mapping.Year == year {
			return mapping
		}
	}

	// Create new mapping
	safeName := sanitizeForFilesystem(officialTitle)
	newMapping := ShowMapping{
		UserTitle:     userTitle,
		OfficialTitle: officialTitle,
		SafeDirName:   safeName,
		Year:          year,
	}

	sms.Mappings = append(sms.Mappings, newMapping)

	// Persist to file
	if err := sms.saveToFile(); err != nil {
		fmt.Printf("Warning: Failed to save show mappings: %v\n", err)
	}

	return newMapping
}

func sanitizeForFilesystem(name string) string {
	// Remove or replace characters that are problematic for filesystems
	// Replace with safe alternatives
	reg := regexp.MustCompile(`[<>:"/\\|?*]`)
	safe := reg.ReplaceAllString(name, "_")

	// Remove multiple consecutive spaces and trim
	reg2 := regexp.MustCompile(`\s+`)
	safe = reg2.ReplaceAllString(safe, " ")
	safe = strings.TrimSpace(safe)

	// Limit length to avoid filesystem limits
	if len(safe) > 100 {
		safe = safe[:100]
	}

	return safe
}

func initializeQueue() {
	fmt.Printf("Initializing queue in directory: %s\n", getCurrentWorkingDirectory())

	if err := downloadQueue.loadFromFile(); err != nil {
		fmt.Printf("Warning: Failed to load queue from file: %v\n", err)
		fmt.Println("Starting with empty queue")
	} else {
		fmt.Printf("Loaded %d items from queue file\n", len(downloadQueue.Items))
	}

	if err := showMappings.loadFromFile(); err != nil {
		fmt.Printf("Warning: Failed to load show mappings from file: %v\n", err)
		fmt.Println("Starting with empty show mappings")
	} else {
		fmt.Printf("Loaded %d show mappings from file\n", len(showMappings.Mappings))
	}
}

func getCurrentWorkingDirectory() string {
	dir, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return dir
}

func generateEpisodeList(searchParams FileSearchParams, referenceTorrentSize int64) ([]EpisodeDownloadItem, error) {
	var episodes []EpisodeDownloadItem

	// Create OMDB client to get series information
	omdbClient, err := NewOMDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create OMDB client: %v", err)
	}

	// Get series information to know total number of seasons and official title
	seriesResp, err := omdbClient.GetSeries(searchParams.Title)
	if err != nil {
		return nil, fmt.Errorf("failed to get series info: %v", err)
	}

	// Create or find existing show mapping
	showMapping := showMappings.findOrCreateMapping(searchParams.Title, seriesResp.Title, seriesResp.Year)
	fmt.Printf("Using show mapping: '%s' -> '%s' (dir: %s)\n",
		showMapping.UserTitle, showMapping.OfficialTitle, showMapping.SafeDirName)

	totalSeasons := 1
	if seriesResp.TotalSeasons != "" {
		if parsed, err := strconv.Atoi(seriesResp.TotalSeasons); err == nil {
			totalSeasons = parsed
		}
	}

	// Set defaults and boundaries
	minSeason := 1
	maxSeason := totalSeasons
	minEpisode := 1
	maxEpisode := -1 // Will be determined per season

	// Apply user-specified parameters
	if searchParams.MinSeason != nil {
		minSeason = *searchParams.MinSeason
	}
	if searchParams.MaxSeason != nil {
		maxSeason = *searchParams.MaxSeason
	}
	if searchParams.MinEpisode != nil {
		minEpisode = *searchParams.MinEpisode
	}
	if searchParams.MaxEpisode != nil {
		maxEpisode = *searchParams.MaxEpisode
	}

	// Validate season boundaries
	if minSeason < 1 {
		minSeason = 1
	}
	if maxSeason > totalSeasons {
		maxSeason = totalSeasons
	}
	if minSeason > maxSeason {
		return nil, fmt.Errorf("minimum season (%d) cannot be greater than maximum season (%d)", minSeason, maxSeason)
	}

	// Generate episodes for each season
	for season := minSeason; season <= maxSeason; season++ {
		// Get season information to know number of episodes
		seasonResp, err := omdbClient.GetSeason(searchParams.Title, season)
		if err != nil {
			fmt.Printf("Warning: Could not get season %d info: %v\n", season, err)
			continue
		}

		totalEpisodesInSeason := len(seasonResp.Episodes)

		// Determine episode range for this season
		seasonMinEpisode := 1
		seasonMaxEpisode := totalEpisodesInSeason

		// For the first season, apply user's minimum episode
		if season == minSeason && minEpisode > 1 {
			seasonMinEpisode = minEpisode
		}

		// For the last season, apply user's maximum episode if specified
		if season == maxSeason && maxEpisode > 0 {
			if maxEpisode <= totalEpisodesInSeason {
				seasonMaxEpisode = maxEpisode
			}
		}

		// Validate episode boundaries for this season
		if seasonMinEpisode > totalEpisodesInSeason {
			fmt.Printf("Warning: Season %d only has %d episodes, skipping\n", season, totalEpisodesInSeason)
			continue
		}
		if seasonMaxEpisode > totalEpisodesInSeason {
			seasonMaxEpisode = totalEpisodesInSeason
		}

		// Add episodes for this season
		for episode := seasonMinEpisode; episode <= seasonMaxEpisode; episode++ {
			episodes = append(episodes, EpisodeDownloadItem{
				Season:               season,
				Episode:              episode,
				Title:                searchParams.Title,        // Keep original for search
				OfficialTitle:        showMapping.OfficialTitle, // OMDB canonical name
				Year:                 searchParams.Year,
				Type:                 searchParams.Type,
				ReferenceTorrentSize: referenceTorrentSize,
			})
		}
	}

	return episodes, nil
}

// selectBestTorrent chooses the best torrent based on size similarity to reference
// and seeders count. It considers the order (index) as a tiebreaker.
func selectBestTorrent(torrents []SearchResultEntry, referenceSize int64) SearchResultEntry {
	if len(torrents) == 0 {
		return SearchResultEntry{}
	}

	if len(torrents) == 1 {
		return torrents[0]
	}

	// Calculate scores for each torrent
	type torrentScore struct {
		torrent SearchResultEntry
		score   float64
		index   int
	}

	var scores []torrentScore

	for i, torrent := range torrents {
		// Size similarity score (closer to reference = higher score)
		torrentSize := torrent.Size
		sizeDiff := math.Abs(float64(torrentSize - referenceSize))
		maxSize := math.Max(float64(torrentSize), float64(referenceSize))
		sizeSimilarity := 1.0 - (sizeDiff / maxSize)

		// Seeders score (normalized by max seeders in the list)
		maxSeeders := 1
		for _, t := range torrents {
			if t.Seeders > maxSeeders {
				maxSeeders = t.Seeders
			}
		}
		seedersScore := float64(torrent.Seeders) / float64(maxSeeders)

		// Order score (earlier results are slightly preferred)
		orderScore := 1.0 - (float64(i)/float64(len(torrents)))*0.1

		// Combined score: size similarity (60%) + seeders (35%) + order (5%)
		totalScore := sizeSimilarity*0.6 + seedersScore*0.35 + orderScore*0.05

		scores = append(scores, torrentScore{
			torrent: torrent,
			score:   totalScore,
			index:   i,
		})
	}

	// Sort by score (highest first), then by index (lower first) as tiebreaker
	sort.Slice(scores, func(i, j int) bool {
		if math.Abs(scores[i].score-scores[j].score) < 0.001 {
			return scores[i].index < scores[j].index
		}
		return scores[i].score > scores[j].score
	})

	return scores[0].torrent
}

func downloadNextEpisode(episode EpisodeDownloadItem, referenceTorrentSize int64) error {
	// Create search parameters for this specific episode
	searchParams := FileSearchParams{
		Title: fmt.Sprintf("%s S%02dE%02d", episode.Title, episode.Season, episode.Episode),
		Year:  episode.Year,
		Type:  episode.Type,
	}

	// Search for torrents for this episode
	client, err := NewJacketClient()
	if err != nil {
		return fmt.Errorf("error initializing client: %v", err)
	}

	result, err := SearchFiles(client, searchParams)
	if err != nil || result == nil || len(result.Results) == 0 {
		return fmt.Errorf("no torrents found for episode S%02dE%02d", episode.Season, episode.Episode)
	}

	// Smart selection: prioritize torrents closest in size to the reference torrent
	// while also considering seeders count
	bestTorrent := selectBestTorrent(result.Results, referenceTorrentSize)

	// Start the torrent download
	service, err := NewTorrentService()
	if err != nil {
		return fmt.Errorf("error creating torrent service: %v", err)
	}

	// Determine save path based on content type using official OMDB title
	var savePath *string
	if episode.Type == "show" {
		// Find the show mapping to get the consistent directory name
		var dirName string
		showMappings.mutex.Lock()
		for _, mapping := range showMappings.Mappings {
			if mapping.OfficialTitle == episode.OfficialTitle {
				dirName = mapping.SafeDirName
				break
			}
		}
		showMappings.mutex.Unlock()

		// Fallback to sanitized official title if mapping not found
		if dirName == "" {
			dirName = sanitizeForFilesystem(episode.OfficialTitle)
		}

		showPath := fmt.Sprintf("/media/jellyfin/SHOWS/%s", dirName)
		savePath = &showPath
		fmt.Printf("Using save path: %s\n", showPath)
	} else if episode.Type == "movie" {
		moviePath := "/media/jellyfin/MOVIES"
		savePath = &moviePath
		fmt.Printf("Using save path for movie: %s\n", moviePath)
	}

	request := NewRequest{
		urls:     bestTorrent.MagnetUri,
		savepath: savePath,
	}

	fmt.Printf("Downloading %s (Seeders: %d)\n", bestTorrent.Title, bestTorrent.Seeders)
	StartNewTorrent(service, request)

	return nil
}

func Server() {
	// Initialize queue from file on server start
	initializeQueue()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Sailor Torrent Service - Running")
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"healthy","service":"sailor"}`)
	})

	http.HandleFunc("/create-torrent", requireAuth(func(w http.ResponseWriter, r *http.Request) {

		body, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Fprintf(w, "Error reading body: %s", err)
			return
		}

		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			fmt.Fprintf(w, "Error unmarshalling: %s", err)
			return
		}

		urlStr, ok := data["urls"].(string)
		if !ok {
			fmt.Fprintf(w, "Error reading urls")
			return
		}

		// Check if this is an episode range download
		searchParamsData, hasSearchParams := data["searchParams"]
		selectedTorrentData, hasSelectedTorrent := data["selectedTorrent"]

		if hasSearchParams && hasSelectedTorrent {
			searchParamsJSON, _ := json.Marshal(searchParamsData)
			selectedTorrentJSON, _ := json.Marshal(selectedTorrentData)

			var searchParams FileSearchParams
			var selectedTorrent map[string]interface{}

			if err := json.Unmarshal(searchParamsJSON, &searchParams); err == nil {
				if err := json.Unmarshal(selectedTorrentJSON, &selectedTorrent); err == nil {
					// Check if it's a TV show with episode range
					hasEpisodeRange := searchParams.Type == "show" &&
						(searchParams.MinSeason != nil || searchParams.MaxSeason != nil ||
							searchParams.MinEpisode != nil || searchParams.MaxEpisode != nil)

					if hasEpisodeRange {
						fmt.Printf("Episode range download detected for %s\n", searchParams.Title)

						// Get the reference torrent size
						referenceTorrentSize := int64(0)
						if sizeFloat, ok := selectedTorrent["Size"].(float64); ok {
							referenceTorrentSize = int64(sizeFloat)
						}

						// Generate episode list and add to queue
						episodes, err := generateEpisodeList(searchParams, referenceTorrentSize)
						if err != nil {
							fmt.Fprintf(w, "Error generating episode list: %s", err)
							return
						}
						fmt.Printf("Generated %d episodes to download (reference size: %d bytes)\n", len(episodes), referenceTorrentSize)

						// Add all episodes to the queue (skip the first one as we'll start it immediately)
						for i := 1; i < len(episodes); i++ {
							downloadQueue.Enqueue(episodes[i])
						}

						// Start the first episode download immediately
						if len(episodes) > 0 {
							go func() {
								err := downloadNextEpisode(episodes[0], referenceTorrentSize)
								if err != nil {
									fmt.Printf("Error starting first episode: %v\n", err)
								}
							}()
							fmt.Fprintf(w, "Episode range download started with %d episodes", len(episodes))
							return
						}
					}
				}
			}
		}

		// Regular single torrent download
		service, err := NewTorrentService()
		if err != nil {
			fmt.Fprintf(w, "Error creating client: %s", err)
			return
		}

		// Determine save path based on content type if search params are available
		var savePath *string
		if hasSearchParams {
			searchParamsJSON, _ := json.Marshal(searchParamsData)
			var searchParams FileSearchParams
			if err := json.Unmarshal(searchParamsJSON, &searchParams); err == nil {
				if searchParams.Type == "movie" {
					moviePath := "/media/jellyfin/MOVIES"
					savePath = &moviePath
					fmt.Printf("Using save path for movie: %s\n", moviePath)
				} else if searchParams.Type == "show" {
					// For shows, we need to get the official title from OMDB
					omdbClient, err := NewOMDBClient()
					if err == nil {
						if seriesResp, err := omdbClient.GetSeries(searchParams.Title); err == nil {
							showMapping := showMappings.findOrCreateMapping(searchParams.Title, seriesResp.Title, seriesResp.Year)
							showPath := fmt.Sprintf("/media/jellyfin/SHOWS/%s", showMapping.SafeDirName)
							savePath = &showPath
							fmt.Printf("Using save path for show: %s\n", showPath)
						}
					}
				}
			}
		}

		request := NewRequest{
			urls:     urlStr,
			savepath: savePath,
		}

		StartNewTorrent(service, request)
		fmt.Fprintf(w, "Single torrent download started")
	}))

	http.HandleFunc("/get-status", requireAuth(func(w http.ResponseWriter, r *http.Request) {

		service, err := NewTorrentService()
		if err == nil {
			fmt.Fprintf(w, "Error creating service")
			return
		}
		ongoingTorrent, err := InfoForOngoingTorrent(service)
		if err != nil {
			fmt.Fprintf(w, "Error searching info")
			return
		}
		percentageDone := ongoingTorrent.completed / ongoingTorrent.size
		fmt.Fprintf(w, "Percentage done: %d", percentageDone)
	}))

	http.HandleFunc("/media-info", requireAuth(func(w http.ResponseWriter, r *http.Request) {

		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		var data FileSearchParams
		if err := json.Unmarshal(body, &data); err != nil {
			return
		}

		service, err := NewOmdbService()
		if err != nil {
			fmt.Fprintf(w, "Error searching for media")
		}

		switch data.Type {
		case "movie":
			{
				response, err := service.omdbClient.GetMovieByTitle(data.Title)
				if err != nil {
					fmt.Fprintf(w, "Error getting movie: %s", err)
				}
				jsonData, _ := json.Marshal(response)

				fmt.Fprintf(w, "%s", jsonData)
			}
		case "show":
			{
				response, err := service.omdbClient.GetSeries(data.Title)
				if err != nil {
					fmt.Fprintf(w, "Error getting show: %s", err)
				}
				jsonData, _ := json.Marshal(response)

				fmt.Fprintf(w, "%s", jsonData)
			}
		}
	}))

	http.HandleFunc("/torrent-finished", requireAuth(func(w http.ResponseWriter, r *http.Request) {

		fmt.Println("Torrent finished callback received")

		// Start next episode download if queue is not empty
		if !downloadQueue.IsEmpty() {
			nextEpisode, hasNext := downloadQueue.Dequeue()
			if hasNext {
				fmt.Printf("Starting next episode download: S%dE%d of %s\n",
					nextEpisode.Season, nextEpisode.Episode, nextEpisode.Title)

				go func() {
					err := downloadNextEpisode(nextEpisode, nextEpisode.ReferenceTorrentSize)
					if err != nil {
						fmt.Printf("Error downloading next episode: %v\n", err)
					}
				}()
			}
		}

		fmt.Fprintf(w, "OK")
	}))

	http.HandleFunc("/queue-status", requireAuth(func(w http.ResponseWriter, r *http.Request) {

		downloadQueue.mutex.Lock()
		queueCopy := DownloadQueue{Items: make([]EpisodeDownloadItem, len(downloadQueue.Items))}
		copy(queueCopy.Items, downloadQueue.Items)
		downloadQueue.mutex.Unlock()

		jsonData, _ := json.Marshal(queueCopy)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s", jsonData)
	}))

	http.HandleFunc("/queue-clear", requireAuth(func(w http.ResponseWriter, r *http.Request) {

		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		downloadQueue.mutex.Lock()
		downloadQueue.Items = make([]EpisodeDownloadItem, 0)
		downloadQueue.mutex.Unlock()

		// Persist the empty queue
		if err := downloadQueue.saveToFile(); err != nil {
			fmt.Printf("Warning: Failed to save cleared queue: %v\n", err)
		}

		fmt.Fprintf(w, "Queue cleared successfully")
	}))

	http.HandleFunc("/show-mappings", requireAuth(func(w http.ResponseWriter, r *http.Request) {

		showMappings.mutex.Lock()
		mappingsCopy := ShowMappingStore{Mappings: make([]ShowMapping, len(showMappings.Mappings))}
		copy(mappingsCopy.Mappings, showMappings.Mappings)
		showMappings.mutex.Unlock()

		jsonData, _ := json.Marshal(mappingsCopy)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s", jsonData)
	}))

	http.HandleFunc("/download-request", requireAuth(func(w http.ResponseWriter, r *http.Request) {

		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}

		client, err := NewJacketClient()
		if err != nil {
			fmt.Fprintf(w, "Error initializing client")
		}
		var data FileSearchParams
		if err := json.Unmarshal(body, &data); err != nil {
			return
		}

		result, err := SearchFiles(client, data)
		if err != nil || result == nil {
			fmt.Fprintf(w, "Error searching for files: %s", err)
			return
		}
		if len(result.Results) == 0 {
			fmt.Fprintf(w, "No results found")
			return
		}

		resultList := result.Results

		// Sort by Seeders in descending order
		sort.Slice(resultList, func(i, j int) bool {
			return resultList[i].Seeders > resultList[j].Seeders
		})

		jsonData, _ := json.Marshal(resultList)

		fmt.Fprintf(w, "%s", jsonData)
	}))

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	fmt.Println("Starting server on :8089...")
	if err := http.ListenAndServe(":8089", nil); err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}

}
