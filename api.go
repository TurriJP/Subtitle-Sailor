package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
)

func enableCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func Server() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to my website!")
	})

	http.HandleFunc("/create-torrent", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}

		service, err := NewTorrentService()
		if err != nil {
			fmt.Fprintf(w, "Error creating client: %s", err)
			return
		}

		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			fmt.Fprintf(w, "Error unmarshalling: %s", err)
			return
		}
		urlStr, ok := data["urls"].(string)
		if !ok {
			// handle error
			fmt.Fprintf(w, "Error reading urls")
			return
		}

		request := NewRequest{
			urls:     urlStr,
			savepath: nil,
		}

		StartNewTorrent(service, request)
	})

	http.HandleFunc("/get-status", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		
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
	})

	http.HandleFunc("/media-info", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		
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
	})

	http.HandleFunc("/download-request", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		
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
	})

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	fmt.Println("Starting server on :8089...")
	if err := http.ListenAndServe(":8089", nil); err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}

}
