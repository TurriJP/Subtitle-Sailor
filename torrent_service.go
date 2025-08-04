package main

import "fmt"

type TorrentService struct {
	qBittorrentClient *QbittorrentClient
}

func NewTorrentService() (*TorrentService, error) {
	client, err := NewQbittorrentClient()
	if err != nil {
		return nil, err
	}

	Authenticate(client)

	return &TorrentService{
		qBittorrentClient: client,
	}, nil
}

func InfoForOngoingTorrent(s *TorrentService) (*Torrent, error) {
	results, err := List(s.qBittorrentClient)
	if err != nil {
		return nil, err
	}

	if results != nil && len(*results) > 0 {
		first := (*results)[0]
		return &first, nil
	}

	return nil, fmt.Errorf("no torrents downloading")
}

func StartNewTorrent(s *TorrentService, params NewRequest) error {
	err := New(s.qBittorrentClient, params)
	if err != nil {
		return err
	}
	return nil
}
