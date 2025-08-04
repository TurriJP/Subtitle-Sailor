package main

type MovieResponse struct {
	Title    string `json:"Title"`
	Year     string `json:"Year"`
	ImdbID   string `json:"imdbID"`
	Type     string `json:"Type"`
	Response string `json:"Response"`
	Error    string `json:"Error,omitempty"`
	Poster   string
}

type SeriesResponse struct {
	Title        string `json:"Title"`
	Year         string `json:"Year"`
	ImdbID       string `json:"imdbID"`
	Type         string `json:"Type"`
	TotalSeasons string `json:"totalSeasons"`
	Response     string `json:"Response"`
	Error        string `json:"Error,omitempty"`
	Poster       string
}

type Episode struct {
	Title    string `json:"Title"`
	Released string `json:"Released"`
	Episode  string `json:"Episode"`
	ImdbID   string `json:"imdbID"`
}

type SeasonResponse struct {
	Title    string    `json:"Title"`
	Season   string    `json:"Season"`
	Episodes []Episode `json:"Episodes"`
	Response string    `json:"Response"`
	Error    string    `json:"Error,omitempty"`
}

type SubtitleAttributes struct {
	SubtitleID      string         `json:"subtitle_id"`
	Language        string         `json:"language"`
	DownloadCount   int            `json:"download_count"`
	HearingImpaired bool           `json:"hearing_impaired"`
	HD              bool           `json:"hd"`
	FPS             float64        `json:"fps"`
	FromTrusted     bool           `json:"from_trusted"`
	Slug            string         `json:"slug"`
	Release         string         `json:"release"`
	Comments        string         `json:"comments"`
	URL             string         `json:"url"`
	Files           []SubtitleFile `json:"files"`
}

type SubtitleFile struct {
	FileID   int    `json:"file_id"`
	CDNumber int    `json:"cd_number"`
	FileName string `json:"file_name"`
}

type SubtitleEntry struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Attributes SubtitleAttributes `json:"attributes"`
}

type SubtitleResponse struct {
	TotalPages int             `json:"total_pages"`
	TotalCount int             `json:"total_count"`
	PerPage    int             `json:"per_page"`
	Page       int             `json:"page"`
	Data       []SubtitleEntry `json:"data"`
}

type SubtitleSearchParams struct {
	Languages     string
	ImdbID        string
	MovieHash     string
	Query         string
	Year          string
	SeasonNumber  string
	EpisodeNumber string
}

type SearchResult struct {
	Results []SearchResultEntry
}

type SearchResultEntry struct {
	Title     string
	Size      int
	Guid      string
	ImdbID    string
	MagnetUri string
	Seeders   int
}

type FileSearchParams struct {
	Title   string
	Type    string
	Year    string
	Season  int
	Episode int
}
